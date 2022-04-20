package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"p9t.io/kuberboat/pkg/api"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver"
	"p9t.io/kuberboat/pkg/kubelet"
	pb "p9t.io/kuberboat/pkg/proto"
)

// FIXME: Move this into config file.
const APISERVER_PORT uint16 = 6443

var nodeManager = apiserver.NewNodeManager()

type server struct {
	pb.UnimplementedApiServerKubeletServiceServer
	pb.UnimplementedApiServerCtlServiceServer
}

func (s *server) CreatePod(ctx context.Context, req *pb.CreatePodRequest) (*pb.CreatePodResponse, error) {
	var pod core.Pod
	err := json.Unmarshal(req.Pod, &pod)
	if err != nil {
		return &pb.CreatePodResponse{Status: -1}, err
	}
	glog.Infof("server got %#v\n", pod)
	// TODO: Create pod logic
	return &pb.CreatePodResponse{Status: 0}, nil
}

func (s *server) DeletePod(ctx context.Context, req *pb.DeletePodRequest) (*pb.DeletePodResponse, error) {
	// TODO: Delete pod logic
	return &pb.DeletePodResponse{Status: 0}, nil
}

func (s *server) RegisterNode(ctx context.Context, req *pb.RegisterNodeRequest) (*pb.RegisterNodeResponse, error) {
	var node core.Node
	err := json.Unmarshal(req.Node, &node)
	if err != nil {
		return &pb.RegisterNodeResponse{Status: -1}, err
	}
	if err := registerNode(ctx, &node); err != nil {
		return &pb.RegisterNodeResponse{Status: -1}, err
	} else {
		return &pb.RegisterNodeResponse{Status: 0}, nil
	}
}

func registerNode(ctx context.Context, node *core.Node) error {
	// Get node address.
	p, _ := peer.FromContext(ctx)
	workerIP := strings.Split(p.Addr.String(), ":")[0]

	node.CreationTimestamp = time.Now()
	node.UUID = uuid.New()
	node.Status.Phase = core.NodePending
	node.Status.Port = kubelet.Port
	node.Status.Address = workerIP
	node.Status.Condition = core.NodeUnavailable

	if err := nodeManager.RegisterNode(node); err != nil {
		glog.Error(err.Error())
		return err
	}

	client := nodeManager.ClientByName(node.Name)
	r, err := client.NotifyRegistered(&core.Cluster{
		Server: os.Getenv(api.ApiServerIP),
		Port:   APISERVER_PORT,
	})
	// If failed to notify worker, rollback registration.
	if err != nil || r.Status != 0 {
		glog.Errorf("cannot notify worker")
		nodeManager.UnregisterNode(node.Name)
		return err
	}

	node.Status.Phase = core.NodeRunning
	node.Status.Condition = core.NodeReady

	return nil
}

func StartServer() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", APISERVER_PORT))
	if err != nil {
		glog.Fatal("Api server failed to connect!")
	}

	apiServer := grpc.NewServer()
	pb.RegisterApiServerCtlServiceServer(apiServer, &server{})
	pb.RegisterApiServerKubeletServiceServer(apiServer, &server{})

	glog.Infof("Api server listening at %v", lis.Addr())

	if err := apiServer.Serve(lis); err != nil {
		glog.Fatal(err)
	}
}
