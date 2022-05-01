package apiserver

import (
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver/client"
	"p9t.io/kuberboat/pkg/apiserver/node"
)

// PodScheduler selects a node to create and run a pod.
type PodScheduler interface {
	// SchedulePod
	SchedulePod(pod *core.Pod) (*core.Node, *client.ApiserverClient)
}

type RRPodScheduler struct {
	// nodeManager provides information about nodes.
	nodeManager node.NodeManager
	// Index of the next node to schedule.
	nextIdx int
}

func (s *RRPodScheduler) SchedulePod(pod *core.Pod) (*core.Node, *client.ApiserverClient) {
	nodes := s.nodeManager.RegisteredNodes()
	if s.nextIdx >= len(nodes) {
		s.nextIdx = 0
	}
	if len(nodes) == 0 {
		return nil, nil
	} else {
		node := nodes[s.nextIdx]
		client := s.nodeManager.ClientByName(node.Name)
		s.nextIdx++
		return node, client
	}
}

// NewPodScheduler returns a new PodScheduler object.
func NewPodScheduler(nm node.NodeManager) PodScheduler {
	return &RRPodScheduler{nodeManager: nm, nextIdx: 0}
}
