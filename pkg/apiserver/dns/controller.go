package dns

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/golang/glog"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver"
	"p9t.io/kuberboat/pkg/apiserver/etcd"
)

const (
	nginxConfigRelativeDir = "/.kube/nginx/"
	// Relative path to home dir of nginx config file.
	nginxConfigFileName = "kubedns.conf"
	// Container name of nginx.
	nginxContainerName = "kuberboat-nginx"
	// Key of nginx IP on etcd.
	nginxIPKey = "/ip/nginx"
	// etcdDNSPrefix is the key prefix of domain name.
	etcdDNSPrefix = "/dns"
	// Indent is four spaces.
	indent = "    "
)

type Controller interface {
	// GetDNSs returns information about DNSs specified by dnsName.
	// Return value is composed of DNSs that are found and DNs names that do not exist.
	GetDNSs(all bool, dnsNames []string) ([]*core.DNS, []string)
	// CreateDNS applies a DNS configuration to nginx and coredns.
	// It will not override existing DNS configurations.
	CreateDNS(*core.DNS) error
	// DeleteDNS deletes a DNS configuration indexed by name.
	DeleteDNSByName(name string)
}

type basicController struct {
	mtx              sync.Mutex
	componentManager apiserver.ComponentManager
	nginxConfigDir   string
	nginxIP          string
}

type location struct {
	dnsName   string
	path      string
	proxyPass string
}

type coreDNSEntry struct {
	Host string `json:"host"`
}

func NewDNSController(componentManager apiserver.ComponentManager) Controller {
	bytes, err := etcd.GetRaw(nginxIPKey)
	if err != nil {
		glog.Fatal(err)
	}
	nginxIP := string(bytes)
	if net.ParseIP(nginxIP) == nil {
		glog.Errorf("got invalid nginx IP from etcd: %v, DNS might not work", nginxIP)
	} else {
		glog.Infof("DNS: got nginx IP %v", nginxIP)
	}
	homedir, err := os.UserHomeDir()
	if err != nil {
		glog.Fatal(err)
	}
	configDir := path.Join(homedir, nginxConfigRelativeDir)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		os.Mkdir(configDir, 0644)
	}

	return &basicController{
		componentManager: componentManager,
		nginxConfigDir:   configDir,
		nginxIP:          nginxIP,
	}
}

func (c *basicController) GetDNSs(all bool, dnsNames []string) ([]*core.DNS, []string) {
	if all {
		return c.componentManager.ListDNS(), make([]string, 0)
	} else {
		found := make([]*core.DNS, 0)
		notFound := make([]string, 0)
		for _, name := range dnsNames {
			if !c.componentManager.DNSExistsByName(name) {
				notFound = append(notFound, name)
			} else {
				dns := c.componentManager.GetDNSByName(name)
				if dns == nil {
					glog.Errorf("dns missing event if cm claims otherwise")
					continue
				}
				found = append(found, dns)
			}
		}
		return found, notFound
	}
}

func (c *basicController) CreateDNS(newDNS *core.DNS) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	host2location := make(map[string][]*location)

	var isNewHost bool = true

	dnss := c.componentManager.ListDNS()

	// Check a few things:
	//   1. If there is any existing path that is the prefix of any new paths or vice versa.
	//   2. If the host name is new.
	//   3. If any service name or port is non-existent.
	for _, dns := range dnss {
		for _, mapping1 := range dns.Spec.Paths {
			location, err := c.validatePath(dns.Name, &mapping1)
			if err != nil {
				return err
			}
			host2location[dns.Spec.Host] = append(host2location[dns.Spec.Host], location)
			if dns.Spec.Host == newDNS.Spec.Host {
				isNewHost = false
				for _, mapping2 := range newDNS.Spec.Paths {
					if strings.HasPrefix(mapping1.Path, mapping2.Path) ||
						strings.HasPrefix(mapping2.Path, mapping1.Path) {
						return fmt.Errorf("path %v conflits with existing path %v", mapping2.Path, mapping1.Path)
					}
				}
			}

		}
	}
	for _, mapping := range newDNS.Spec.Paths {
		location, err := c.validatePath(newDNS.Name, &mapping)
		if err != nil {
			return err
		}
		host2location[newDNS.Spec.Host] = append(host2location[newDNS.Spec.Host], location)
	}

	// If the host is new, add a new dns entry.
	// Sadly, no rollback for this.
	if isNewHost {
		etcdKey, err := host2CoreDNSPath(newDNS.Spec.Host)
		if err != nil {
			return err
		}
		etcd.Put(etcdKey, coreDNSEntry{Host: c.nginxIP})
	}

	// Generate nginx configration file.
	if err := c.generateNginxConf(host2location); err != nil {
		return err
	}

	// Reload nginx configuration.
	// docker exec with SDK is too troublesome. Twenty lines of code for one simple command.
	cmd := exec.Command("/usr/bin/docker", "exec", nginxContainerName, "nginx", "-s", "reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload nginx config: %v", err.Error())
	}

	// Update metadata.
	newDNS.Status.Applied = true
	c.componentManager.SetDNS(newDNS)

	glog.Infof("DNS [%v]: dns created", newDNS.Name)

	return nil
}

