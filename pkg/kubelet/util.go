package kubelet

import (
	"fmt"

	"p9t.io/kuberboat/pkg/api/core"
)

// GetPodSpecificName prepends the name of any resource, be it container, volume
// or whatever with the UUID of the pod.
func GetPodSpecificName(pod *core.Pod, name string) string {
	return fmt.Sprintf("%v_%v", pod.UUID.String(), name)
}
