package pod

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver"
)

type Controller interface {
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
}

type basicController struct {
	// cm stores metadata of everything.
	cm apiserver.ComponentManager
	// ps tells which node to schedule a pod on.
	ps apiserver.PodScheduler
	// nm provides grpc client to pod controller.
	nm apiserver.NodeManager
}

func NewPodController(cm apiserver.ComponentManager, ps apiserver.PodScheduler, nm apiserver.NodeManager) Controller {
	return &basicController{
		cm: cm,
		ps: ps,
		nm: nm,
	}
}

func (c *basicController) CreatePod(pod *core.Pod) error {
	if c.cm.PodExistsByName(pod.Name) {
		return fmt.Errorf("pod already exists: %v", pod.Name)
	}
	node, client := c.ps.SchedulePod(pod)
	if node == nil {
		return errors.New("no available worker to schedule the pod")
	}
	pod.UUID = uuid.New()
	pod.CreationTimestamp = time.Now()
	pod.Status.Phase = core.PodPending
	pod.Status.HostIP = node.Status.Address

	c.cm.SetPod(pod)

	_, err := client.CreatePod(pod)
	if err != nil {
		return err
	}

	return nil
}

func (c *basicController) DeletePodByName(name string) error {
	if !c.cm.PodExistsByName(name) {
		return fmt.Errorf("no such pod: %v", name)
	}
	pod := c.cm.GetPodByName(name)
	if pod == nil {
		return fmt.Errorf("race condition on pod: %v", name)
	}

	ip := pod.Status.HostIP
	client := c.nm.ClientByIP(ip)
	if client == nil {
		return fmt.Errorf("cannot find grpc client for worker at address: %v", ip)
	}

	_, err := client.DeletePodByName(name)
	if err != nil {
		return fmt.Errorf("cannot remove pod: %v", err.Error())
	}
	c.cm.DeletePodByName(name)

	return nil
}

func (c *basicController) DeleteAllPods() error {
	for _, pod := range c.cm.ListPods() {
		if err := c.DeletePodByName(pod.Name); err != nil {
			return err
		}
	}
	return nil
}
