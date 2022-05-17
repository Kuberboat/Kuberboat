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

func (c *ApiserverClient) TransferFile(data []byte) (*pb.DefaultResponse, error) {
	ctx := context.Background()
	return c.kubeletClient.TransferFile(ctx, &pb.KubeletTransferFileRequest{File: data})
}

func (c *ApiserverClient) GetPodLog(name string) (*pb.KubeletGetPodLogResponse, error) {
	ctx := context.Background()
	return c.kubeletClient.GetPodLog(ctx, &pb.KubeletGetPodLogRequest{PodName: name})
}

func (c *ApiserverClient) CreateService(service *core.Service, pods *list.List) (*pb.DefaultResponse, error) {
	ctx := context.Background()
	servicePorts := make([][]byte, 0, len(service.Spec.Ports))
	for _, port := range service.Spec.Ports {
		portBytes, err := json.Marshal(port)
		if err != nil {
			return &pb.DefaultResponse{Status: -1}, err
		}
		servicePorts = append(servicePorts, portBytes)
	}
	podNames := make([]string, 0, pods.Len())
	podIPs := make([]string, 0, pods.Len())
	for it := pods.Front(); it != nil; it = it.Next() {
		pod := it.Value.(*core.Pod)
		podNames = append(podNames, pod.Name)
		podIPs = append(podIPs, pod.Status.PodIP)
	}
	request := pb.KubeletCreateServiceRequest{
		ServiceName:  service.Name,
		ClusterIp:    service.Spec.ClusterIP,
		ServicePorts: servicePorts,
		PodNames:     podNames,
		PodIps:       podIPs,
	}
	return c.kubeletClient.CreateService(ctx, &request)
}

func (c *ApiserverClient) DeleteService(serviceName string) (*pb.DefaultResponse, error) {
	ctx := context.Background()
	return c.kubeletClient.DeleteService(ctx, &pb.KubeletDeleteServiceRequest{
		ServiceName: serviceName,
	})
}

func (c *ApiserverClient) AddPodToServices(serviceNames []string, podName string, podIP string) (*pb.DefaultResponse, error) {
	ctx := context.Background()
	return c.kubeletClient.AddPodToServices(ctx, &pb.KubeletUpdateServiceRequest{
		ServiceNames: serviceNames,
		PodName:      podName,
		PodIp:        podIP,
	})
}

func (c *ApiserverClient) DeletePodFromServices(serviceNames []string, podName string) (*pb.DefaultResponse, error) {
	ctx := context.Background()
	return c.kubeletClient.DeletePodFromServices(ctx, &pb.KubeletUpdateServiceRequest{
		ServiceNames: serviceNames,
		PodName:      podName,
	})
}
