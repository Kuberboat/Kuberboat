package client

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"p9t.io/kuberboat/cmd/apiserver/app"
	pb "p9t.io/kuberboat/pkg/proto"
)

// FIXME: Move this into config file.
const CONN_TIMEOUT time.Duration = time.Second

type kubeletClient struct {
	connection *grpc.ClientConn
	client     pb.ApiServerKubeletServiceClient
}

func NewKubeletClient() *kubeletClient {
	addr := fmt.Sprint("localhost:", app.APISERVER_PORT)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		glog.Fatal("Kubelet client failed to connect to api server")
	}
	return &kubeletClient{
		connection: conn,
		client:     pb.NewApiServerKubeletServiceClient(conn),
	}
}

func (c *kubeletClient) RegisterNode(node *pb.Node) (*pb.RegisterNodeResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	return c.client.RegisterNode(ctx, &pb.RegisterNodeRequest{})
}
