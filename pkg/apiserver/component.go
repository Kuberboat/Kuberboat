package apiserver

import (
	"container/list"
	"reflect"
	"sync"

	"p9t.io/kuberboat/pkg/api/core"
)

// ComponentManager serves as a cache for pods, services and deployments of the cluster in
// API Server. All the operations to ComponentManager are thread safe.
type ComponentManager interface {
	// SetPod sets a pod into ComponentManager. This function will not check the existence of the
	// pod. To check for existence, you should call `PodExistsByName`.
	SetPod(pod *core.Pod)
	// DeletePodByName deletes a pod by name from ComponentManager. This function will not check
	// the existence of the pod.
	DeletePodByName(name string)
	// GetPodByName gets a pod from ComponentManager by name.
	GetPodByName(name string) *core.Pod
	// PodExistsByName checks whether a pod of a specific name exists.
	PodExistsByName(name string) bool
	// ListPods lists all the pods present.
	ListPods() []*core.Pod
	// ListPodsByPhase lists all pods whose phases match exactly with the given phase.
	ListPodsByPhase(phase core.PodPhase) []*core.Pod
	// ListPodsByLabels lists all pods whose labels and phases match exactly with the given ones.
	ListPodsByLabelsAndPhase(labels *map[string]string, phase core.PodPhase) *list.List

	// SetDeployment sets a deployment and the pods it creates into ComponentManager. This
	// function will not check the existence of the deployment.
	SetDeployment(deployment *core.Deployment, pods *list.List)
	// DeleteDeploymentByName deletes a deployment by its name as well as all of the pods it creates
	// from ComponentManager. This function will not check the existence of the deployment.
	DeleteDeploymentByName(deploymentName string)
	// GetDeploymentByName gets a deployment from ComponentManager by name.
	GetDeploymentByName(name string) *core.Deployment
	// DeploymentExistsByName checks whether a deployment of a specific name exists.
	DeploymentExistsByName(name string) bool
	// ListDeployments lists all the deployments present.
	ListDeployments() []*core.Deployment
	// ListPodsByDeployment lists all the pods given the name of a deployment. This function will not
	// check the existence of the deployment. If the deployment does not exist, an empty array will be
	// returned.
	ListPodsByDeploymentName(deploymentName string) *list.List
	// GetDeploymentByPod gets the deployment a pod belongs to by the name of the pod. This function will not
	// check the existence of the pod. If the pod does not belong to any deployment, the function will return
	// nil.
	GetDeploymentByPodName(podName string) *core.Deployment

	// SetService sets a pod into ComponentManager. This function will not check the existence of the
	// service. To check for existence, you should call `ServiceExistsByName`.
	SetService(service *core.Service, pods *list.List)
	// DeleteServiceByName deletes a service by name from ComponentManager. This function will not check
	// the existence of the service.
	DeleteServiceByName(name string)
	// GetServiceByName gets a service from ComponentManager by name.
	GetServiceByName(name string) *core.Service
	// ServiceExistsByName checks whether a service of a specific name exists.
	ServiceExistsByName(name string) bool
	// ListServices lists all the services present.
	ListServices() []*core.Service
	// ListPodsByServiceName lists all the pods given the name of a service. This function will not
	// check the existence of the service. If the service does not exist, an empty array will be
	// returned.
	ListPodsByServiceName(serviceName string) *list.List

	// SetDNS sets a DNS configuration into ComponentManager. This function will not check the existence of the
	// DNS. To check for existence, you should call `DNSExistsByName`.
	SetDNS(dns *core.DNS)
	// DNSExistsByName checks whether a DNS of a specific name exists.
	DNSExistsByName(name string) bool
	// DeleteDNSByName deletes a DNS by name from ComponentManager. This function will not check
	// the existence of the DNS.
	DeleteDNSByName(name string)
	// GetDNSByName gets a DNS from ComponentManager by name.
	GetDNSByName(name string) *core.DNS
	// ListDNS lists all the DNS configurations present.
	ListDNS() []*core.DNS
}

type componentManagerInner struct {
	mtx sync.RWMutex
	// Stores the mapping from pod name to pod.
	pods map[string]*core.Pod
	// Stores the mapping from service name to service.
	services map[string]*core.Service
	// Stores the mapping from service name to service.
	deployments map[string]*core.Deployment
	// Stores the mapping from DNS name to DNS.
	dns map[string]*core.DNS
	// Stores the mapping from the name of a deployment to the pods it creates.
	deploymentToPods map[string]*list.List
	// Stores the mapping from the name of a service to the pods it selects by label.
	servicesToPods map[string]*list.List
}

func NewComponentManager() ComponentManager {
	return &componentManagerInner{
		mtx:              sync.RWMutex{},
		pods:             map[string]*core.Pod{},
		services:         map[string]*core.Service{},
		deployments:      map[string]*core.Deployment{},
		dns:              map[string]*core.DNS{},
		deploymentToPods: map[string]*list.List{},
		servicesToPods:   map[string]*list.List{},
	}
}

func (cm *componentManagerInner) SetPod(pod *core.Pod) {
	cm.mtx.Lock()
	defer cm.mtx.Unlock()
	cm.pods[pod.Name] = pod
}

