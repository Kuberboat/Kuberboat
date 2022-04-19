package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/golang/glog"

	"google.golang.org/grpc"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/kubelet"
	pb "p9t.io/kuberboat/pkg/proto"
)

type server struct {
	pb.UnimplementedKubeletApiServerServiceServer
}

func (s *server) CreatePod(ctx context.Context, req *pb.KubeletCreatePodRequest) (*pb.KubeletCreatePodResponse, error) {
	var pod core.Pod
	err := json.Unmarshal(req.Pod, &pod)
	if err != nil {
		return &pb.KubeletCreatePodResponse{Status: -1}, err
	}

	go kubelet.Instance().AddPod(context.Background(), &pod)

	return &pb.KubeletCreatePodResponse{Status: 0}, nil
}

func (s *server) DeletePod(ctx context.Context, req *pb.KubeletDeletePodRequest) (*pb.KubeletDeletePodResponse, error) {
	go kubelet.Instance().DeletePodByName(ctx, req.PodName)
	return &pb.KubeletDeletePodResponse{Status: 0}, nil
}

func StartServer(config *KubeletConfig) {
	grpcServer := grpc.NewServer()
	pb.RegisterKubeletApiServerServiceServer(grpcServer, &server{})

	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", config.Port))
	if err != nil {
		glog.Fatal(err)
	}

	if err := grpcServer.Serve(lis); err != nil {
		glog.Fatal(err)
	}
}
