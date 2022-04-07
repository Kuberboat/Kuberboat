package client

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"p9t.io/kuberboat/cmd/apiserver/app"
	pb "p9t.io/kuberboat/pkg/proto"
)

// FIXME: Move this into config file.
const CONN_TIMEOUT time.Duration = time.Second

type ctlClient struct {
	connection *grpc.ClientConn
	client     pb.ApiServerServiceClient
}

func NewCtlClient() *ctlClient {
	addr := fmt.Sprint("localhost:", app.APISERVER_PORT)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Kubectl client failed to connect to api server")
	}
	return &ctlClient{
		connection: conn,
		client:     pb.NewApiServerServiceClient(conn),
	}
}

func (c *ctlClient) CreatePod(pod *pb.Pod) (*pb.CreatePodResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	return c.client.CreatePod(ctx, &pb.CreatePodRequest{
		Pod: pod,
	})
}

func (c *ctlClient) DeletePod(podName string) (*pb.DeletePodResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	return c.client.DeletePod(ctx, &pb.DeletePodRequest{
		PodName: podName,
	})
}