func (cm *componentManagerInner) DeletePodByName(name string) {
	cm.mtx.Lock()
	defer cm.mtx.Unlock()
	delete(cm.pods, name)
	for _, pods := range cm.deploymentToPods {
		for it := pods.Front(); it != nil; it = it.Next() {
			if it.Value.(*core.Pod).Name == name {
				pods.Remove(it)
				break
			}
		}
	}
}

func (cm *componentManagerInner) GetPodByName(name string) *core.Pod {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	return cm.pods[name]
}

func (cm *componentManagerInner) PodExistsByName(name string) bool {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	_, ok := cm.pods[name]
	return ok
}

func (cm *componentManagerInner) ListPods() []*core.Pod {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	pods := make([]*core.Pod, 0, len(cm.pods))
	for _, pod := range cm.pods {
		pods = append(pods, pod)
	}
	return pods
}

func (cm *componentManagerInner) ListPodsByPhase(phase core.PodPhase) []*core.Pod {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	pods := make([]*core.Pod, 0)
	for _, pod := range cm.pods {
		if pod.Status.Phase == phase {
			pods = append(pods, pod)
		}
	}
	return pods
}

func (cm *componentManagerInner) ListPodsByLabelsAndPhase(
	labels *map[string]string,
	phase core.PodPhase,
) *list.List {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	pods := list.New()
	for _, pod := range cm.pods {
		if pod.Status.Phase == phase && reflect.DeepEqual(*labels, pod.Labels) {
			pods.PushBack(pod)
		}
	}
	return pods
}

func (cm *componentManagerInner) SetDeployment(deployment *core.Deployment, pods *list.List) {
	cm.mtx.Lock()
	defer cm.mtx.Unlock()
	for it := pods.Front(); it != nil; it = it.Next() {
		pod := it.Value.(*core.Pod)
		cm.pods[pod.Name] = pod
	}
	cm.deployments[deployment.Name] = deployment
	cm.deploymentToPods[deployment.Name] = pods
}

func (cm *componentManagerInner) DeleteDeploymentByName(deploymentName string) {
	cm.mtx.Lock()
	defer cm.mtx.Unlock()
	pods := cm.deploymentToPods[deploymentName]
	for it := pods.Front(); it != nil; it = it.Next() {
		pod := it.Value.(*core.Pod)
		delete(cm.pods, pod.Name)
	}
	delete(cm.deploymentToPods, deploymentName)
	delete(cm.deployments, deploymentName)
}

func (cm *componentManagerInner) GetDeploymentByName(name string) *core.Deployment {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	return cm.deployments[name]
}

func (cm *componentManagerInner) DeploymentExistsByName(name string) bool {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	_, ok := cm.deployments[name]
	return ok
}

func (cm *componentManagerInner) ListDeployments() []*core.Deployment {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	deployments := make([]*core.Deployment, 0, len(cm.deployments))
	for _, deployment := range cm.deployments {
		deployments = append(deployments, deployment)
	}
	return deployments
}

func (cm *componentManagerInner) ListPodsByDeploymentName(deploymentName string) *list.List {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	return cm.deploymentToPods[deploymentName]
}

func (cm *componentManagerInner) GetDeploymentByPodName(podName string) *core.Deployment {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	for deploymentName, pods := range cm.deploymentToPods {
		for it := pods.Front(); it != nil; it = it.Next() {
			pod := it.Value.(*core.Pod)
			if pod.Name == podName {
				return cm.deployments[deploymentName]
			}
		}
	}
	return nil
}

func (cm *componentManagerInner) SetService(service *core.Service, pods *list.List) {
	cm.mtx.Lock()
	defer cm.mtx.Unlock()
	cm.servicesToPods[service.Name] = pods
	cm.services[service.Name] = service
}

func (cm *componentManagerInner) DeleteServiceByName(name string) {
	cm.mtx.Lock()
	defer cm.mtx.Unlock()
	delete(cm.servicesToPods, name)
	delete(cm.services, name)
}

func (cm *componentManagerInner) GetServiceByName(name string) *core.Service {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	return cm.services[name]
}

func (cm *componentManagerInner) ServiceExistsByName(name string) bool {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	_, ok := cm.services[name]
	return ok
}

func (cm *componentManagerInner) ListServices() []*core.Service {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	services := make([]*core.Service, 0, len(cm.services))
	for _, service := range cm.services {
		services = append(services, service)
	}
	return services
}

func (cm *componentManagerInner) ListPodsByServiceName(serviceName string) *list.List {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	return cm.servicesToPods[serviceName]
}

func (cm *componentManagerInner) SetDNS(dns *core.DNS) {
	cm.mtx.Lock()
	defer cm.mtx.Unlock()
	cm.dns[dns.Name] = dns
}

func (cm *componentManagerInner) DNSExistsByName(name string) bool {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	_, ok := cm.dns[name]
	return ok
}

func (cm *componentManagerInner) DeleteDNSByName(name string) {
	cm.mtx.Lock()
	defer cm.mtx.Unlock()
	delete(cm.dns, name)
}

func (cm *componentManagerInner) GetDNSByName(name string) *core.DNS {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	return cm.dns[name]
}

func (cm *componentManagerInner) ListDNS() []*core.DNS {
	cm.mtx.RLock()
	defer cm.mtx.RUnlock()
	dnss := make([]*core.DNS, 0, len(cm.dns))
	for _, dns := range cm.dns {
		dnss = append(dnss, dns)
	}
	return dnss
}
