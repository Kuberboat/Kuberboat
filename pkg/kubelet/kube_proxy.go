package kubelet

import (
	"fmt"
	"sync"

	"github.com/golang/glog"

	"p9t.io/kuberboat/pkg/api/core"
	kubeproxy "p9t.io/kuberboat/pkg/kubelet/proxy"
)

// KubeProxy manages the communication between pods via service.
type KubeProxy interface {
	// CreateService creates a service by modifying the kernel iptables.
	CreateService(
		serviceName string, clusterIP string, servicePorts []*core.ServicePort,
		podNames []string, podIPs []string) error
	// DeleteService deletes a service by removing relevant chains and rules in kernel iptables.
	DeleteService(serviceName string) error
}

type kubeProxyInner struct {
	// mtx ensures concurrent access to inner data structures are safe.
	mtx sync.Mutex
	// serviceMetaManager maintains the metadata for created services.
	serviceMetaManager kubeproxy.MetaManager
	// iptablesClient provides APIs to manage kernel iptables for service.
	iptablesClient kubeproxy.IPTablesClient
}

var kubeProxy KubeProxy

func ProxyInstance() KubeProxy {
	if kubeProxy == nil {
		kubeProxy = newKubeProxy()
	}
	return kubeProxy
}

func newKubeProxy() KubeProxy {
	cli, err := kubeproxy.NewIptablesClient()
	if err != nil {
		glog.Fatal(err)
	}
	err = cli.InitServiceIPTables()
	if err != nil {
		glog.Fatal(err)
	}
	return &kubeProxyInner{
		mtx:                sync.Mutex{},
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

	if kp.serviceMetaManager.ServiceExists(serviceName) {
		kp.mtx.Unlock()
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

			// Add a DNAT rule to the chain.
			err := kp.iptablesClient.ApplyPodChainRules(podChainName, podIP, servicePort.TargetPort)
			if err != nil {
				kp.mtx.Unlock()
				return err
			}

			// Add a rule that jumps to the chain in the service iptables chain.
			err = kp.iptablesClient.ApplyPodChain(serviceName, serviceChainName, podName, podChainName, i+1)
			if err != nil {
				kp.mtx.Unlock()
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

	kp.mtx.Unlock()
	return nil
}

func (kp *kubeProxyInner) DeleteService(serviceName string) error {
	kp.mtx.Lock()

	if !kp.serviceMetaManager.ServiceExists(serviceName) {
		kp.mtx.Unlock()
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
				kp.mtx.Unlock()
				return err
			}
		}

		// Update metadata.
		kp.serviceMetaManager.DeletePodChains(serviceChain.ChainName)
	}

	// Update metadata.
	kp.serviceMetaManager.DeleteServiceChains(serviceName)
	kp.serviceMetaManager.DeleteServiceClusterIP(serviceName)

	kp.mtx.Unlock()
	return nil
}
