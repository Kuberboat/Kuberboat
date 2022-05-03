package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver"
	"p9t.io/kuberboat/pkg/apiserver/deployment"
	"p9t.io/kuberboat/pkg/apiserver/dns"
	"p9t.io/kuberboat/pkg/apiserver/etcd"
	"p9t.io/kuberboat/pkg/apiserver/metrics"
	"p9t.io/kuberboat/pkg/apiserver/node"
	"p9t.io/kuberboat/pkg/apiserver/pod"
	"p9t.io/kuberboat/pkg/apiserver/schedule"
	"p9t.io/kuberboat/pkg/apiserver/service"
	pb "p9t.io/kuberboat/pkg/proto"
)

// FIXME: Move the managers and controllers into a wrapper.
var nodeManager node.NodeManager
var componentManager apiserver.ComponentManager
var legacyManager apiserver.LegacyManager
var podScheduler schedule.PodScheduler
var podController pod.Controller
var serviceController service.Controller
var deploymentController deployment.Contoller
var nodeController node.Controller
var metricsManager metrics.MetricsManager
var dnsController dns.Controller

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
	if err := nodeController.RegisterNode(ctx, &node); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	} else {
		return &pb.DefaultResponse{Status: 0}, nil
	}
}

func (s *server) CreateDeployment(ctx context.Context, req *pb.CreateDeploymentRequest) (*pb.DefaultResponse, error) {
	var deployment core.Deployment
	if err := json.Unmarshal(req.Deployment, &deployment); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	if err := deploymentController.ApplyDeployment(&deployment); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	return &pb.DefaultResponse{Status: 0}, nil
}

func (s *server) DeleteDeployment(ctx context.Context, req *pb.DeleteDeploymentRequest) (*pb.DefaultResponse, error) {
	if req.DeploymentName == "" {
		if err := deploymentController.DeleteAllDeployments(); err != nil {
			return &pb.DefaultResponse{Status: -1}, err
		}
	} else {
		if err := deploymentController.DeleteDeploymentByName(req.DeploymentName); err != nil {
			return &pb.DefaultResponse{Status: -1}, err
		}
	}
	return &pb.DefaultResponse{Status: 0}, nil
}

func (s *server) UpdatePodStatus(ctx context.Context, req *pb.UpdatePodStatusRequest) (*pb.DefaultResponse, error) {
	var status core.PodStatus
	if err := json.Unmarshal(req.PodStatus, &status); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	prevStatus, err := podController.UpdatePodStatus(req.PodName, &status)
	if err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}

	// Try to dispatch PodReadyEvent.
	if prevStatus.Phase != core.PodReady && status.Phase == core.PodReady {
		apiserver.Dispatch(&apiserver.PodReadyEvent{PodName: req.PodName})
	}
	// Try to dispatch PodFailEvent.
	if prevStatus.Phase != core.PodFailed && status.Phase == core.PodFailed {
		apiserver.Dispatch(&apiserver.PodFailEvent{PodName: req.PodName})
	}

	return &pb.DefaultResponse{Status: 0}, nil
}

func (*server) NotifyPodDeletion(ctx context.Context, req *pb.NotifyPodDeletionRequest) (*pb.DefaultResponse, error) {
	var deletedPod core.Pod
	if err := json.Unmarshal(req.DeletedPod, &deletedPod); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	legacy := legacyManager.GetPodLegacyByName(deletedPod.Name)
	apiserver.Dispatch(&apiserver.PodDeletionEvent{Pod: &deletedPod, PodLegacy: legacy})
	legacyManager.DeletePodLegacyByName(deletedPod.Name)
	return &pb.DefaultResponse{Status: 0}, nil
}

func (*server) CreateService(ctx context.Context, req *pb.CreateServiceRequest) (*pb.DefaultResponse, error) {
	var service core.Service
	if err := json.Unmarshal(req.Service, &service); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	if err := serviceController.CreateService(&service); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	return &pb.DefaultResponse{Status: 0}, nil
}

func (*server) DeleteService(ctx context.Context, req *pb.DeleteServiceRequest) (*pb.DefaultResponse, error) {
	if req.ServiceName == "" {
		if err := serviceController.DeleteAllServices(); err != nil {
			return &pb.DefaultResponse{Status: -1}, err
		}
	} else {
		if err := serviceController.DeleteServiceByName(req.ServiceName); err != nil {
			return &pb.DefaultResponse{Status: -1}, err
		}
	}

	return &pb.DefaultResponse{Status: 0}, nil
}

