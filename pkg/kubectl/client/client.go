package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"p9t.io/kuberboat/cmd/apiserver/app"
	"p9t.io/kuberboat/pkg/api/core"
	pb "p9t.io/kuberboat/pkg/proto"
)

// FIXME: Move this into config file.
const CONN_TIMEOUT time.Duration = time.Second

type ctlClient struct {
	connection *grpc.ClientConn
	client     pb.ApiServerCtlServiceClient
}

func NewCtlClient() *ctlClient {
	addr := fmt.Sprint("localhost:", app.APISERVER_PORT)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		glog.Fatal("Kubectl client failed to connect to api server")
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
