package scale

import (
	"container/list"
	"fmt"
	"math"
	"time"

	"github.com/golang/glog"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver"
)

// To avoid frequent scaling in and out when the resource usage of a pod is close to the
// threshold, the scale-out threshold is slightly increased by multiplying a constant
// ScaleOutTargetExpansionRate.
const ScaleOutTargetExpansionRate float64 = 1.05

type Controller interface {
	// CreateAutoscaler creates an autoscaler.
	CreateAutoscaler(autoscaler *core.HorizontalPodAutoscaler) error
	// DescribeAutoscalers returns information about autoscalers specified by autoscalerNames.
	DescribeAutoscalers(all bool, autoscalerNames []string) ([]*core.HorizontalPodAutoscaler, []string)
}

type basicController struct {
	componentManager apiserver.ComponentManager
	metricsManager   MetricsManager
}

func NewAutoscalerController(
	componentManager apiserver.ComponentManager,
	metricsManager MetricsManager,
) Controller {
	return &basicController{
		componentManager: componentManager,
		metricsManager:   metricsManager,
	}
}

func (bc *basicController) startAutoscalerMonitor(autoscaler *core.HorizontalPodAutoscaler) {
	deploymentName := autoscaler.Spec.ScaleTargetRef.Name
	monitorInterval := time.Second * time.Duration(autoscaler.Spec.ScaleInterval)
	ticker := time.NewTicker(monitorInterval)
	for range ticker.C {
		if !bc.componentManager.DeploymentExistsByName(deploymentName) {
			// Deployment does not exist. Just delete the autoscaler.
			bc.componentManager.DeleteAutoscalerByName(autoscaler.Name)
			return
		}
		deployment := bc.componentManager.GetDeploymentByName(deploymentName)
		bc.monitorAndScaleDeployment(autoscaler, deployment)
	}
}

func (bc *basicController) monitorAndScaleDeployment(
	autoscaler *core.HorizontalPodAutoscaler,
	deployment *core.Deployment,
) {
	// All the computations done below are based on this snapshot of pods.
	pods := bc.componentManager.ListPodsByDeploymentName(deployment.Name)

	// If deployment has no pod, just return.
	podNum := pods.Len()
	if podNum == 0 {
		return
	}
	// Old states have not been cleared. Just wait for next round of monitor.
	if deployment.Status.ReadyReplicas > autoscaler.Spec.MaxReplicas ||
		deployment.Status.ReadyReplicas < autoscaler.Spec.MinReplicas {
		return
	}
	// It should be ensured that all pods in the deployments are ready.
	for it := pods.Front(); it != nil; it = it.Next() {
		pod := it.Value.(*core.Pod)
		if pod.Status.Phase != core.PodReady {
			return
		}
	}

	var deploymentCPUUsage float64
	var cpuUsagePerPod float64
	var deploymentMemoryUsage uint64
	var memoryUsagePerPod uint64
	var err error

	for _, metric := range autoscaler.Spec.Metrics {
		switch metric.Resource {
		case core.ResourceCPU:
			deploymentCPUUsage, err = bc.sumDeploymentCPUUsage(pods)
			if err != nil {
				// If any error occurs during fetching the data of a pod, just print out the
				// error and return. This might be caused by the data inconsistency between
				// apiserver and node, which would be synchronized later.
				glog.Warning(err)
				return
			}
			cpuUsagePerPod = deploymentCPUUsage / float64(podNum)
			if deployment.Status.Replicas < autoscaler.Spec.MaxReplicas &&
				cpuUsagePerPod > float64(metric.TargetUtilization)*ScaleOutTargetExpansionRate {
				// We just alter the number of deployment replicas here. DeploymentController
				// will monitor the change of replica number and do the scale automatically.
				// We increase the replica number each time by one.
				deployment.Spec.Replicas++
				glog.Infof(
					"AUTOSCALER [%s]: cpu usage per pod reaches %f, deployment %s scales out to %d replicas\n",
					autoscaler.Name,
					cpuUsagePerPod,
					deployment.Name,
					deployment.Spec.Replicas,
				)
				return
			}

		case core.ResourceMemory:
			deploymentMemoryUsage, err = bc.sumDeploymentMemoryUsage(pods)
			if err != nil {
				// Handle the error same as above.
				glog.Warning(err)
				return
			}
			memoryUsagePerPod = deploymentMemoryUsage / uint64(podNum)
			if deployment.Status.Replicas < autoscaler.Spec.MaxReplicas &&
				memoryUsagePerPod > uint64(float64(metric.TargetUtilization)*ScaleOutTargetExpansionRate) {
				deployment.Spec.Replicas++
				glog.Infof(
					"AUTOSCALER [%s]: memory usage per pod reaches %d, deployment %s scales out to %d replicas\n",
					autoscaler.Name,
					memoryUsagePerPod,
					deployment.Name,
					deployment.Spec.Replicas,
				)
				return
			}
		}
	}

	// If the deployment does not scale out on all metrics, and if there is no room for scaling in,
	// just return.
	if deployment.Status.Replicas <= autoscaler.Spec.MinReplicas {
		return
	}

	// Chances are that the deployment might scale in.
	shouldScaleInMetrics := 0
	for _, metric := range autoscaler.Spec.Metrics {
		switch metric.Resource {
		case core.ResourceCPU:
			desiredPodNum := int(math.Ceil(
				deploymentCPUUsage / (float64(metric.TargetUtilization)),
			))
			if desiredPodNum < podNum {
				shouldScaleInMetrics++
			}
		case core.ResourceMemory:
			var carry int
			if deploymentMemoryUsage%metric.TargetUtilization != 0 {
				carry = 1
			} else {
				carry = 0
			}
			desiredPodNum := int(deploymentMemoryUsage/metric.TargetUtilization) + carry
			if desiredPodNum < podNum {
				shouldScaleInMetrics++
			}
		}
	}
	// If all metrics satisfy the condition of scaling in, we do scaling in by decreasing the
	// replica number each time by one.
	if shouldScaleInMetrics == len(autoscaler.Spec.Metrics) {
		deployment.Spec.Replicas--
		glog.Infof(
			"AUTOSCALER [%s]: deployment %s scales in to %d replica(s)\n",
			autoscaler.Name,
			deployment.Name,
			deployment.Spec.Replicas,
		)
	}
}

