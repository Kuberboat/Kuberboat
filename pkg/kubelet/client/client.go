package client

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"p9t.io/kuberboat/pkg/apiserver"
	pb "p9t.io/kuberboat/pkg/proto"
)

// FIXME: Move this into config file.
const CONN_TIMEOUT time.Duration = time.Second

type kubeletClient struct {
	connection *grpc.ClientConn
	client     pb.ApiServerKubeletServiceClient
}

func NewKubeletClient() *kubeletClient {
	addr := fmt.Sprint("localhost:", apiserver.APISERVER_PORT)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		glog.Fatal("Kubelet client failed to connect to api server")
	}
	return &kubeletClient{
		connection: conn,
		client:     pb.NewApiServerKubeletServiceClient(conn),
	}
}
