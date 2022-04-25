package client

import (
	"container/list"
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

func (c *ApiserverClient) NotifyRegistered(apiserver *core.ApiserverStatus) (*pb.DefaultResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	data, err := json.Marshal(apiserver)
	if err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	return c.kubeletClient.NotifyRegistered(ctx, &pb.NotifyRegisteredRequest{
		Apiserver: data,
	})
}

func (c *ApiserverClient) CreatePod(pod *core.Pod) (*pb.DefaultResponse, error) {
	ctx := context.Background()
	data, err := json.Marshal(pod)
	if err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	return c.kubeletClient.CreatePod(ctx, &pb.KubeletCreatePodRequest{Pod: data})
}

func (c *ApiserverClient) DeletePodByName(name string) (*pb.DefaultResponse, error) {
	ctx := context.Background()
	return c.kubeletClient.DeletePod(ctx, &pb.KubeletDeletePodRequest{PodName: name})
}

func (c *ApiserverClient) CreateService(service *core.Service, pods *list.List) (*pb.DefaultResponse, error) {
	ctx := context.Background()
	podIPs := make([]string, 0, pods.Len())
	for it := pods.Front(); it != nil; it = it.Next() {
		podIPs = append(podIPs, it.Value.(*core.Pod).Status.PodIP)
	}
	request := pb.KubeletCreateServiceRequest{
		ServiceName: service.Name,
		ServiceId:   service.UUID.String()[:8],
		ClusterIp:   service.Spec.ClusterIP,
		PodIp:       podIPs,
	}
	return c.kubeletClient.CreateService(ctx, &request)
}

func (c *ApiserverClient) DeleteService(serviceName string) (*pb.DefaultResponse, error) {
	ctx := context.Background()
	return c.kubeletClient.DeleteService(ctx, &pb.KubeletDeleteServiceRequest{
		ServiceName: serviceName,
	})
}
