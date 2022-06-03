package kubelet

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/golang/glog"

	"p9t.io/kuberboat/pkg/api/core"
	kubeletpod "p9t.io/kuberboat/pkg/kubelet/pod"
	kubeproxy "p9t.io/kuberboat/pkg/kubelet/proxy"
)

const (
	FlannelNetInterfaceName string = "flannel.1"
	Ens3NetInterfaceName    string = "ens3"
	Eth0NetInterfaceName    string = "eth0"
)

// KubeProxy manages the communication between pods via service.
type KubeProxy interface {
	// CreateService creates a service by modifying the kernel iptables.
	CreateService(
		serviceName string, clusterIP string, servicePorts []*core.ServicePort,
		podNames []string, podIPs []string) error
	// DeleteService deletes a service by removing relevant chains and rules in kernel iptables.
	DeleteService(serviceName string) error
	// AddPodToServices adds a pod to an existing service by adding the rules in kernel iptables.
	AddPodToServices(serviceNames []string, podName string, podIP string) error
	// DeletePodFromService deletes a pod from an existing service by rewriting all the rules.
	DeletePodFromServices(serviceNames []string, podName string) error
}

type kubeProxyInner struct {
	// mtx ensures concurrent access to inner data structures are safe.
	mtx sync.Mutex
	// podMetaManager manages pod metadata.
	podMetaManager kubeletpod.MetaManager
	// serviceMetaManager maintains the metadata for created services.
	serviceMetaManager kubeproxy.MetaManager
	// iptablesClient provides APIs to manage kernel iptables for service.
	iptablesClient kubeproxy.IPTablesClient
}

func NewKubeProxy(podMetaManager kubeletpod.MetaManager) KubeProxy {
	hostInetIP := getNetInterfaceIpv4Addr(Ens3NetInterfaceName)
	if hostInetIP == "" {
		hostInetIP = getNetInterfaceIpv4Addr(Eth0NetInterfaceName)
		if hostInetIP == "" {
			glog.Fatalf("cannot get ipv4 address for the host")
		}
	}
	flannelIP := getNetInterfaceIpv4Addr(FlannelNetInterfaceName)
	if flannelIP == "" {
		glog.Fatalf("cannot get ipv4 address for %s", FlannelNetInterfaceName)
	}
	cli, err := kubeproxy.NewIptablesClient(hostInetIP, flannelIP)
	if err != nil {
		glog.Fatal(err)
	}
	err = cli.InitServiceIPTables()
	if err != nil {
		glog.Fatal(err)
	}
	return &kubeProxyInner{
		mtx:                sync.Mutex{},
		podMetaManager:     podMetaManager,
		serviceMetaManager: kubeproxy.NewMetaManager(),
		iptablesClient:     cli,
	}
}

func (kp *kubeProxyInner) CreateService(
	serviceName string,
	clusterIP string,
	servicePorts []*core.ServicePort,
	podNames []string,
	podIPs []string,
) error {
	kp.mtx.Lock()
	defer kp.mtx.Unlock()

	if kp.serviceMetaManager.ServiceExists(serviceName) {
		return fmt.Errorf("service %s already exists", serviceName)
	}

	for _, servicePort := range servicePorts {
		// Create an iptables chain for each port mapping.
		serviceChainName := kp.iptablesClient.CreateServiceChain()

		for i := range podNames {
			podName := podNames[i]
			podIP := podIPs[i]

			// Create an iptables chain for each pod in the service.
			podChainName := kp.iptablesClient.CreatePodChain()

			// Add a jump-to-mark rule and a DNAT rule to the chain.
			_, exist := kp.podMetaManager.PodByName(podName)
			err := kp.iptablesClient.ApplyPodChainRules(podChainName, podIP, servicePort.TargetPort, exist)
			if err != nil {
				return err
			}

			// Add a rule that jumps to the chain in the service iptables chain.
			err = kp.iptablesClient.ApplyPodChain(serviceName, serviceChainName, podName, podChainName, i+1)
			if err != nil {
				return err
			}

			// Update metadata.
			podChain := kubeproxy.PodChain{
				ChainName: podChainName,
				PodName:   podName,
				PodIP:     podIP,
			}
			kp.serviceMetaManager.AddPodChain(serviceChainName, &podChain)
		}

		// Add a rule that jumps to the service chain when the destination of a packet is <clusterIP>:<port>.
		kp.iptablesClient.ApplyServiceChain(serviceName, clusterIP, serviceChainName, servicePort.Port)

		// Update metadata.
		serviceChain := kubeproxy.ServiceChain{
			ChainName:   serviceChainName,
			ServicePort: servicePort,
		}
		kp.serviceMetaManager.AddServiceChain(serviceName, &serviceChain)
	}

	kp.serviceMetaManager.AddServiceClusterIP(serviceName, clusterIP)

	return nil
}

