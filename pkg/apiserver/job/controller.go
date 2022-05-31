package job

import (
	"fmt"
	"os"

	"github.com/golang/glog"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver"
	"p9t.io/kuberboat/pkg/apiserver/node"
	"p9t.io/kuberboat/pkg/apiserver/pod"
)

// JobController actually is just a wrapper of PodController which supports run-to-completion.
type Controller interface {
	// ApplyJob creates a specific working pod.
	ApplyJob(job *core.Job) error
	// GetJobLog returns the job info as well as the output of cuda.
	GetJobLog(jobName string) (string, error)
}

type basicController struct {
	podController    pod.Controller
	nodeManager      node.NodeManager
	componentManager apiserver.ComponentManager
	retryBudget      map[string]int
}

var jobPodBase core.Pod = core.Pod{
	Kind: core.PodType,
	ObjectMeta: core.ObjectMeta{
		Labels: map[string]string{
			"JobSpecificLabel": "true",
		},
	},
	Spec: core.PodSpec{
		Containers: []core.Container{
			{
				Name:  "slurm-server",
				Image: "windowsxpbeta/slurm-server:latest",
				Resources: map[core.ResourceName]uint64{
					core.ResourceCPU:    1,
					core.ResourceMemory: 102400000,
				},
				VolumeMounts: []core.VolumeMount{
					{
						Name:      "/tmp/cuda", // We use Name to represent source since we use bind mount
						MountPath: "/src/cuda",
					},
				},
			},
		},
	},
}

func NewJobController(pc pod.Controller, nm node.NodeManager, cm apiserver.ComponentManager) *basicController {
	controller := &basicController{
		podController:    pc,
		nodeManager:      nm,
		componentManager: cm,
		retryBudget:      make(map[string]int),
	}

	apiserver.SubscribeToEvent(controller, apiserver.PodFail)
	apiserver.SubscribeToEvent(controller, apiserver.PodSucceed)
	return controller
}

func (m *basicController) createCorrespondingPod(jobName string) error {
	jobPod := jobPodBase
	jobPod.Name = jobName
	if err := m.podController.CreatePod(&jobPod); err != nil {
		// we use podController to avoid duplicate job name
		glog.Errorf("job fail to create corrsponding pod: %v", err.Error())
		return err
	}
	return nil
}

func (m *basicController) ApplyJob(job *core.Job) error {
	// we need to persist the cuda file since we may restart the Pod for recovery
	if err := os.MkdirAll("/tmp/cuda", 0777); err != nil {
		glog.Fatalf("failed to create cuda dir: %v", err.Error())
	}
	if err := os.WriteFile("/tmp/cuda/cuda.cu", job.CudaData, 0777); err != nil {
		glog.Fatalf("fail to persist cuda file: %v", err.Error())
	}
	if err := os.WriteFile("/tmp/cuda/Makefile", job.ScriptData, 0777); err != nil {
		glog.Fatalf("fail to persist compile script: %v", err.Error())
	}
	if err := m.createCorrespondingPod(job.Name); err != nil {
		return err
	}
	m.retryBudget[job.Name] = 3
	return nil
}

func (m *basicController) GetJobLog(jobName string) (string, error) {
	pod := m.componentManager.GetPodByName(jobName)
	if pod == nil {
		return "", fmt.Errorf("job %v has no pod", jobName)
	}
	client := m.nodeManager.ClientByIP(pod.Status.HostIP)
	resp, err := client.GetPodLog(jobName)
	return resp.Log, err
}

func (m *basicController) HandleEvent(event apiserver.Event) {
	switch event.Type() {
	case apiserver.PodFail:
		podName := event.(*apiserver.PodFailEvent).PodName
		budget, ok := m.retryBudget[podName]
		if ok {
			if budget > 0 {
				m.podController.DeletePodByName(podName) // avoid duplicate pod name
				m.createCorrespondingPod(podName)
				glog.Infof("job %v remain retry opportunities: %v", podName, budget-1)
				m.retryBudget[podName] = budget - 1
			} else {
				glog.Infof("job %v failed completely probably due to HPC error; goto hpc to have a check", podName)
				m.podController.DeletePodByName(podName)
			}
		}
	case apiserver.PodSucceed:
		// we cannot delete the pod otherwise we cannot reach its log
	}
}