func (*server) DescribeDeployments(ctx context.Context, req *pb.DescribeDeploymentsRequest) (*pb.DescribeDeploymentsResponse, error) {
	foundDeployments, deploymentPods, notFoundDeployments := deploymentController.DescribeDeployments(req.All, req.DeploymentNames)
	serializeErrResponse := &pb.DescribeDeploymentsResponse{
		Status:             -1,
		Deployments:        nil,
		DeploymentPodNames: nil,
	}

	foundDeploymentsData, err := json.Marshal(foundDeployments)
	if err != nil {
		return serializeErrResponse, err
	}

	deploymentPodsData, err := json.Marshal(deploymentPods)
	if err != nil {
		return serializeErrResponse, err
	}

	notFoundDeploymentsData, err := json.Marshal(notFoundDeployments)
	if err != nil {
		return serializeErrResponse, err
	}

	var status int32
	if len(notFoundDeployments) > 0 {
		status = -2
	} else {
		status = 0
	}

	return &pb.DescribeDeploymentsResponse{
		Status:              status,
		Deployments:         foundDeploymentsData,
		DeploymentPodNames:  deploymentPodsData,
		NotFoundDeployments: notFoundDeploymentsData,
	}, nil
}

func (*server) DescribeServices(ctx context.Context, req *pb.DescribeServicesRequest) (*pb.DescribeServicesResponse, error) {
	foundServices, servicePods, notFoundServices := serviceController.DescribeServices(req.All, req.ServiceNames)
	serializeErrResponse := &pb.DescribeServicesResponse{
		Status:          -1,
		Services:        nil,
		ServicePodNames: nil,
	}

	foundServicesData, err := json.Marshal(foundServices)
	if err != nil {
		return serializeErrResponse, err
	}

	servicePodsData, err := json.Marshal(servicePods)
	if err != nil {
		return serializeErrResponse, err
	}

	notFoundServicesData, err := json.Marshal(notFoundServices)
	if err != nil {
		return serializeErrResponse, err
	}

	var status int32
	if len(notFoundServices) > 0 {
		status = -2
	} else {
		status = 0
	}

	return &pb.DescribeServicesResponse{
		Status:           status,
		Services:         foundServicesData,
		ServicePodNames:  servicePodsData,
		NotFoundServices: notFoundServicesData,
	}, nil
}

func (*server) CreateDNS(ctx context.Context, req *pb.CreateDNSRequest) (*pb.DefaultResponse, error) {
	var dns core.DNS
	if err := json.Unmarshal(req.Dns, &dns); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	if err := dnsController.CreateDNS(&dns); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	return &pb.DefaultResponse{Status: 0}, nil
}

func StartServer(etcdServers string) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", core.APISERVER_PORT))
	if err != nil {
		glog.Fatal("Api server failed to connect!")
	}
	err = etcd.InitializeClient(etcdServers)
	if err != nil {
		glog.Fatal(err)
	}

	nodeManager = node.NewNodeManager()
	componentManager = apiserver.NewComponentManager()
	legacyManager = apiserver.NewLegacyManager(componentManager)
	podScheduler = schedule.NewPodScheduler(nodeManager)
	podController = pod.NewPodController(componentManager, podScheduler, nodeManager, legacyManager)
	serviceController, _ = service.NewServiceController(componentManager, nodeManager)
	deploymentController = deployment.NewDeploymentController(componentManager, podController)
	nodeController = node.NewNodeController(nodeManager)
	metricsManager, _ = metrics.NewMetricsManager(componentManager)
	dnsController = dns.NewDNSController(componentManager)

	apiServer := grpc.NewServer()
	pb.RegisterApiServerCtlServiceServer(apiServer, &server{})
	pb.RegisterApiServerKubeletServiceServer(apiServer, &server{})

	glog.Infof("Api server listening at %v", lis.Addr())

	// Empty prometheus target file
	metrics.GeneratePrometheusTargets([]*core.Node{})

	go metricsManager.StartMonitor()

	if err := apiServer.Serve(lis); err != nil {
		glog.Fatal(err)
	}
}
