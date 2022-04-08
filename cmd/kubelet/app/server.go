package app

import (
	"fmt"
	"github.com/golang/glog"
	"net"

	"google.golang.org/grpc"
)

func StartServer(config *KubeletConfig) {
	grpcServer := grpc.NewServer()
	// TODO: register rpc services

	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", config.Port))
	if err != nil {
		glog.Fatal(err)
	}

	if err := grpcServer.Serve(lis); err != nil {
		glog.Fatal(err)
	}
}
