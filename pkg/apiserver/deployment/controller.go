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
	"p9t.io/kuberboat/pkg/apiserver/etcd"
	"p9t.io/kuberboat/pkg/apiserver/pod"
)

const (
	// Interval in seconds between two checks of deployment spec against status.
	monitorInterval = 3
)

// DeploymentController manages deployments.
type Contoller interface {
	// DescribeDeployments return all the deployments and their respective pods.
	DescribeDeployments(all bool, names []string) ([]*core.Deployment, [][]string, []string)
	// ApplyDeployment creates a new deployment currently no deployment with the same name exists.
	// Otherwise, update the pods in the deployment.
	//
	// Either way, the deployment object in ComponentManager will be replaced with the new deployment,
	// so deployment needs to inherit the status of its older version (if it exists).
	ApplyDeployment(deployment *core.Deployment) error
	// DeleteDeploymentByName deletes the deployment and its pods.
	// IMPORTANT: Must be called BEFORE metadata is modified, so the deployment can know what pods to delete.
	DeleteDeploymentByName(name string) error
	// DeleteAllDeployments deletes all deployments by calling DeleteDeploymentByName.
	DeleteAllDeployments() error
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
	// expectDeletedPod is used to avoid updating deployment status twice when a deployment
	// requests a pod to be deleted. The first update happens when issuing deletion request,
	// and the second (if not checked) will happen in pod deletion handler.
	expectDeletedPod map[string]struct{}
}

func NewDeploymentController(componentManager apiserver.ComponentManager, pc pod.Controller) *basicController {
	controller := &basicController{
		componentManager: componentManager,
		podController:    pc,
		expectDeletedPod: map[string]struct{}{},
	}
	go func() {
		for range time.Tick(time.Second * monitorInterval) {
			controller.monitorDeployment()
		}
	}()

	apiserver.SubscribeToEvent(controller, apiserver.PodDeletion)
	apiserver.SubscribeToEvent(controller, apiserver.PodReady)
	apiserver.SubscribeToEvent(controller, apiserver.PodFail)

	return controller
}

func (m *basicController) DescribeDeployments(all bool, names []string) ([]*core.Deployment, [][]string, []string) {
	getDeploymentPodNames := func(deployment *core.Deployment) []string {
		ret := make([]string, 0)
		pods := m.componentManager.ListPodsByDeploymentName(deployment.Name)
		for i := pods.Front(); i != nil; i = i.Next() {
			ret = append(ret, i.Value.(*core.Pod).Name)
		}
		return ret
	}
	deploymentPods := make([][]string, 0)
	if all {
		deployments := m.componentManager.ListDeployments()
		for _, deployment := range deployments {
			deploymentPods = append(deploymentPods, getDeploymentPodNames(deployment))
		}
		return deployments, deploymentPods, make([]string, 0)
	} else {
		foundDeployments := make([]*core.Deployment, 0)
		notFoundDeployments := make([]string, 0)
		for _, name := range names {
			if !m.componentManager.DeploymentExistsByName(name) {
				notFoundDeployments = append(notFoundDeployments, name)
			} else {
				deployment := m.componentManager.GetDeploymentByName(name)
				if deployment == nil {
					glog.Errorf("deployment missing even if cm claims otherwise")
					continue
				}
				foundDeployments = append(foundDeployments, deployment)
				deploymentPods = append(deploymentPods, getDeploymentPodNames(deployment))
			}
		}
		return foundDeployments, deploymentPods, notFoundDeployments
	}
}

