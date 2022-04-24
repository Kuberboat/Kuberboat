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
	"p9t.io/kuberboat/pkg/apiserver/pod"
	"p9t.io/kuberboat/pkg/apiserver/service"
	"p9t.io/kuberboat/pkg/kubelet"
	pb "p9t.io/kuberboat/pkg/proto"
)

const APISERVER_PORT uint16 = 6443

// FIXME: Move the managers and controllers into a wrapper.
var nodeManager = apiserver.NewNodeManager()
var componentManager = apiserver.NewComponentManager()
var podScheduler = apiserver.NewPodScheduler(nodeManager)
var podController = pod.NewPodController(componentManager, podScheduler, nodeManager)
var serviceController, err = service.NewServiceController(componentManager, nodeManager)

type server struct {
	pb.UnimplementedApiServerKubeletServiceServer
	pb.UnimplementedApiServerCtlServiceServer
}

func (s *server) GetPods(ctx context.Context, req *pb.GetPodsRequest) (*pb.GetPodsResponse, error) {
	foundPods, notFoundPods := podController.GetPods(req.All, req.PodNames)

	foundPodsData, err := json.Marshal(foundPods)
	if err != nil {
		return &pb.GetPodsResponse{
			Status:       -1,
			Pods:         nil,
			NotFoundPods: nil,
		}, err
	}

	notFoundPodsData, err := json.Marshal(notFoundPods)
	if err != nil {
		return &pb.GetPodsResponse{
			Status:       -1,
			Pods:         nil,
			NotFoundPods: nil,
		}, err
	}

	var status int32
	if len(notFoundPods) > 0 {
		status = -2
	} else {
		status = 0
	}

	return &pb.GetPodsResponse{
		Status:       status,
		Pods:         foundPodsData,
		NotFoundPods: notFoundPodsData,
	}, nil
}

func (s *server) CreatePod(ctx context.Context, req *pb.CreatePodRequest) (*pb.DefaultResponse, error) {
	var pod core.Pod
	if err := json.Unmarshal(req.Pod, &pod); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}

	if err := podController.CreatePod(&pod); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	return &pb.DefaultResponse{Status: 0}, nil
}

func (s *server) DeletePod(ctx context.Context, req *pb.DeletePodRequest) (*pb.DefaultResponse, error) {
	if req.PodName == "" {
		if err := podController.DeleteAllPods(); err != nil {
			return &pb.DefaultResponse{Status: -1}, err
		}
	} else {
		if err := podController.DeletePodByName(req.PodName); err != nil {
			return &pb.DefaultResponse{Status: -1}, err
		}
	}

	return &pb.DefaultResponse{Status: 0}, nil
}

func (s *server) RegisterNode(ctx context.Context, req *pb.RegisterNodeRequest) (*pb.DefaultResponse, error) {
	var node core.Node
	if err := json.Unmarshal(req.Node, &node); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	if err := registerNode(ctx, &node); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	} else {
		return &pb.DefaultResponse{Status: 0}, nil
	}
}

func registerNode(ctx context.Context, node *core.Node) error {
	// Get node address.
	var workerIP string
	p, _ := peer.FromContext(ctx)
	workerAddr := p.Addr.String()
	if strings.Count(workerAddr, ":") < 2 {
		// IPv4 address
		workerIP = strings.Split(p.Addr.String(), ":")[0]
	} else {
		// IPv6 address
		workerIP = workerAddr[0:strings.LastIndex(workerAddr, ":")]
	}

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
	r, err := client.NotifyRegistered(&core.ApiserverStatus{
		IP:   os.Getenv(api.ApiServerIP),
		Port: apiserver.APISERVER_PORT,
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

func (s *server) UpdatePodStatus(ctx context.Context, req *pb.UpdatePodStatusRequest) (*pb.DefaultResponse, error) {
	var status core.PodStatus
	if err := json.Unmarshal(req.PodStatus, &status); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	if err := podController.UpdatePodStatus(req.PodName, &status); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	return &pb.DefaultResponse{Status: 0}, nil
}

func (*server) CreateService(ctx context.Context, req *pb.CreateServiceRequest) (*pb.DefaultResponse, error) {
	var service core.Service
	if err := json.Unmarshal(req.Service, &service); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	if err := serviceController.CreateService(&service); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	} else {
		return &pb.DefaultResponse{Status: 0}, nil
	}
}

func StartServer() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", apiserver.APISERVER_PORT))
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