func (kp *kubeProxyInner) DeleteService(serviceName string) error {
	kp.mtx.Lock()
	defer kp.mtx.Unlock()

	if !kp.serviceMetaManager.ServiceExists(serviceName) {
		return fmt.Errorf("no such service: %s", serviceName)
	}

	clusterIP := kp.serviceMetaManager.GetClusterIP(serviceName)
	serviceChains := kp.serviceMetaManager.GetServiceChains(serviceName)

	for _, serviceChain := range serviceChains {
		// Delete the service chain and its rules.
		kp.iptablesClient.DeleteServiceChain(
			serviceName, clusterIP, serviceChain.ChainName, serviceChain.ServicePort.Port)

		// Delete the pod chains and their rules.
		podChains := kp.serviceMetaManager.GetPodChains(serviceChain.ChainName)
		for _, podChain := range podChains {
			err := kp.iptablesClient.DeletePodChain(podChain.PodName, podChain.ChainName)
			if err != nil {
				glog.Error(err)
			}
		}

		// Update metadata.
		kp.serviceMetaManager.DeletePodChains(serviceChain.ChainName)
	}

	// Update metadata.
	kp.serviceMetaManager.DeleteServiceChains(serviceName)
	kp.serviceMetaManager.DeleteServiceClusterIP(serviceName)

	return nil
}

func (kp *kubeProxyInner) AddPodToServices(serviceNames []string, podName string, podIP string) error {
	kp.mtx.Lock()
	defer kp.mtx.Unlock()

	var err error = nil
	notExistServices := make([]string, 0)

	for _, serviceName := range serviceNames {
		if !kp.serviceMetaManager.ServiceExists(serviceName) {
			notExistServices = append(notExistServices, serviceName)
			continue
		}

		serviceChains := kp.serviceMetaManager.GetServiceChains(serviceName)
		for _, serviceChain := range serviceChains {
			podChains := kp.serviceMetaManager.GetPodChains(serviceChain.ChainName)
			chainNum := len(podChains)

			// Create an iptables chain for the new pod in the service.
			podChainName := kp.iptablesClient.CreatePodChain()

			// Add a jump-to-mark rule and a DNAT rule to the chain.
			_, exist := kp.podMetaManager.PodByName(podName)
			err = kp.iptablesClient.ApplyPodChainRules(
				podChainName,
				podIP,
				serviceChain.ServicePort.TargetPort,
				exist,
			)
			if err != nil {
				continue
			}

			// Add a rule that jumps to the chain in the service iptables chain.
			err = kp.iptablesClient.ApplyPodChain(
				serviceName,
				serviceChain.ChainName,
				podName,
				podChainName,
				chainNum+1,
			)
			if err != nil {
				continue
			}

			// Update metadata.
			podChain := kubeproxy.PodChain{
				ChainName: podChainName,
				PodName:   podName,
				PodIP:     podIP,
			}
			kp.serviceMetaManager.AddPodChain(serviceChain.ChainName, &podChain)
		}
	}

	if len(notExistServices) != 0 {
		err = fmt.Errorf("no such service to add pod to: %v", notExistServices)
	}

	return err
}

func (kp *kubeProxyInner) DeletePodFromServices(serviceNames []string, podName string) error {
	kp.mtx.Lock()
	defer kp.mtx.Unlock()

	var err error = nil
	notExistServices := make([]string, 0)

	for _, serviceName := range serviceNames {
		if !kp.serviceMetaManager.ServiceExists(serviceName) {
			notExistServices = append(notExistServices, serviceName)
			continue
		}

		serviceChains := kp.serviceMetaManager.GetServiceChains(serviceName)
		for _, serviceChain := range serviceChains {
			// Clear the service chain in order to reassign round robin number for the rest pods
			kp.iptablesClient.ClearServiceChain(serviceName, serviceChain.ChainName)

			podChains := kp.serviceMetaManager.GetPodChains(serviceChain.ChainName)
			roundRobin := 0
			for _, podChain := range podChains {
				if podChain.PodName == podName {
					// Remove the chain of the deleted pod
					kp.iptablesClient.DeletePodChain(podChain.PodName, podChain.ChainName)
				} else {
					// Add a rule that jumps to the chain in the service iptables chain.
					roundRobin++
					err = kp.iptablesClient.ApplyPodChain(
						serviceName,
						serviceChain.ChainName,
						podChain.PodName,
						podChain.ChainName,
						roundRobin,
					)
				}
			}

			// Update metadata
			kp.serviceMetaManager.DeletePodChainFromServiceChain(podName, serviceChain.ChainName)
		}
	}

	if len(notExistServices) != 0 {
		err = fmt.Errorf("no such service to delete pod from: %v", notExistServices)
	}

	return err
}

// getNetInterfaceIpv4Addr gets the IPV4 address of given net interface.
func getNetInterfaceIpv4Addr(interfaceName string) string {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Name == interfaceName {
			addrs, err := iface.Addrs()
			if err != nil {
				return ""
			}
			for _, addr := range addrs {
				addrStr := addr.String()
				split := strings.Split(addrStr, "/")
				addrStr0 := split[0]
				ip := net.ParseIP(addrStr0)
				if ip.To4() != nil {
					return ip.To4().String()
				}
			}
		}
	}
	return ""
}
