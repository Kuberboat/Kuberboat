package schedule

import (
	"fmt"

	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver"
	"p9t.io/kuberboat/pkg/apiserver/node"
)

// PodScheduler selects a node to create and run a pod.
type PodScheduler interface {
	// SchedulePod schedules a pod by round robin. If an affinity pod is specified, the pod will be
	// scheduled to the node where its affinity pod has been scheduled.
	SchedulePod(pod *core.Pod) (*core.Node, error)
}

// NewPodScheduler returns a new PodScheduler object.
func NewPodScheduler(
	nodeManager node.NodeManager,
	componentManager apiserver.ComponentManager,
) PodScheduler {
	return &schedulerInner{
		nodeManager:      nodeManager,
		componentManager: componentManager,
		nextIdx:          0,
	}
}

type schedulerInner struct {
	// nodeManager provides information about nodes.
	nodeManager node.NodeManager
	// componentManager stores the components and the dependencies between them.
	componentManager apiserver.ComponentManager
	// nextIdx is the index of the next node to schedule.
	nextIdx int
}

func (s *schedulerInner) SchedulePod(pod *core.Pod) (*core.Node, error) {
	if pod.Spec.Affinity != "" {
		return s.scheduleByAffinity(pod)
	} else {
		return s.scheduleByRoundRobin(pod), nil
	}
}

// scheduleByAffinity schedules a pod to where its affinity pod has been scheduled.
func (s *schedulerInner) scheduleByAffinity(pod *core.Pod) (*core.Node, error) {
	affinityPodName := pod.Spec.Affinity
	if !s.componentManager.PodExistsByName(affinityPodName) {
		return nil, fmt.Errorf(
			"affinity pod %s for pod %s does not exist",
			affinityPodName,
			pod.Name,
		)
	}
	affinityPod := s.componentManager.GetPodByName(affinityPodName)
	if affinityPod.Status.HostIP == "" {
		return nil, fmt.Errorf(
			"fail to fetch ip address of affinity pod %s for pod %s",
			affinityPodName,
			pod.Name,
		)
	}
	return s.nodeManager.NodeByIP(affinityPod.Status.HostIP), nil
}

// scheduleByRoundRobin schedules a pod by round robin.
func (s *schedulerInner) scheduleByRoundRobin(pod *core.Pod) *core.Node {
	nodes := s.nodeManager.RegisteredNodes()
	if s.nextIdx >= len(nodes) {
		s.nextIdx = 0
	}
	if len(nodes) == 0 {
		return nil
	} else {
		node := nodes[s.nextIdx]
		s.nextIdx++
		return node
	}
}
