package core

import (
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