func (bc *basicController) sumDeploymentCPUUsage(podsInDeployment *list.List) (float64, error) {
	var totalCPUUsage float64 = 0.0
	for it := podsInDeployment.Front(); it != nil; it = it.Next() {
		pod := it.Value.(*core.Pod)
		podCPUUsage, err := bc.metricsManager.PodCPUUsage(pod)
		if err != nil {
			return 0.0, err
		}
		totalCPUUsage += podCPUUsage
	}
	return totalCPUUsage, nil
}

func (bc *basicController) sumDeploymentMemoryUsage(podsInDeployment *list.List) (uint64, error) {
	var memoryUsage uint64 = 0
	for it := podsInDeployment.Front(); it != nil; it = it.Next() {
		pod := it.Value.(*core.Pod)
		podMemoryUsage, err := bc.metricsManager.PodMemoryUsage(pod)
		if err != nil {
			return 0, err
		}
		memoryUsage += podMemoryUsage
	}
	return memoryUsage, nil
}

func (bc *basicController) CreateAutoscaler(autoscaler *core.HorizontalPodAutoscaler) error {
	if bc.componentManager.AutoscalerExistsByName(autoscaler.Name) {
		return fmt.Errorf("autoscaler already exists: %v", autoscaler.Name)
	}

	deploymentName := autoscaler.Spec.ScaleTargetRef.Name
	if !bc.componentManager.DeploymentExistsByName(deploymentName) {
		return fmt.Errorf("no such deployment to be monitored by autoscaler: %v", deploymentName)
	}
	if bc.componentManager.DeploymentAutoscaled(deploymentName) {
		return fmt.Errorf("deployment %v already monitored by autoscaler", deploymentName)
	}

	autoscaler.CreationTimestamp = time.Now()
	bc.componentManager.SetAutoscaler(autoscaler)

	deployment := bc.componentManager.GetDeploymentByName(deploymentName)
	clipDeploymentReplicas(deployment, autoscaler)

	go bc.startAutoscalerMonitor(autoscaler)

	glog.Infof("AUTOSCALER [%v]: autoscaler created on deployment %v", autoscaler.Name, deploymentName)

	return nil
}

func clipDeploymentReplicas(deployment *core.Deployment, autoscaler *core.HorizontalPodAutoscaler) {
	if deployment.Spec.Replicas < autoscaler.Spec.MinReplicas {
		deployment.Spec.Replicas = autoscaler.Spec.MinReplicas
	} else if deployment.Spec.Replicas > autoscaler.Spec.MaxReplicas {
		deployment.Spec.Replicas = autoscaler.Spec.MaxReplicas
	}
}

func (bc *basicController) DescribeAutoscalers(all bool, autoscalerNames []string) (
	[]*core.HorizontalPodAutoscaler,
	[]string,
) {
	if all {
		return bc.componentManager.ListAutoscalers(), []string{}
	} else {
		foundAutoscalers := make([]*core.HorizontalPodAutoscaler, 0)
		notFoundAutoscalers := make([]string, 0)
		for _, name := range autoscalerNames {
			if !bc.componentManager.AutoscalerExistsByName(name) {
				notFoundAutoscalers = append(notFoundAutoscalers, name)
			} else {
				autoscaler := bc.componentManager.GetAutoscalerByName(name)
				if autoscaler == nil {
					glog.Errorf("autoscaler missing event if cm claims otherwise")
					continue
				}
				foundAutoscalers = append(foundAutoscalers, autoscaler)
			}
		}
		return foundAutoscalers, notFoundAutoscalers
	}
}