func (m *basicController) ApplyDeployment(deployment *core.Deployment) error {
	// Updating a deployment monitored by autoscaler is not allowed.
	if m.componentManager.DeploymentAutoscaled(deployment.Name) {
		return fmt.Errorf(
			"deployment %s is monitored by autoscaler and cannot be updated",
			deployment.Name,
		)
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()

	isDeploymentExistent := m.componentManager.DeploymentExistsByName(deployment.Name)
	if isDeploymentExistent {
		existingDeployment := m.componentManager.GetDeploymentByName(deployment.Name)

		// Trigger rolling update by setting updatedPods to 0.
		if !isDeploymentUpdated(existingDeployment, deployment) {
			if deployment.Spec.RollingUpdate.MaxSurge == 0 && deployment.Spec.RollingUpdate.MaxUnavailable == 0 {
				return errors.New("cannot trigger rolling update when maxSurge and maxUnavailable are both 0")
			}
			updateDeploymentTemplate(existingDeployment, deployment)
			existingDeployment.Status.UpdatedReplicas = 0
		}
		existingDeployment.Spec.Replicas = deployment.Spec.Replicas
		existingDeployment.Spec.RollingUpdate = deployment.Spec.RollingUpdate

		// Update the deployment metadata.
		// Should have updated etcd before modifying existingDeployment.
		if err := setDeploymentInEtcd(deployment.Name, existingDeployment); err != nil {
			return err
		}
	} else {
		initDeployment(deployment)
		if err := setDeploymentInEtcd(deployment.Name, deployment); err != nil {
			return err
		}
		m.componentManager.SetDeployment(deployment, list.New())
	}

	glog.Infof(
		"DEPLOYMENT [%v]: deployment created with %d replicas",
		deployment.Name,
		deployment.Spec.Replicas,
	)

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
	specHash := computeSpecHash(deployment)
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
	if err := etcd.Put(fmt.Sprintf("/Deployments/Meta/%s", deployment.Name), deployment); err != nil {
		glog.Errorf("failed to update deployment's metadata: %v", err)
	}
	if err := etcd.Put(fmt.Sprintf("/Deployments/Pods/%s", deployment.Name), core.GetPodNames(existingPods)); err != nil {
		glog.Errorf("failed to update deployment's corresponding pods: %v", err)
	}
	glog.Infof("DEPLOYMENT [%v]: expected to add %v pods, actually added %v", deployment.Name, numPodsToAdd, numPodsAdded)
}

// Replicas, ReadyReplicas and UpdatedReplicas are updated immediately after grpc returns successfully.
// Try to find and delete outdated pods first.
func (m *basicController) fewerPods(deployment *core.Deployment, existingPods *list.List, numPodsToDelete int) {
	if existingPods == nil {
		panic("pod list is nil even after checking")
	}

	glog.Infof("DEPLOYMENT [%v]: deleting %v pods", deployment.Name, numPodsToDelete)
	// Remove the latest pods
	if existingPods.Len() < numPodsToDelete {
		glog.Errorf("deployment status and number of pods to remove do not match: %v vs. %v", existingPods.Len(), numPodsToDelete)
	}

	// Try to delete outdated pods first.
	numOutdatedPodsDeleted := 0
	outdatedPods := findOutdatedPods(deployment, existingPods)
	for idx, p := range outdatedPods {
		if idx >= numPodsToDelete {
			break
		}
		if err := m.deleteDeploymentPod(deployment, p); err != nil {
			glog.Errorf("DEPLOYMENT [%v]: failed to delete pod: %v", deployment.Name, err.Error())
			continue
		}
		numOutdatedPodsDeleted++
	}

	numPodsDeleted := 0
	for i := 0; i < api.Max(0, int(numPodsToDelete)-len(outdatedPods)); i++ {
		it := existingPods.Back()
		p := it.Value.(*core.Pod)
		if err := m.deleteDeploymentPod(deployment, p); err != nil {
			glog.Errorf("DEPLOYMENT [%v]: failed to delete pod: %v", deployment.Name, err.Error())
			continue
		}
		numPodsDeleted++
	}

	if err := etcd.Put(fmt.Sprintf("/Deployments/Meta/%s", deployment.Name), deployment); err != nil {
		glog.Errorf("failed to update deployment's metadata: %v", err)
	}
	if err := etcd.Put(fmt.Sprintf("/Deployments/Pods/%s", deployment.Name), core.GetPodNames(existingPods)); err != nil {
		glog.Errorf("failed to update deployment's corresponding pods: %v", err)
	}
	glog.Infof("DEPLOYMENT [%v]: expected to delete %v pods, actually deleted %v updated, %v outdated",
		deployment.Name,
		numPodsToDelete,
		numPodsDeleted,
		numOutdatedPodsDeleted)
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
		// Delete the deployment in etcd and memory.
		DeleteDeploymentInEtcd(name)
		m.componentManager.DeleteDeploymentByName(name)
		glog.Infof("DEPLOYMENT [%v]: deployment deleted", name)
	} else {
		return fmt.Errorf("no such deployment: %v", name)
	}

	return nil
}

func (m *basicController) DeleteAllDeployments() error {
	deployments := m.componentManager.ListDeployments()
	for _, deployment := range deployments {
		if err := m.DeleteDeploymentByName(deployment.Name); err != nil {
			return err
		}
	}
	return nil
}

