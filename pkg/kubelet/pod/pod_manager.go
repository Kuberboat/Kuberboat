package pod

import (
	"encoding/json"
	"sync"

	"p9t.io/kuberboat/pkg/api/core"
)

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
