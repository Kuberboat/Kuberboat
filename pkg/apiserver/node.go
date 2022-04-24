package apiserver

import (
	"errors"
	"fmt"

	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver/client"
)

const APISERVER_PORT uint16 = 6443

type NodeWithClient struct {
	node   *core.Node
	client *client.ApiserverClient
}

type NodeManager interface {
	// RegisterNode adds metadata for the node and creates grpc client to Kubelet and Kubeproxy to that node.
	RegisterNode(node *core.Node) error
	// UnregisterNode is for rolling back registeration.
	UnregisterNode(name string) error
	// RegisterNodes returns all the node registered.
	RegisteredNodes() []*core.Node
	// ClientByName returns the grpc client indexed by node name.
	ClientByName(name string) *client.ApiserverClient
	// ClientByIP returns grpc client indexed by worker IP.
	ClientByIP(ip string) *client.ApiserverClient
	// Clients returns all the grpc clients for workers in the cluster.
	Clients() []*client.ApiserverClient
	// Empty returns true if no node is registered.
	Empty() bool
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
		node:   node,
		client: client,
	}
	return nil
}

func (nm *nodeManagerInner) UnregisterNode(name string) error {
	if _, ok := nm.nodes[name]; ok {
		delete(nm.nodes, name)
		return nil
	} else {
		return fmt.Errorf("no such node: %v", name)
	}
}

func (nm *nodeManagerInner) RegisteredNodes() []*core.Node {
	registeredNodes := make([]*core.Node, 0, len(nm.nodes))
	for _, nodeWithClient := range nm.nodes {
		registeredNodes = append(registeredNodes, nodeWithClient.node)
	}
	return registeredNodes
}

func (nm *nodeManagerInner) ClientByName(name string) *client.ApiserverClient {
	if nodeWithClient, ok := nm.nodes[name]; ok {
		return nodeWithClient.client
	}
	return nil
}

func (nm *nodeManagerInner) ClientByIP(ip string) *client.ApiserverClient {
	for _, node := range nm.nodes {
		if node.node.Status.Address == ip {
			return node.client
		}
	}
	return nil
}

func (nm *nodeManagerInner) Clients() []*client.ApiserverClient {
	clients := make([]*client.ApiserverClient, 0, len(nm.nodes))
	for _, nodeWithClient := range nm.nodes {
		clients = append(clients, nodeWithClient.client)
	}
	return clients
}

func (nm *nodeManagerInner) Empty() bool {
	return len(nm.nodes) == 0
}
