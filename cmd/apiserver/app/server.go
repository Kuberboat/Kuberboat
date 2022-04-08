package app

import (
	"context"
	"fmt"
	"net"

	"github.com/golang/glog"

	"google.golang.org/grpc"
	pb "p9t.io/kuberboat/pkg/proto"
)

// FIXME: Move this into config file.
const APISERVER_PORT uint16 = 6789

type server struct {
	pb.UnimplementedApiServerServiceServer
}

func (s *server) CreatePod(ctx context.Context, req *pb.CreatePodRequest) (*pb.CreatePodResponse, error) {
	// TODO: Create pod logic
	return &pb.CreatePodResponse{Status: 0}, nil
}

func (s *server) DeletePod(ctx context.Context, req *pb.DeletePodRequest) (*pb.DeletePodResponse, error) {
	// TODO: Delete pod logic
	return &pb.DeletePodResponse{Status: 0}, nil
}

func Run() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", APISERVER_PORT))
	if err != nil {
		glog.Fatal("Api server failed to connect!")
	}

	apiServer := grpc.NewServer()
	pb.RegisterApiServerServiceServer(apiServer, &server{})
	glog.Infof("Api server listening at %v\n", lis.Addr())

	if err := apiServer.Serve(lis); err != nil {
		glog.Fatal(err)
	}
}
