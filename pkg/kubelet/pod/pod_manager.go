package pod

import (
	"encoding/json"
	"github.com/golang/glog"
	"sync"

	"p9t.io/kuberboat/pkg/api/core"
)

// MetaManager defines public methods of a pod metadata manager.
type MetaManager interface {
	// Pods returns the pods bound to the kubelet and their spec.
	Pods() []*core.Pod
	// PodByName provides the (non-mirror) pod that matches namespace and
	// name, as well as whether the pod was found.
	PodByName(name string) (*core.Pod, bool)
	// AddPod adds the given pod to the manager.
	// Assumes the pod being added is always new.
	AddPod(pod *core.Pod)
	// DeletePodByName deletes the given pod indexed by name from the manager.
	// Assumes the pod being deleted always exists.
	DeletePodByName(name string)
}

// basicManager manages metadata of pods.
// All fields in PodManager are read-only and are updated calling AddPod or DeletePod.
type basicManager struct {
	mtx sync.RWMutex
	// Pods indexed by name for easy access.
	podByName map[string]*core.Pod
}

// NewMetaManager returns a pod meta data manager.
func NewMetaManager() MetaManager {
	return &basicManager{
		podByName: map[string]*core.Pod{},
	}
}

func (pm *basicManager) Pods() []*core.Pod {
	pods := make([]*core.Pod, 0, len(pm.podByName))
	pm.mtx.RLock()
	defer pm.mtx.RUnlock()
	for _, pod := range pm.podByName {
		pods = append(pods, pod)
	}
	return pods
}

func (pm *basicManager) PodByName(name string) (*core.Pod, bool) {
	pm.mtx.RLock()
	defer pm.mtx.RUnlock()
	pod, ok := pm.podByName[name]
	return pod, ok
}

func (pm *basicManager) AddPod(pod *core.Pod) {
	pm.mtx.Lock()
	defer pm.mtx.Unlock()
	if _, ok := pm.podByName[pod.Name]; ok {
		glog.Errorf("pod already exists: %v", pod.Name)
		return
	}
	pm.podByName[pod.Name] = pod
}

func (pm *basicManager) DeletePodByName(name string) {
	pm.mtx.Lock()
	defer pm.mtx.Unlock()
	if _, ok := pm.podByName[name]; !ok {
		glog.Errorf("pod does not exist: %v", name)
		return
	}
	delete(pm.podByName, name)
}

// RuntimeManager manages the container runtime resources that a pod uses.
type RuntimeManager interface {
	// AddPodContainer records a container as a member of a pod.
	AddPodContainer(pod *core.Pod, name string)
	// AddPodVolume records a volume as being used by a pod.
	AddPodVolume(pod *core.Pod, name string)
	// ContainersByPod returns all the containers created by a pod.
	ContainersByPod(pod *core.Pod) ([]string, bool)
	// VolumesByPod returns all the volumes created by a pod.
	VolumesByPod(pod *core.Pod) ([]string, bool)
	// StringifyPodResources returns a human-readable representation of pod run time resources.
	StringifyPodResources(pod *core.Pod) string
}

// dockerRuntimeManager manages docker resources that a pod uses.
type dockerRuntimeManager struct {
	mtx sync.RWMutex
	// Docker container IDs indexed by pod.
	// ContainerCreate only returns the ID, so that is what will be stored.
	// Does not contain pause container.
	containersByPod map[*core.Pod][]string
	// Docker volumes indexed by pod.
	// We only need the name to manipulate the volume.
	volumesByPod map[*core.Pod][]string
}

func NewRuntimeManager() RuntimeManager {
	return &dockerRuntimeManager{
		containersByPod: map[*core.Pod][]string{},
		volumesByPod:    map[*core.Pod][]string{},
	}
}

func (rm *dockerRuntimeManager) AddPodContainer(pod *core.Pod, name string) {
	rm.mtx.Lock()
	defer rm.mtx.Unlock()
	rm.containersByPod[pod] = append(rm.containersByPod[pod], name)
}

func (rm *dockerRuntimeManager) AddPodVolume(pod *core.Pod, name string) {
	rm.mtx.Lock()
	defer rm.mtx.Unlock()
	rm.volumesByPod[pod] = append(rm.volumesByPod[pod], name)
}

func (rm *dockerRuntimeManager) ContainersByPod(pod *core.Pod) ([]string, bool) {
	rm.mtx.RLock()
	defer rm.mtx.RUnlock()
	c, ok := rm.containersByPod[pod]
	return c, ok
}

func (rm *dockerRuntimeManager) VolumesByPod(pod *core.Pod) ([]string, bool) {
	rm.mtx.RLock()
	defer rm.mtx.RUnlock()
	v, ok := rm.volumesByPod[pod]
	return v, ok
}

func (rm *dockerRuntimeManager) StringifyPodResources(pod *core.Pod) string {
	rm.mtx.RLock()
	c, _ := rm.ContainersByPod(pod)
	v, _ := rm.VolumesByPod(pod)
	rm.mtx.RUnlock()

	var str = ""

	cStr, _ := json.Marshal(c)
	str += "Containers: " + string(cStr) + "\n"

	vStr, _ := json.Marshal(v)
	str += "Volumes: " + string(vStr)

	return str
}
