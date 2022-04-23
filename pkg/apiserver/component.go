package apiserver

import (
	"container/list"

	"p9t.io/kuberboat/pkg/api/core"
)

/// ComponentManager serves as a cache for pods, services and deployments of the cluster in
/// API Server. There is no lock inside. The outer logic should be responsible for thread safety.
type ComponentManager interface {
	/// SetPod sets a pod into ComponentManager. This function will not check the existence of the
	/// pod. To check for existence, you should call `PodExistsByName`.
	SetPod(pod *core.Pod)
	/// DeletePodByName deletes a pod by name from ComponentManager. This function will not check
	/// the existence of the pod.
	DeletePodByName(name string)
	/// GetPodByName gets a pod from ComponentManager by name.
	GetPodByName(name string) *core.Pod
	/// PodExistsByName checks whether a pod of a specific name exists.
	PodExistsByName(name string) bool
	/// ListPods lists all the pods present.
	ListPods() []*core.Pod

	/// SetDeployment sets a deployment and the pods it creates into ComponentManager. This
	/// function will not check the existence of the deployment.
	SetDeployment(deployment *core.Deployment, pods *list.List)
	/// DeleteDeploymentByName deletes a deployment by its name as well as all of the pods it creates
	/// from ComponentManager. This function will not check the existence of the deployment.
	DeleteDeploymentByName(deploymentName string)
	/// GetDeploymentByName gets a deployment from ComponentManager by name.
	GetDeploymentByName(name string) *core.Deployment
	/// DeploymentExistsByName checks whether a deployment of a specific name exists.
	DeploymentExistsByName(name string) bool

	/// ListPodsByDeployment lists all the pods given the name of a deployment. This function will not
	/// check the existence of the deployment. If the deployment does not exist, an empty array will be
	/// returned.
	ListPodsByDeploymentName(deploymentName string) *list.List
	/// GetDeploymentByPod gets the deployment a pod belongs to by the name of the pod. This function will not
	/// check the existence of the pod. If the pod does not belong to any deployment, the function will return
	/// nil.
	GetDeploymentByPodName(podName string) *core.Deployment
}

type componentManagerInner struct {
	/// Stores the mapping from pod name to pod.
	pods map[string]*core.Pod
	/// Stores the mapping from service name to service.
	services map[string]*core.Service
	/// Stores the mapping from service name to service.
	deployments map[string]*core.Deployment
	/// Stores the mapping from the name of a deployment to the pods it creates.
	deploymentToPods map[string]*list.List
}

func NewComponentManager() ComponentManager {
	return &componentManagerInner{
		pods:             map[string]*core.Pod{},
		services:         map[string]*core.Service{},
		deployments:      map[string]*core.Deployment{},
		deploymentToPods: map[string]*list.List{},
	}
}

func (cm *componentManagerInner) SetPod(pod *core.Pod) {
	cm.pods[pod.Name] = pod
}

func (cm *componentManagerInner) DeletePodByName(name string) {
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
	return cm.pods[name]
}

func (cm *componentManagerInner) PodExistsByName(name string) bool {
	_, ok := cm.pods[name]
	return ok
}

func (cm *componentManagerInner) ListPods() []*core.Pod {
	pods := make([]*core.Pod, 0, len(cm.pods))
	for _, pod := range cm.pods {
		pods = append(pods, pod)
	}
	return pods
}

func (cm *componentManagerInner) SetDeployment(deployment *core.Deployment, pods *list.List) {
	for it := pods.Front(); it != nil; it = it.Next() {
		pod := it.Value.(*core.Pod)
		cm.pods[pod.Name] = pod
	}
	cm.deployments[deployment.Name] = deployment
	cm.deploymentToPods[deployment.Name] = pods
}

func (cm *componentManagerInner) DeleteDeploymentByName(deploymentName string) {
	pods := cm.deploymentToPods[deploymentName]
	for it := pods.Front(); it != nil; it = it.Next() {
		pod := it.Value.(*core.Pod)
		delete(cm.pods, pod.Name)
	}
	delete(cm.deploymentToPods, deploymentName)
	delete(cm.deployments, deploymentName)
}

func (cm *componentManagerInner) GetDeploymentByName(name string) *core.Deployment {
	return cm.deployments[name]
}

func (cm *componentManagerInner) DeploymentExistsByName(name string) bool {
	_, ok := cm.deployments[name]
	return ok
}

func (cm *componentManagerInner) ListPodsByDeploymentName(deploymentName string) *list.List {
	return cm.deploymentToPods[deploymentName]
}

func (cm *componentManagerInner) GetDeploymentByPodName(podName string) *core.Deployment {
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
