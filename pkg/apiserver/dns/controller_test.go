package dns

import (
	"container/list"
	"testing"

	"github.com/stretchr/testify/assert"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver"
)

var testDNSs = []*core.DNS{
	{
		Kind: core.DNSType,
		ObjectMeta: core.ObjectMeta{
			Name: "dns-1",
		},
		Spec: core.DNSSpec{
			Host: "test.com",
			Paths: []core.PathMapping{
				{
					Path:        "/aaa",
					ServiceName: "svc-1",
					ServicePort: 2000,
				},
				{
					Path:        "/bbb",
					ServiceName: "svc-2",
					ServicePort: 2000,
				},
			},
		},
	},
	{
		Kind: core.DNSType,
		ObjectMeta: core.ObjectMeta{
			Name: "dns-2",
		},
		Spec: core.DNSSpec{
			Host: "test.com",
			Paths: []core.PathMapping{
				{
					Path:        "/ccc",
					ServiceName: "svc-1",
					ServicePort: 3000,
				},
				{
					Path:        "/ddd",
					ServiceName: "svc-2",
					ServicePort: 3000,
				},
			},
		},
	},
	{
		Kind: core.DNSType,
		ObjectMeta: core.ObjectMeta{
			Name: "dns-3",
		},
		Spec: core.DNSSpec{
			Host: "example.com",
			Paths: []core.PathMapping{
				{
					Path:        "/ccc",
					ServiceName: "svc-1",
					ServicePort: 3000,
				},
				{
					Path:        "/ddd",
					ServiceName: "svc-2",
					ServicePort: 3000,
				},
			},
		},
	},
}

var testServices = []*core.Service{
	{
		Kind: core.ServiceType,
		ObjectMeta: core.ObjectMeta{
			Name: "svc-1",
		},
		Spec: core.ServiceSpec{
			ClusterIP: "240.0.0.1",
		},
	},
	{
		Kind: core.ServiceType,
		ObjectMeta: core.ObjectMeta{
			Name: "svc-2",
		},
		Spec: core.ServiceSpec{
			ClusterIP: "240.0.0.2",
		},
	},
}

func TestGenerateNginxConfig(t *testing.T) {
	componentManager := apiserver.NewComponentManager()
	dnsController := &basicController{
		componentManager: componentManager,
		nginxConfigDir:   ".",
	}
	for _, service := range testServices {
		componentManager.SetService(service, list.New())
	}

	host2locations := map[string][]*location{}
	for _, dns := range testDNSs {
		for _, mapping := range dns.Spec.Paths {
			location, err := dnsController.validatePath(dns.Name, &mapping)
			assert.Nil(t, err)
			host2locations[dns.Spec.Host] = append(host2locations[dns.Spec.Host], location)
		}

	}

	err := dnsController.generateNginxConf(host2locations)
	assert.Nil(t, err)
}
