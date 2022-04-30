package deployment

import (
	"container/list"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"p9t.io/kuberboat/pkg/api"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver"
	"p9t.io/kuberboat/pkg/apiserver/pod"
)

const (
	// Interval in seconds between two checks of deployment spec against status.
	monitorInterval = 3
)

// DeploymentController manages deployments.
type Contoller interface {
	// ApplyDeployment creates a new deployment currently no deployment with the same name exists.
	// Otherwise, update the pods in the deployment.
	//
	// Either way, the deployment object in ComponentManager will be replaced with the new deployment,
	// so deployment needs to inherit the status of its older version (if it exists).
	ApplyDeployment(deployment *core.Deployment) error
	// DeleteDeploymentByName deletes the deployment and its pods.
	// IMPORTANT: Must be called BEFORE metadata is modified, so the deployment can know what pods to delete.
	DeleteDeploymentByName(name string) error
	// monitorDeployment checks if the status of deployments matches their specs.
	// If not, make adjustments.
	monitorDeployment()
}

type basicController struct {
	mtx sync.Mutex
	// podManager provides basicManager interfaces to manipulate pods.
	componentManager apiserver.ComponentManager
	// podController performs the actual creating/deleting pods.
	podController pod.Controller
}

func NewDeploymentController(componentManager apiserver.ComponentManager, pc pod.Controller) *basicController {
	controller := &basicController{
		componentManager: componentManager,
		podController:    pc,
	}
	go func() {
		for range time.Tick(time.Second * monitorInterval) {
			controller.monitorDeployment()
		}
	}()

	apiserver.SubscribeToEvent(controller, apiserver.PodDeletion)
	apiserver.SubscribeToEvent(controller, apiserver.PodReady)

	return controller
}

func (m *basicController) ApplyDeployment(deployment *core.Deployment) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	isDeploymentExistent := m.componentManager.DeploymentExistsByName(deployment.Name)
	if isDeploymentExistent {
		existingDeployment := m.componentManager.GetDeploymentByName(deployment.Name)
		if isDeploymentUpdated(existingDeployment, deployment) {
			// TODO(zhidong.guo): Mark the pod as being rolling updated.
			return errors.New("rolling update not implemented")
		} else {
			existingDeployment.Spec.Replicas = deployment.Spec.Replicas
		}
	} else {
		initDeployment(deployment)
		m.componentManager.SetDeployment(deployment, list.New())
	}
	return nil
}

// Only Replicas will be incremented. ReadyReplcas and UpdatedRelicas will be modified when receiving events.
func (m *basicController) morePods(deployment *core.Deployment, existingPods *list.List, numPodsToAdd int) {
	if existingPods == nil {
		panic("pod list is nil even after checking")
	}

	glog.Infof("DEPLOYMENT [%v]: adding %v pods", deployment.Name, numPodsToAdd)
	numPodsAdded := 0
	// Create new pods from template. Keep creating even if it fails.
	specHash := api.Hash(deployment.Spec)
	for i := 0; i < int(numPodsToAdd); i++ {
		p := &core.Pod{Kind: core.PodType}
		p.Name = getPodName(deployment, specHash)
		p.Labels = deployment.Spec.Template.Labels
		p.Spec = deployment.Spec.Template.Spec

		if err := m.podController.CreatePod(p); err != nil {
			glog.Errorf("DEPLOYMENT [%v]: failed to create pod: ", deployment.Name, err.Error())
			continue
		} else {
			deployment.Status.Replicas++
			numPodsAdded++
		}

		existingPods.PushBack(p)
		glog.Infof("DEPLOYMENT [%v]: added pod [%v]", deployment.Name, p.Name)
	}
	glog.Infof("DEPLOYMENT [%v]: expected to add %v pods, actually added %v", deployment.Name, numPodsToAdd, numPodsAdded)
}

// Replicas, ReadyReplicas and UpdatedReplicas are updated immediately after grpc returns successfully.
func (m *basicController) fewerPods(deployment *core.Deployment, existingPods *list.List, numPodsToDelete int) {
	if existingPods == nil {
		panic("pod list is nil even after checking")
	}

	glog.Infof("DEPLOYMENT [%v]: deleting %v pods", deployment.Name, numPodsToDelete)
	// Remove the latest pods
	if existingPods.Len() < numPodsToDelete {
		glog.Errorf("deployment status and number of pods to remove do not match: %v vs. %v", existingPods.Len(), numPodsToDelete)
	}

	numPodsDeleted := 0
	for i := 0; i < int(numPodsToDelete); i++ {
		it := existingPods.Back()
		p := it.Value.(*core.Pod)
		if err := m.podController.DeletePodByName(p.Name); err != nil {
			glog.Errorf("DEPLOYMENT [%v]: failed to delete pod: %v", deployment.Name, err.Error())
			continue
		}

		// DeletePodByName will alter deployment's pod list, so no need to modify it here.
		numPodsDeleted++
		updateDeploymentStatusOnPodRemoval(deployment, p)
		glog.Infof("DEPLOYMENT [%v]: deleted pod [%v]", deployment.Name, p.Name)
	}

	glog.Infof("DEPLOYMENT [%v]: expected to delete %v pods, actually deleted %v", deployment.Name, numPodsToDelete, numPodsDeleted)
}

