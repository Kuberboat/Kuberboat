package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/golang/glog"

	"google.golang.org/grpc"
	"p9t.io/kuberboat/pkg/api/core"
	kubeerror "p9t.io/kuberboat/pkg/api/error"
	"p9t.io/kuberboat/pkg/kubelet"
	pb "p9t.io/kuberboat/pkg/proto"
)

type server struct {
	pb.UnimplementedKubeletApiServerServiceServer
}

func (s *server) NotifyRegistered(ctx context.Context, req *pb.NotifyRegisteredRequest) (*pb.DefaultResponse, error) {
	var apiserver core.ApiserverStatus
	if err := json.Unmarshal(req.Apiserver, &apiserver); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	if err := kubelet.KubeletInstance().ConnectToServer(&apiserver); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	return &pb.DefaultResponse{Status: 0}, nil
}

func (s *server) CreatePod(ctx context.Context, req *pb.KubeletCreatePodRequest) (*pb.DefaultResponse, error) {
	var pod core.Pod
	if err := json.Unmarshal(req.Pod, &pod); err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}

	go kubelet.KubeletInstance().AddPod(context.Background(), &pod)

	return &pb.DefaultResponse{Status: 0}, nil
}

func (s *server) DeletePod(ctx context.Context, req *pb.KubeletDeletePodRequest) (*pb.DefaultResponse, error) {
	go kubelet.KubeletInstance().DeletePodByName(context.Background(), req.PodName)
	return &pb.DefaultResponse{Status: 0}, nil
}

func (s *server) CreateService(ctx context.Context, req *pb.KubeletCreateServiceRequest) (*pb.DefaultResponse, error) {
	if len(req.PodNames) != len(req.PodIps) {
		return &pb.DefaultResponse{Status: -2}, kubeerror.KubeError{
			Type:    kubeerror.KubeErrGrpc,
			Message: "different numbers of pod id and pod ip",
		}
	}
	servicePorts := make([]*core.ServicePort, 0, len(req.ServicePorts))
	for _, bytes := range req.ServicePorts {
		var port core.ServicePort
		err := json.Unmarshal(bytes, &port)
		if err != nil {
			return &pb.DefaultResponse{Status: -1}, err
		}
		servicePorts = append(servicePorts, &port)
	}
	err := kubelet.ProxyInstance().CreateService(req.ServiceName, req.ClusterIp, servicePorts, req.PodNames, req.PodIps)
	if err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	return &pb.DefaultResponse{Status: 0}, nil
}

func (s *server) DeleteService(ctx context.Context, req *pb.KubeletDeleteServiceRequest) (*pb.DefaultResponse, error) {
	err := kubelet.ProxyInstance().DeleteService(req.ServiceName)
	if err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	return &pb.DefaultResponse{Status: 0}, nil
}

func StartServer() {
	grpcServer := grpc.NewServer()
	pb.RegisterKubeletApiServerServiceServer(grpcServer, &server{})

	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", kubelet.Port))
	if err != nil {
		glog.Fatal(err)
	}

	glog.Infof("kubelet server listening at port %v", kubelet.Port)
	if err := grpcServer.Serve(lis); err != nil {
		glog.Fatal(err)
	}
}
