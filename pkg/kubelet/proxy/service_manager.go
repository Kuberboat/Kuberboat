package proxy

import (
	"fmt"

	"github.com/golang/glog"
	kuberetcd "p9t.io/kuberboat/pkg/apiserver/etcd"
	"p9t.io/kuberboat/pkg/kubelet/proxy/types"
)

type MetaManager interface {
	AddServiceClusterIP(serviceName string, clusterIP string, isRecover bool)
	AddServiceChain(serviceName string, chain *types.ServiceChain, isRecover bool)
	AddPodChain(serviceChainName string, chain *types.PodChain, isRecover bool)

	GetClusterIP(serviceName string) string
	GetServiceChains(serviceName string) []*types.ServiceChain
	GetPodChains(serviceChainName string) []*types.PodChain

	DeleteServiceClusterIP(serviceName string)
	DeleteServiceChains(serviceName string)
	DeletePodChains(serviceChainName string)
	DeletePodChainFromServiceChain(podName string, serviceChainName string)

	ServiceExists(serviceName string) bool
}

type basicManager struct {
	// serviceToClusterIP is a map from service name to its cluster IP.
	serviceToClusterIP map[string]string
	// serviceChains is a map from service name to the iptables chains of the service. Since a
	// service might have multiple port mappings, it could have multiple iptables chains.
	serviceChains map[string][]*types.ServiceChain
	// podChains is a map from service chain name to the pod iptables chains that the service
	// chain could jump to.
	podChains map[string][]*types.PodChain
}

func NewMetaManager() MetaManager {
	return &basicManager{
		serviceToClusterIP: map[string]string{},
		serviceChains:      map[string][]*types.ServiceChain{},
		podChains:          map[string][]*types.PodChain{},
	}
}

func (bm *basicManager) AddServiceClusterIP(serviceName string, clusterIP string, isRecover bool) {
	bm.serviceToClusterIP[serviceName] = clusterIP
	if !isRecover {
		go func() {
			if err := kuberetcd.Put(fmt.Sprintf("/Kubeproxy/Service/ClusterIP/%v", serviceName), types.ServiceNameWithClusterIP{ServiceName: serviceName, ClusterIP: clusterIP}); err != nil {
				glog.Errorf("persist service clusterIP error: %v", err)
			}
		}()
	}
}

func (bm *basicManager) AddServiceChain(serviceName string, chain *types.ServiceChain, isRecover bool) {
	bm.serviceChains[serviceName] = append(bm.serviceChains[serviceName], chain)
	if !isRecover {
		go func() {
			if err := kuberetcd.Put(fmt.Sprintf("/Kubeproxy/Service/ServiceChain/%v", serviceName), bm.serviceChains[serviceName]); err != nil {
				glog.Errorf("persist service chain error: %v", err)
			}
		}()
	}
}

func (bm *basicManager) AddPodChain(serviceChainName string, chain *types.PodChain, isRecover bool) {
	bm.podChains[serviceChainName] = append(bm.podChains[serviceChainName], chain)
	if !isRecover {
		go func() {
			if err := kuberetcd.Put(fmt.Sprintf("/Kubeproxy/ServiceChain/%v", serviceChainName), bm.podChains[serviceChainName]); err != nil {
				glog.Errorf("persist service podChain error: %v", err)
			}
		}()
	}
}

func (bm *basicManager) GetClusterIP(serviceName string) string {
	return bm.serviceToClusterIP[serviceName]
}

func (bm *basicManager) GetServiceChains(serviceName string) []*types.ServiceChain {
	return bm.serviceChains[serviceName]
}

func (bm *basicManager) GetPodChains(serviceChainName string) []*types.PodChain {
	return bm.podChains[serviceChainName]
}

func (bm *basicManager) DeleteServiceClusterIP(serviceName string) {
	delete(bm.serviceToClusterIP, serviceName)
	go func() {
		if err := kuberetcd.Delete(fmt.Sprintf("/Kubeproxy/Service/ClusterIP/%v", serviceName)); err != nil {
			glog.Errorf("delete clusterIP error: %v", err)
		}
	}()
}

func (bm *basicManager) DeleteServiceChains(serviceName string) {
	delete(bm.serviceChains, serviceName)
	go func() {
		if err := kuberetcd.Delete(fmt.Sprintf("/Kubeproxy/Service/ServiceChain/%v", serviceName)); err != nil {
			glog.Errorf("delete service chain error: %v", err)
		}
	}()
}

func (bm *basicManager) DeletePodChains(serviceChainName string) {
	delete(bm.podChains, serviceChainName)
	go func() {
		if err := kuberetcd.Delete(fmt.Sprintf("/Kubeproxy/ServiceChain/%v", serviceChainName)); err != nil {
			glog.Errorf("delete service podChain error: %v", err)
		}
	}()
}

func (bm *basicManager) DeletePodChainFromServiceChain(podName string, serviceChainName string) {
	podChains := bm.podChains[serviceChainName]
	for i, podChain := range podChains {
		if podChain.PodName == podName {
			podChains = append(podChains[:i], podChains[i+1:]...)
			break
		}
	}
	bm.podChains[serviceChainName] = podChains
}

func (bm *basicManager) ServiceExists(serviceName string) bool {
	_, ok := bm.serviceToClusterIP[serviceName]
	return ok
}
