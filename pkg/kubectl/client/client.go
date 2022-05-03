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
var APISERVER_PORT uint16 = core.APISERVER_PORT

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

func (c *ctlClient) GetPods(all bool, names []string) (*pb.GetPodsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	return c.client.GetPods(ctx, &pb.GetPodsRequest{
		All:      all,
		PodNames: names,
	})
}

func (c *ctlClient) CreatePod(pod *core.Pod) (*pb.DefaultResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	data, err := json.Marshal(pod)
	if err != nil {
		return &pb.DefaultResponse{Status: 1}, err
	}
	return c.client.CreatePod(ctx, &pb.CreatePodRequest{
		Pod: data,
	})
}

func (c *ctlClient) DeletePod(podName string) (*pb.DefaultResponse, error) {
	// We use an empty string to represent all pods.
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	return c.client.DeletePod(ctx, &pb.DeletePodRequest{
		PodName: podName,
	})
}

func (c *ctlClient) RegisterNode(node *core.Node) (*pb.DefaultResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	data, err := json.Marshal(node)
	if err != nil {
		return &pb.DefaultResponse{Status: 1}, err
	}
	return c.client.RegisterNode(ctx, &pb.RegisterNodeRequest{
		Node: data,
	})
}

func (c *ctlClient) UnregisterNode(nodeName string) (*pb.DefaultResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	return c.client.UnregisterNode(ctx, &pb.UnregisterNodeRequest{
		NodeName: nodeName,
	})
}

func (c *ctlClient) CreateDeployment(deployment *core.Deployment) (*pb.DefaultResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	data, err := json.Marshal(deployment)
	if err != nil {
		return &pb.DefaultResponse{Status: 1}, err
	}
	return c.client.CreateDeployment(ctx, &pb.CreateDeploymentRequest{
		Deployment: data,
	})
}

func (c *ctlClient) DeleteDeployment(deploymentName string) (*pb.DefaultResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	return c.client.DeleteDeployment(ctx, &pb.DeleteDeploymentRequest{
		DeploymentName: deploymentName,
	})
}

func (c *ctlClient) CreateService(service *core.Service) (*pb.DefaultResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	data, err := json.Marshal(service)
	if err != nil {
		return &pb.DefaultResponse{Status: 1}, err
	}
	return c.client.CreateService(ctx, &pb.CreateServiceRequest{
		Service: data,
	})
}

func (c *ctlClient) DeleteService(serviceName string) (*pb.DefaultResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	return c.client.DeleteService(ctx, &pb.DeleteServiceRequest{
		ServiceName: serviceName,
	})
}

func (c *ctlClient) DescribeDeployments(all bool, names []string) (*pb.DescribeDeploymentsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	return c.client.DescribeDeployments(ctx, &pb.DescribeDeploymentsRequest{
		All:             all,
		DeploymentNames: names,
	})
}

func (c *ctlClient) DescribeServices(all bool, names []string) (*pb.DescribeServicesResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	return c.client.DescribeServices(ctx, &pb.DescribeServicesRequest{
		All:          all,
		ServiceNames: names,
	})
}
