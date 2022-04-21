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
	DeletePod(pod *core.Pod) error
}

type basicController struct {
	// cm stores metadata of everything.
	cm apiserver.ComponentManager
	// ps tells which node to schedule a pod on.
	ps apiserver.PodScheduler
}

func NewPodController(cm apiserver.ComponentManager, ps apiserver.PodScheduler) Controller {
	return &basicController{
		cm: cm,
		ps: ps,
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

func (c *basicController) DeletePod(pod *core.Pod) error {
	c.cm.DeletePodByName(pod.Name)
	// TODO(zhidong.guo): grpc
	return nil
}
