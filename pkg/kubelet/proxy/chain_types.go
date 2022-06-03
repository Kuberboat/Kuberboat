package proxy

import "p9t.io/kuberboat/pkg/api/core"

// PodChain contains the name of an iptables chain for a pod, as well as pod name and pod IP.
type PodChain struct {
	// ChainName is the name of the iptables chain.
	ChainName string
	// PodName is the name of the pod.
	PodName string
	// PodIP is the ip of the pod.
	PodIP string
}

// ServiceChain contains the name of an iptables chain for a service, as well as the mapping
// from service port to pod port. Each mapping will have its own service iptables chain.
type ServiceChain struct {
	// ChainName is the name of the iptables chain.
	ChainName string
	// ServicePort is the mapping from service port to pod port.
	ServicePort *core.ServicePort
}
