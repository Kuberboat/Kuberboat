package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"p9t.io/kuberboat/pkg/api/core"
	pb "p9t.io/kuberboat/pkg/proto"
)

var CONN_TIMEOUT time.Duration = time.Second
var APISERVER_URL string = "localhost"
var APISERVER_PORT uint16 = 6443

type ctlClient struct {
	connection *grpc.ClientConn
	client     pb.ApiServerCtlServiceClient
}

func NewCtlClient() *ctlClient {
	addr := fmt.Sprintf("%v:%v", APISERVER_URL, APISERVER_PORT)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Kubectl client failed to connect to api server")
	}
	return &ctlClient{
		connection: conn,
		client:     pb.NewApiServerCtlServiceClient(conn),
	}
}

func (c *ctlClient) CreatePod(pod *core.Pod) (*pb.CreatePodResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	data, err := json.Marshal(pod)
	if err != nil {
		return &pb.CreatePodResponse{Status: 1}, err
	}
	return c.client.CreatePod(ctx, &pb.CreatePodRequest{
		Pod: data,
	})
}

func (c *ctlClient) DeletePod(podName string) (*pb.DeletePodResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	return c.client.DeletePod(ctx, &pb.DeletePodRequest{
		PodName: podName,
	})
}

func (c *ctlClient) RegisterNode(node *core.Node) (*pb.RegisterNodeResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	data, err := json.Marshal(node)
	if err != nil {
		return &pb.RegisterNodeResponse{Status: 1}, err
	}
	return c.client.RegisterNode(ctx, &pb.RegisterNodeRequest{
		Node: data,
	})
}
