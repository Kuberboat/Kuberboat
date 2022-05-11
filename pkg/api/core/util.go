package core

import (
	"container/list"
	"fmt"
)

// GetPodSpecificName prepends the name of any resource, be it container, volume
// or whatever with the UUID of the pod.
func GetPodSpecificName(pod *Pod, name string) string {
	return fmt.Sprintf("%v_%v", pod.UUID.String(), name)
}

func GetPodSpecificPauseName(pod *Pod) string {
	return GetPodSpecificName(pod, "pause")
}

func GetPodNames(pods *list.List) []string {
	podNames := make([]string, 0, pods.Len())
	for e := pods.Front(); e != nil; e = e.Next() {
		podNames = append(podNames, e.Value.(*Pod).Name)
	}
	return podNames
}
