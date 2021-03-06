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
	"p9t.io/kuberboat/pkg/apiserver/job"
	"p9t.io/kuberboat/pkg/apiserver/node"
	"p9t.io/kuberboat/pkg/apiserver/pod"
	"p9t.io/kuberboat/pkg/apiserver/recover"
	"p9t.io/kuberboat/pkg/apiserver/scale"
	"p9t.io/kuberboat/pkg/apiserver/schedule"
	"p9t.io/kuberboat/pkg/apiserver/service"
	pb "p9t.io/kuberboat/pkg/proto"
)

// FIXME: Move the managers and controllers into a wrapper.
var nodeManager node.NodeManager
var componentManager apiserver.ComponentManager
var legacyManager apiserver.LegacyManager
var metricsManager scale.MetricsManager
var podScheduler schedule.PodScheduler
var podController pod.Controller
var jobController job.Controller
var serviceController service.Controller
var deploymentController deployment.Contoller
var nodeController node.Controller
var dnsController dns.Controller
var autoscalerController scale.Controller

type server struct {
	pb.UnimplementedApiServerKubeletServiceServer
	pb.UnimplementedApiServerCtlServiceServer
}

func (s *server) DescribePods(ctx context.Context, req *pb.DescribePodsRequest) (*pb.DescribePodsResponse, error) {
	foundPods, notFoundPods := podController.GetPods(req.All, req.PodNames)

	foundPodsData, err := json.Marshal(foundPods)
	if err != nil {
		return &pb.DescribePodsResponse{
			Status:       -1,
			Pods:         nil,
			NotFoundPods: nil,
		}, err
	}

	notFoundPodsData, err := json.Marshal(notFoundPods)
	if err != nil {
		return &pb.DescribePodsResponse{
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

	return &pb.DescribePodsResponse{
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

func (s *server) CreateJob(ctx context.Context, req *pb.CreateJobRequest) (*pb.DefaultResponse, error) {
	var job core.Job
	if err := json.Unmarshal(req.Job, &job); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	if err := jobController.ApplyJob(&job); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	return &pb.DefaultResponse{Status: 0}, nil
}

func (s *server) GetJobLog(ctx context.Context, req *pb.LogJobRequest) (*pb.LogJobResponse, error) {
	resp, err := jobController.GetJobLog(req.JobName)
	return &pb.LogJobResponse{Log: resp}, err
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
		glog.Infof("EVENT: pod %v is ready", req.PodName)
		apiserver.Dispatch(&apiserver.PodReadyEvent{PodName: req.PodName})
	}
	// Try to dispatch PodFailEvent.
	if prevStatus.Phase != core.PodFailed && status.Phase == core.PodFailed {
		glog.Infof("EVENT: pod %v failed", req.PodName)
		apiserver.Dispatch(&apiserver.PodFailEvent{PodName: req.PodName})
	}
	// Try to dispatch PodSucceedEvent
	if prevStatus.Phase != core.PodSucceeded && status.Phase == core.PodSucceeded {
		glog.Infof("EVENT: pod %v succeeded", req.PodName)
		apiserver.Dispatch(&apiserver.PodSucceedEvent{PodName: req.PodName})
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

func (*server) DescribeDNSs(ctx context.Context, req *pb.DescribeDNSsRequest) (*pb.DescribeDNSsResponse, error) {
	foundDNSs, notFoundDNSs := dnsController.GetDNSs(req.All, req.DnsNames)
	serializeErrorResponse := &pb.DescribeDNSsResponse{
		Status:       -1,
		Dnss:         nil,
		NotFoundDnss: nil,
	}

	foundDNSsData, err := json.Marshal(foundDNSs)
	if err != nil {
		return serializeErrorResponse, err
	}

	notFoundDNSsData, err := json.Marshal(notFoundDNSs)
	if err != nil {
		return serializeErrorResponse, err
	}

	var status int32
	if len(notFoundDNSs) > 0 {
		status = -2
	} else {
		status = 0
	}

	return &pb.DescribeDNSsResponse{
		Status:       status,
		Dnss:         foundDNSsData,
		NotFoundDnss: notFoundDNSsData,
	}, nil
}

func (*server) CreateAutoscaler(ctx context.Context, req *pb.CreateAutoscalerRequest) (*pb.DefaultResponse, error) {
	var autoscaler core.HorizontalPodAutoscaler

	if err := json.Unmarshal(req.Autoscaler, &autoscaler); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	if err := autoscalerController.CreateAutoscaler(&autoscaler); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	return &pb.DefaultResponse{Status: 0}, nil
}

func (*server) DescribeNodes(ctx context.Context, req *pb.EmptyRequest) (*pb.DescribeNodesResponse, error) {
	nodes := nodeController.GetRegisteredNodes()
	data, err := json.Marshal(nodes)
	if err != nil {
		return &pb.DescribeNodesResponse{Status: -1}, err
	}
	return &pb.DescribeNodesResponse{Status: 0, Nodes: data}, nil
}

func (*server) DescribeAutoscalers(ctx context.Context, req *pb.DescribeAutoscalersRequest) (
	*pb.DescribeAutoscalersResponse,
	error,
) {
	foundAutoscalers, notFoundAutoscalers := autoscalerController.DescribeAutoscalers(req.All, req.AutoscalerNames)
	foundPodsData, err := json.Marshal(foundAutoscalers)
	if err != nil {
		return &pb.DescribeAutoscalersResponse{
			Status:              -1,
			Autoscalers:         nil,
			NotFoundAutoscalers: nil,
		}, err
	}

	notFoundPodsData, err := json.Marshal(notFoundAutoscalers)
	if err != nil {
		return &pb.DescribeAutoscalersResponse{
			Status:              -1,
			Autoscalers:         nil,
			NotFoundAutoscalers: nil,
		}, err
	}

	var status int32
	if len(notFoundAutoscalers) > 0 {
		status = -2
	} else {
		status = 0
	}

	return &pb.DescribeAutoscalersResponse{
		Status:              status,
		Autoscalers:         foundPodsData,
		NotFoundAutoscalers: notFoundPodsData,
	}, nil
}

func StartServer(etcdServers string) {
	if err := etcd.InitializeClient(etcdServers); err != nil {
		glog.Fatal(err)
	}
	nodeManager = node.NewNodeManager()
	componentManager = apiserver.NewComponentManager()
	legacyManager = apiserver.NewLegacyManager(componentManager)
	metricsManager = scale.NewMetricsManager(componentManager)
	podScheduler = schedule.NewPodScheduler(nodeManager, componentManager)
	podController = pod.NewPodController(componentManager, podScheduler, nodeManager, legacyManager)
	jobController = job.NewJobController(podController, nodeManager, componentManager)
	serviceController = service.NewServiceController(componentManager, nodeManager)
	deploymentController = deployment.NewDeploymentController(componentManager, podController)
	nodeController = node.NewNodeController(nodeManager)
	dnsController = dns.NewDNSController(componentManager)
	autoscalerController = scale.NewAutoscalerController(componentManager, metricsManager)

	if err := recover.Recover(&nodeManager, &componentManager, serviceController); err != nil {
		glog.Fatal(err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", core.APISERVER_PORT))
	if err != nil {
		glog.Fatal("Api server failed to connect!")
	}
	if err != nil {
		glog.Fatal(err)
	}

	apiServer := grpc.NewServer()
	pb.RegisterApiServerCtlServiceServer(apiServer, &server{})
	pb.RegisterApiServerKubeletServiceServer(apiServer, &server{})

	glog.Infof("API SERVER: listening at %v", lis.Addr())

	if err := apiServer.Serve(lis); err != nil {
		glog.Fatal(err)
	}
}
