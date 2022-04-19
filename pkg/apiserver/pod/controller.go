package pod

import (
	"fmt"
	"net"

	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver"
)

type Controller interface {
	// CreatePod does the following:
	//		1. Sanity check pod information.
	// 		2. Modify metadata in component manager.
	// 		3. Use grpc to inform kubelet on the node to create the pod.
	// The information of a pod should be valid.
	CreatePod(pod *core.Pod) error
	// DeletePod does the following:
	//		1. Sanity check pod information.
	// 		2. Modify metadata in component manager.
	// 		3. Use grpc to inform kubelet on the node to remove the pod.
	DeletePod(pod *core.Pod) error
}

type basicController struct {
	// cm stores metadata of everything.
	cm apiserver.ComponentManager
}

func NewPodController(cm apiserver.ComponentManager) Controller {
	return &basicController{
		cm: cm,
	}
}

func (c *basicController) CreatePod(pod *core.Pod) error {
	if err := sanityCheck(pod); err != nil {
		return err
	}
	c.cm.SetPod(pod)
	// TODO(zhidong.guo): grpc
	return nil
}

func (c *basicController) DeletePod(pod *core.Pod) error {
	c.cm.DeletePodByName(pod.Name)
	// TODO(zhidong.guo): grpc
	return nil
}

func sanityCheck(pod *core.Pod) error {
	if pod.Status.Phase != core.PodPending {
		return fmt.Errorf("invalid pod phase: %v", pod.Status.Phase)
	}
	if net.ParseIP(pod.Status.HostIP) == nil {
		return fmt.Errorf("invalid host IP: %v", pod.Status.HostIP)
	}
	return nil
}
