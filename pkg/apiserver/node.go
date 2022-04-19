package apiserver

import (
	"errors"

	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver/client"
)

type NodeWithClient struct {
	node   core.NodeStatus
	client *client.ApiserverClient
}

type NodeManager interface {
	RegisterNode(node *core.Node) error
	RegisteredNodes() []*NodeWithClient
}

type nodeManagerInner struct {
	nodes map[string]*NodeWithClient
}

func NewNodeManager() NodeManager {
	return &nodeManagerInner{
		nodes: make(map[string]*NodeWithClient),
	}
}

func (nm *nodeManagerInner) RegisterNode(node *core.Node) error {
	if nm.nodes[node.Name] != nil {
		return errors.New("duplicate node name")
	}
	client, err := client.NewCtlClient(node.Status.Address, node.Status.Port)
	if err != nil {
		return err
	}
	nm.nodes[node.Name] = &NodeWithClient{
		node:   node.Status,
		client: client,
	}
	return nil
}

func (nm *nodeManagerInner) RegisteredNodes() []*NodeWithClient {
	registeredNodes := make([]*NodeWithClient, 0, len(nm.nodes))
	for _, nodeWithClient := range nm.nodes {
		registeredNodes = append(registeredNodes, nodeWithClient)
	}
	return registeredNodes
}