func (m *basicController) DeleteDeploymentByName(name string) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if m.componentManager.DeploymentExistsByName(name) {
		glog.Infof("DEPLOYMENT [%v]: deleting", name)
		// Delete all pods belonging to the deployment.
		podList := m.componentManager.ListPodsByDeploymentName(name)
		if podList == nil {
			glog.Errorf("DEPLOYMENT [%v]: nil pod list", name)
			return fmt.Errorf("unable to find pods for deployment [%v]", name)
		}
		// Repeatedly call Front(), because list is being changed for each call to DeletePodByName.
		for i := podList.Front(); podList.Len() > 0; i = podList.Front() {
			podName := i.Value.(*core.Pod).Name
			if err := m.podController.DeletePodByName(i.Value.(*core.Pod).Name); err != nil {
				glog.Errorf("DEPLOYMENT [%v]: unable to delete pod [%v]: %v", name, podName, err.Error())
			}
			glog.Infof("DEPLOYMENT [%v]: deleted pod [%v]", name, podName)
		}

		// Delete the deployment.
		m.componentManager.DeleteDeploymentByName(name)
		glog.Infof("DEPLOYMENT [%v]: successfully deleted", name)
	} else {
		return fmt.Errorf("no such deployment: %v", name)
	}

	return nil
}

func (m *basicController) HandleEvent(event apiserver.Event) {
	switch event.Type() {
	case apiserver.PodDeletion:
		m.handlePodDeletion(event.(*apiserver.PodDeletionEvent).Pod)
	case apiserver.PodReady:
		m.handlePodReady(event.(*apiserver.PodReadyEvent).Pod)
	}
}

func (m *basicController) handlePodDeletion(pod *core.Pod) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if deployment := m.componentManager.GetDeploymentByPodName(pod.Name); deployment != nil {
		updateDeploymentStatusOnPodRemoval(deployment, pod)
		// TODO: If the pod deletion is issued by the deployment owning the pod, we should not apply current deployment,
		// otherwise a loop is formed.
		return m.ApplyDeployment(deployment)
	} else {
		return nil
	}
}

func (m *basicController) handlePodReady(pod *core.Pod) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	// It's not likely that the number of pods exceed the deployment's desired number, the only case being when
	// the number of desired replicas is decreased by auto scaler or the user. In that case, applyDeployment will handle pod deletion.
	if deployment := m.componentManager.GetDeploymentByPodName(pod.Name); deployment != nil {
		deployment.Status.ReadyReplicas++
		deployment := m.componentManager.GetDeploymentByPodName(pod.Name)
		// If a pod becomes ready but is not updated, it could only be that the deployment template has been chaged
		// during pod creation. And we currently have no support for this scenario.
		if isPodUpdated(deployment, pod) {
			deployment.Status.ReadyReplicas++
		} else {
			glog.Error("pod ready but is outdated")
		}
	}
}

func (m *basicController) monitorDeployment() {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	deployments := m.componentManager.ListDeployments()
	for _, deployment := range deployments {
		numPodDiff := int(deployment.Spec.Replicas) - int(deployment.Status.Replicas)
		pods := m.componentManager.ListPodsByDeploymentName(deployment.Name)
		if pods == nil {
			glog.Errorf("DEPLOYMENT [%v]: nil pod list", deployment.Name)
		}
		if numPodDiff > 0 {
			m.morePods(deployment, pods, numPodDiff)
		} else if numPodDiff < 0 {
			m.fewerPods(deployment, pods, -numPodDiff)
		}
	}
}

func initDeployment(deployment *core.Deployment) {
	deployment.UUID = uuid.New()
	deployment.CreationTimestamp = time.Now()
	deployment.Status.Replicas = 0
	deployment.Status.ReadyReplicas = 0
	deployment.Status.UpdatedReplicas = 0
}

func getPodName(d *core.Deployment, specHash string) string {
	// Pod UUID is still unknown, so we will just generate a new UUID.
	return d.Name + "-" + specHash[0:10] + "-" + uuid.NewString()[0:5]
}

func isDeploymentUpdated(d1 *core.Deployment, d2 *core.Deployment) bool {
	return api.Hash(d1.Spec.Template.Labels) == api.Hash(d2.Spec.Template.Labels) &&
		api.Hash(d1.Spec.Template.Spec) == api.Hash(d2.Spec.Template.Spec)
}

func isPodUpdated(deployment *core.Deployment, pod *core.Pod) bool {
	return api.Hash(deployment.Spec.Template.Labels) == api.Hash(pod.Labels) &&
		api.Hash(deployment.Spec.Template.Spec) == api.Hash(pod.Spec)
}

func updateDeploymentStatusOnPodRemoval(deployment *core.Deployment, pod *core.Pod) {
	// Deleting pod is presumably guaranteed to succeed.
	deployment.Status.Replicas--
	if isPodUpdated(deployment, pod) {
		deployment.Status.UpdatedReplicas--
	}
	if pod.Status.Phase == core.PodReady {
		deployment.Status.ReadyReplicas--
	}
}
