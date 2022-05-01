package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"p9t.io/kuberboat/pkg/api/core"
	pb "p9t.io/kuberboat/pkg/proto"
)

const CONN_TIMEOUT time.Duration = time.Second

type KubeletClient struct {
	connection *grpc.ClientConn
	client     pb.ApiServerKubeletServiceClient
}

func NewKubeletClient(apiserverIP string, apiserverPort uint16) (*KubeletClient, error) {
	addr := fmt.Sprintf("%v:%v", apiserverIP, apiserverPort)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to api server: %v", err.Error())
	}
	return &KubeletClient{
		connection: conn,
		client:     pb.NewApiServerKubeletServiceClient(conn),
	}, nil
}

func (c *KubeletClient) UpdatePodStatus(pod *core.Pod) (*pb.DefaultResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	status, err := json.Marshal(pod.Status)
	if err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	return c.client.UpdatePodStatus(ctx, &pb.UpdatePodStatusRequest{
		PodName:   pod.Name,
		PodStatus: status,
	})
}

func (c *KubeletClient) NotifyPodDeletion(success bool, pod *core.Pod) (*pb.DefaultResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CONN_TIMEOUT)
	defer cancel()
	if !success {
		c.client.NotifyPodDeletion(ctx, &pb.NotifyPodDeletionRequest{
			Success:    false,
			DeletedPod: nil,
		})
	}
	podData, err := json.Marshal(pod)
	if err != nil {
		return &pb.DefaultResponse{Status: -1}, err
	}
	return c.client.NotifyPodDeletion(ctx, &pb.NotifyPodDeletionRequest{
		Success:    true,
		DeletedPod: podData,
	})
}