func (c *basicController) DeleteDNSByName(name string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
}

func (c *basicController) validatePath(dnsName string, mapping *core.PathMapping) (*location, error) {
	if !c.componentManager.ServiceExistsByName(mapping.ServiceName) {
		return nil, fmt.Errorf("service does not exist: %v", mapping.ServiceName)
	}
	service := c.componentManager.GetServiceByName(mapping.ServiceName)
	if service == nil {
		return nil, fmt.Errorf("service does not exist: %v", mapping.ServiceName)
	}
	serviceIP := service.Spec.ClusterIP
	if net.ParseIP(serviceIP) == nil {
		return nil, fmt.Errorf("service %v does not have valid cluster IP: %v", service.Name, serviceIP)
	}
	return &location{
		dnsName:   dnsName,
		path:      mapping.Path,
		proxyPass: fmt.Sprintf("http://%v:%v/", serviceIP, mapping.ServicePort),
	}, nil
}

func (c *basicController) generateNginxConf(host2location map[string][]*location) error {
	// Generate nginx config file.
	file, err := os.OpenFile(path.Join(c.nginxConfigDir, nginxConfigFileName), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	// First server block does not need a leading '\n'
	var firstServer bool = true
	// Each kv pair in host2location correspond to a server block.
	for host, locations := range host2location {
		if firstServer {
			firstServer = false
		} else {
			writeIndent(file, 0, "\n")
		}

		writeIndent(file, 0, "server {\n")
		writeIndent(file, 1, "listen 80;\n")
		writeIndent(file, 1, "listen [::]:80;\n")
		writeIndent(file, 1, fmt.Sprintf("server_name %v;\n", host))

		for _, location := range locations {
			// Write location to match paths not ending with slash.
			writeLocation(file, location, true)

			// Write location to match paths ending with slash and other subpaths.
			writeLocation(file, location, false)
		}

		writeIndent(file, 0, "}\n")
	}

	err = file.Close()
	if err != nil {
		return err
	}

	return nil
}

func writeIndent(file *os.File, indents int, str string) (int, error) {
	var bytesWritten int = 0
	for i := 0; i < indents; i++ {
		n, err := file.WriteString(indent)
		if err != nil {
			return bytesWritten, err
		}
		bytesWritten += n
	}
	n, err := file.WriteString(str)
	if err != nil {
		return bytesWritten, err
	}
	bytesWritten += n
	return bytesWritten, nil
}

func writeLocation(file *os.File, location *location, exact bool) {
	writeIndent(file, 0, "\n")
	writeIndent(file, 1, fmt.Sprintf("# DNS config: %v\n", location.dnsName))
	if exact {
		writeIndent(file, 1, fmt.Sprintf("location = %v {\n", location.path))
	} else {
		writeIndent(file, 1, fmt.Sprintf("location ^~ %v/ {\n", location.path))
	}
	writeIndent(file, 2, fmt.Sprintf("proxy_pass %v;\n", location.proxyPass))
	writeIndent(file, 1, "}\n")
}

func host2CoreDNSPath(host string) (string, error) {
	var builder strings.Builder
	_, err := builder.WriteString(etcdDNSPrefix)
	if err != nil {
		return "", err
	}
	domains := strings.Split(host, ".")
	for i := len(domains) - 1; i >= 0; i-- {
		domain := domains[i]
		_, err := builder.WriteString(fmt.Sprintf("/%v", domain))
		if err != nil {
			return "", err
		}
	}
	return builder.String(), nil
}

// TODO: When deleting a service, check if there is a dns configuration pointing to it.
