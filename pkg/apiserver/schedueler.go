package apiserver

import (
	"p9t.io/kuberboat/pkg/api/core"
)

// PodScheduler selects a node to create and run a pod.
type PodScheduler interface {
	// SchedulePod
	SchedulePod(pod *core.Pod) *core.Node
	// Schedulable tells if there is any node available for scheduling.
	Schedulable() bool
}

type RRPodScheduler struct {
	// nodeManager provides information about nodes.
	nodeManager NodeManager
	// Index of the next node to schedule.
	nextIdx int
}

func (s *RRPodScheduler) SchedulePod(pod *core.Pod) *core.Node {
	nodes := s.nodeManager.RegisteredNodes()
	if s.nextIdx >= len(nodes) {
		s.nextIdx = 0
	}
	if len(nodes) == 0 {
		return nil
	} else {
		ret := nodes[s.nextIdx]
		s.nextIdx++
		return ret
	}
}

func (s *RRPodScheduler) Schedulable() bool {
	return s.nodeManager.Empty()
}

// NewPodScheduler returns a new PodScheduler object.
func NewPodScheduler(nm NodeManager) PodScheduler {
	return &RRPodScheduler{nodeManager: nm, nextIdx: 0}
}
