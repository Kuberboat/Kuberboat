package proxy

type MetaManager interface {
	AddServiceClusterIP(serviceName string, clusterIP string)
	AddServiceChain(serviceName string, chain *ServiceChain)
	AddPodChain(serviceChainName string, chain *PodChain)

	GetClusterIP(serviceName string) string
	GetServiceChains(serviceName string) []*ServiceChain
	GetPodChains(serviceChainName string) []*PodChain

	DeleteServiceClusterIP(serviceName string)
	DeleteServiceChains(serviceName string)
	DeletePodChains(serviceChainName string)

	ServiceExists(serviceName string) bool
}

type basicManager struct {
	// serviceToClusterIP is a map from service name to its cluster IP.
	serviceToClusterIP map[string]string
	// serviceChains is a map from service name to the iptables chains of the service. Since a
	// service might have multiple port mappings, it could have multiple iptables chains.
	serviceChains map[string][]*ServiceChain
	// podChains is a map from service chain name to the pod iptables chains that the service
	// chain could jump to.
	podChains map[string][]*PodChain
}

func NewMetaManager() MetaManager {
	return &basicManager{
		serviceToClusterIP: map[string]string{},
		serviceChains:      map[string][]*ServiceChain{},
		podChains:          map[string][]*PodChain{},
	}
}

func (bm *basicManager) AddServiceClusterIP(serviceName string, clusterIP string) {
	bm.serviceToClusterIP[serviceName] = clusterIP
}

func (bm *basicManager) AddServiceChain(serviceName string, chain *ServiceChain) {
	bm.serviceChains[serviceName] = append(bm.serviceChains[serviceName], chain)
}

func (bm *basicManager) AddPodChain(serviceChainName string, chain *PodChain) {
	bm.podChains[serviceChainName] = append(bm.podChains[serviceChainName], chain)
}

func (bm *basicManager) GetClusterIP(serviceName string) string {
	return bm.serviceToClusterIP[serviceName]
}

func (bm *basicManager) GetServiceChains(serviceName string) []*ServiceChain {
	return bm.serviceChains[serviceName]
}

func (bm *basicManager) GetPodChains(serviceChainName string) []*PodChain {
	return bm.podChains[serviceChainName]
}

func (bm *basicManager) DeleteServiceClusterIP(serviceName string) {
	delete(bm.serviceToClusterIP, serviceName)
}

func (bm *basicManager) DeleteServiceChains(serviceName string) {
	delete(bm.serviceChains, serviceName)
}

func (bm *basicManager) DeletePodChains(serviceChainName string) {
	delete(bm.podChains, serviceChainName)
}

func (bm *basicManager) ServiceExists(serviceName string) bool {
	_, ok := bm.serviceToClusterIP[serviceName]
	return ok
}
