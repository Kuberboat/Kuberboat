package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"p9t.io/kuberboat/pkg/api/core"
	pb "p9t.io/kuberboat/pkg/proto"
)

// FIXME: put this into config file
var CONN_TIMEOUT time.Duration = time.Second

type ApiserverClient struct {
	connection    *grpc.ClientConn
	kubeletClient pb.KubeletApiServerServiceClient
}

func NewCtlClient(url string, kubeletPort uint16) (*ApiserverClient, error) {
	addr := fmt.Sprintf("%v:%v", url, kubeletPort)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.New("apiserver client failed to connect to worker node")
	}
	return &ApiserverClient{
		connection:    conn,
		kubeletClient: pb.NewKubeletApiServerServiceClient(conn),
	}, nil
}

func (c *ApiserverClient) NotifyRegistered(apiserver *core.Cluster) (*pb.NotifyRegisteredResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	data, err := json.Marshal(apiserver)
	if err != nil {
		return &pb.NotifyRegisteredResponse{Status: -1}, err
	}
	return c.kubeletClient.NotifyRegistered(ctx, &pb.NotifyRegisteredRequest{
		Apiserver: data,
	})
}

func (c *ApiserverClient) CreatePod(pod *core.Pod) (*pb.KubeletCreatePodResponse, error) {
	ctx := context.Background()
	data, err := json.Marshal(pod)
	if err != nil {
		return &pb.KubeletCreatePodResponse{Status: -1}, err
	}
	return c.kubeletClient.CreatePod(ctx, &pb.KubeletCreatePodRequest{Pod: data})
}