func (m *basicController) HandleEvent(event apiserver.Event) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	var err error = nil

	switch event.Type() {
	case apiserver.PodDeletion:
		legacy := event.(*apiserver.PodDeletionEvent).PodLegacy
		var deploymentName string
		if legacy != nil {
			deploymentName = legacy.DeploymentName
		}
		m.handlePodDeletion(event.(*apiserver.PodDeletionEvent).Pod, deploymentName)
	case apiserver.PodReady:
		podName := event.(*apiserver.PodReadyEvent).PodName
		if m.componentManager.PodExistsByName(podName) {
			err = m.handlePodReady(m.componentManager.GetPodByName(podName))
		}
	case apiserver.PodFail:
		podName := event.(*apiserver.PodFailEvent).PodName
		pod := m.componentManager.GetPodByName(podName)
		if pod == nil {
			glog.Errorf("failed pod does not exist: %v", podName)
			return
		}
		if deployment := m.componentManager.GetDeploymentByPodName(podName); deployment != nil {
			m.deleteDeploymentPod(deployment, pod)
		}
	}

	if err != nil {
		glog.Error(err)
	}
}

func (m *basicController) handlePodDeletion(pod *core.Pod, deploymentName string) error {
	// Avoid updating pod status twice.
	if _, present := m.expectDeletedPod[pod.Name]; present {
		delete(m.expectDeletedPod, pod.Name)
		return nil
	}
	// If deployment is not found, then the pod must be deleted because its managing deployment is deleted.
	if deployment := m.componentManager.GetDeploymentByName(deploymentName); deployment != nil {
		updateDeploymentStatusOnPodRemoval(deployment, pod)
		if err := etcd.Put(fmt.Sprintf("/Deployments/Meta/%s", deploymentName), deployment); err != nil {
			return err
		}
	}
	return nil
}

