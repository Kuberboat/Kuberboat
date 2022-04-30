package pod

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver"
)

type Controller interface {
	// GetPods returns information about pods specified by podName.
	// Return value is composed of pods that are found and pod names that do not exist.
	GetPods(all bool, podNames []string) ([]*core.Pod, []string)
	// CreatePod does the following:
	//		1. Choose a node for the pod.
	// 		2. Fill some system-generated properties of the pod.
	// 		3. Modify metadata in component manager.
	// 		4. Use grpc to inform kubelet on the node to create the pod.
	// The information of a pod should be valid.
	CreatePod(pod *core.Pod) error
	// DeletePod does the following:
	// 		1. Modify metadata in component manager.
	// 		2. Use grpc to inform kubelet on the node to remove the pod.
	DeletePodByName(name string) error
	// DeleteAllPods is just a wrapper that iterates through all pods and call DeletePodByName on it.
	DeleteAllPods() error
	// UpdatePodStatus updates the status of a pod when API server is notified by Kubelet.
	UpdatePodStatus(podName string, podStatus *core.PodStatus) error
}

type basicController struct {
	// componentManager stores the components and the dependencies between them.
	componentManager apiserver.ComponentManager
	// podScheduler tells which node to schedule a pod on.
	podScheduler apiserver.PodScheduler
	// nodeManager provides grpc client to pod controller.
	nodeManager apiserver.NodeManager
}

func NewPodController(
	componentManager apiserver.ComponentManager,
	podScheduler apiserver.PodScheduler,
	nodeManager apiserver.NodeManager,
) Controller {
	return &basicController{
		componentManager: componentManager,
		podScheduler:     podScheduler,
		nodeManager:      nodeManager,
	}
}

func (c *basicController) GetPods(all bool, podNames []string) ([]*core.Pod, []string) {
	if all {
		return c.componentManager.ListPods(), make([]string, 0)
	} else {
		foundPods := make([]*core.Pod, 0)
		notFoundPods := make([]string, 0)
		for _, name := range podNames {
			if !c.componentManager.PodExistsByName(name) {
				notFoundPods = append(notFoundPods, name)
			} else {
				pod := c.componentManager.GetPodByName(name)
				if pod == nil {
					glog.Errorf("pod missing event if cm claims otherwise")
					continue
				}
				foundPods = append(foundPods, pod)
			}
		}
		return foundPods, notFoundPods
	}
}

func (c *basicController) CreatePod(pod *core.Pod) error {
	if c.componentManager.PodExistsByName(pod.Name) {
		return fmt.Errorf("pod already exists: %v", pod.Name)
	}
	node, client := c.podScheduler.SchedulePod(pod)
	if node == nil {
		return errors.New("no available worker to schedule the pod")
	}
	pod.UUID = uuid.New()
	pod.CreationTimestamp = time.Now()
	pod.Status.Phase = core.PodPending
	pod.Status.HostIP = node.Status.Address

	c.componentManager.SetPod(pod)

	_, err := client.CreatePod(pod)
	if err != nil {
		return err
	}
	glog.Infof("created pod %v on node with IP %v", pod.Name, pod.Status.HostIP)

	return nil
}

func (c *basicController) DeletePodByName(name string) error {
	if !c.componentManager.PodExistsByName(name) {
		return fmt.Errorf("no such pod: %v", name)
	}
	pod := c.componentManager.GetPodByName(name)
	if pod == nil {
		return fmt.Errorf("race condition on pod: %v", name)
	}

	ip := pod.Status.HostIP
	client := c.nodeManager.ClientByIP(ip)
	if client == nil {
		return fmt.Errorf("cannot find grpc client for worker at address: %v", ip)
	}

	_, err := client.DeletePodByName(name)
	if err != nil {
		return fmt.Errorf("cannot remove pod: %v", err.Error())
	}
	c.componentManager.DeletePodByName(name)

	return nil
}

func (c *basicController) DeleteAllPods() error {
	for _, pod := range c.componentManager.ListPods() {
		if err := c.DeletePodByName(pod.Name); err != nil {
			return err
		}
	}
	return nil
}

func (c *basicController) UpdatePodStatus(podName string, podStatus *core.PodStatus) error {
	if !c.componentManager.PodExistsByName(podName) {
		return fmt.Errorf("no such pod: %v", podName)
	}

	pod := c.componentManager.GetPodByName(podName)
	if pod == nil {
		return fmt.Errorf("race condition on pod: %v", podName)
	}

	pod.Status = *podStatus

	return nil
}
