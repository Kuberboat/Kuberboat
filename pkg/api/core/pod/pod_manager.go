package pod

import (
	"sync"

	"github.com/golang/glog"
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