func (m *basicController) handlePodReady(pod *core.Pod) error {
	if pod == nil {
		return fmt.Errorf("pod is nil")
	}
	// It's not likely that the number of pods exceed the deployment's desired number, the only case being when
	// the number of desired replicas is decreased by auto scaler or the user. In that case, applyDeployment will handle pod deletion.
	if deployment := m.componentManager.GetDeploymentByPodName(pod.Name); deployment != nil {
		deployment.Status.ReadyReplicas++
		// If a pod becomes ready but is not updated, it could only be that the deployment template has been chaged
		// during pod creation.
		if isPodUpdated(deployment, pod) {
			deployment.Status.UpdatedReplicas++
			if err := etcd.Put(fmt.Sprintf("/Deployments/Meta/%s", deployment.Name), deployment); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *basicController) monitorDeployment() {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	deployments := m.componentManager.ListDeployments()
	for _, deployment := range deployments {
		pods := m.componentManager.ListPodsByDeploymentName(deployment.Name)
		if pods == nil {
			glog.Errorf("DEPLOYMENT [%v]: nil pod list", deployment.Name)
			continue
		}
		// Only consider maxSurge and maxUnavailable when the deployment is under rolling update.
		if deployment.Status.UpdatedReplicas < deployment.Status.ReadyReplicas {
			// Pods might be created and deleted at the same time, so cannot reuse normal update logic.
			// Delete outdated pods.
			numPodDiff := computeRollingUpdatePodDeletion(deployment, pods)
			if numPodDiff > 0 {
				m.fewerPods(deployment, pods, numPodDiff)
			}

			numPodDiff = computeRollingUpdatePodCreation(deployment, pods)
			if numPodDiff > 0 {
				m.morePods(deployment, pods, numPodDiff)
			}
		} else {
			numPodDiff := int(deployment.Spec.Replicas) - int(deployment.Status.Replicas)
			if numPodDiff > 0 {
				m.morePods(deployment, pods, numPodDiff)
			} else if numPodDiff < 0 {
				m.fewerPods(deployment, pods, -numPodDiff)
			}
		}
	}
}

func (m *basicController) deleteDeploymentPod(deployment *core.Deployment, pod *core.Pod) error {
	if err := m.podController.DeletePodByName(pod.Name); err != nil {
		return err
	}

	// DeletePodByName will alter deployment's pod list, so no need to modify it here.
	m.expectDeletedPod[pod.Name] = struct{}{}
	updateDeploymentStatusOnPodRemoval(deployment, pod)
	glog.Infof("DEPLOYMENT [%v]: deleted pod [%v]", deployment.Name, pod.Name)

	return nil
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

// computeSpecHash, isDeploymentUpdated, updateDeploymentTemplate, isPodUpdated must be consistent.
func computeSpecHash(deployment *core.Deployment) string {
	type relevantSpec struct {
		labels  map[string]string
		podSpec core.PodSpec
	}
	return api.Hash(&relevantSpec{
		labels:  deployment.Spec.Template.ObjectMeta.Labels,
		podSpec: deployment.Spec.Template.Spec,
	})
}

func isDeploymentUpdated(d1 *core.Deployment, d2 *core.Deployment) bool {
	return api.Hash(d1.Spec.Template.Labels) == api.Hash(d2.Spec.Template.Labels) &&
		api.Hash(d1.Spec.Template.Spec) == api.Hash(d2.Spec.Template.Spec)
}

func updateDeploymentTemplate(existingDeployment *core.Deployment, newDeployment *core.Deployment) {
	existingDeployment.Spec.Template.ObjectMeta.Labels = newDeployment.Spec.Template.Labels
	existingDeployment.Spec.Template.Spec = newDeployment.Spec.Template.Spec
}

func isPodUpdated(deployment *core.Deployment, pod *core.Pod) bool {
	return api.Hash(deployment.Spec.Template.Labels) == api.Hash(pod.Labels) &&
		api.Hash(deployment.Spec.Template.Spec) == api.Hash(pod.Spec)
}

func updateDeploymentStatusOnPodRemoval(deployment *core.Deployment, pod *core.Pod) {
	// Deleting pod is presumably guaranteed to succeed.
	deployment.Status.Replicas--
	if isPodUpdated(deployment, pod) && (pod.Status.Phase == core.PodReady || pod.Status.Phase == core.PodFailed) {
		deployment.Status.UpdatedReplicas--
	}
	// Failed pods must be Ready previously.
	if pod.Status.Phase == core.PodReady || pod.Status.Phase == core.PodFailed {
		deployment.Status.ReadyReplicas--
	}
}

func setDeploymentInEtcd(name string, deployment *core.Deployment) error {
	return etcd.Put(fmt.Sprintf("/Deployments/Meta/%s", name), deployment)
}

func DeleteDeploymentInEtcd(deploymentName string) error {
	if err := etcd.Delete(fmt.Sprintf("/Deployments/Meta/%s", deploymentName)); err != nil {
		return err
	}
	if err := etcd.Delete(fmt.Sprintf("/Deployments/Pods/%s", deploymentName)); err != nil {
		return err
	}
	return nil
}

func computeRollingUpdatePodDeletion(deployment *core.Deployment, pods *list.List) int {
	var readyReplicas int64 = int64(deployment.Status.ReadyReplicas)
	var outdatedReplicas int64 = int64(len(findOutdatedPods(deployment, pods)))
	var replicas int64 = int64(deployment.Spec.Replicas)
	var maxUnavailable int64 = int64(deployment.Spec.RollingUpdate.MaxUnavailable)
	var minReadyReplicas int64 = api.Max64(0, replicas-maxUnavailable)
	return int(api.Min64(outdatedReplicas, api.Max64(0, readyReplicas-minReadyReplicas)))
}

func computeRollingUpdatePodCreation(deployment *core.Deployment, pods *list.List) int {
	var specReplicas int64 = int64(deployment.Spec.Replicas)
	var statusReplicas int64 = int64(deployment.Status.Replicas)
	var maxReplicas int64 = specReplicas + int64(deployment.Spec.RollingUpdate.MaxSurge)
	var allUpdatedReplicas int64 = int64(len(findUpdatedPods(deployment, pods)))
	var idealReplicasToCreate int64 = api.Max64(0, specReplicas-allUpdatedReplicas)
	return int(api.Min64(maxReplicas-statusReplicas, idealReplicasToCreate))
}

func findOutdatedPods(deployment *core.Deployment, pods *list.List) []*core.Pod {
	outdatedPods := make([]*core.Pod, 0)
	for i := pods.Front(); i != nil; i = i.Next() {
		pod := i.Value.(*core.Pod)
		if !isPodUpdated(deployment, pod) {
			outdatedPods = append(outdatedPods, pod)
		}
	}
	return outdatedPods
}

func findUpdatedPods(deployment *core.Deployment, pods *list.List) []*core.Pod {
	outdatedPods := make([]*core.Pod, 0)
	for i := pods.Front(); i != nil; i = i.Next() {
		pod := i.Value.(*core.Pod)
		if isPodUpdated(deployment, pod) {
			outdatedPods = append(outdatedPods, pod)
		}
	}
	return outdatedPods
}
