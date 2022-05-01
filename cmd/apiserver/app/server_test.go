package app

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"
	"p9t.io/kuberboat/pkg/api"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver/node"
	pb "p9t.io/kuberboat/pkg/proto"
)

var testPod = core.Pod{
	Kind: core.PodType,
	ObjectMeta: core.ObjectMeta{
		Name:              "test-pod",
		UUID:              uuid.New(),
		CreationTimestamp: time.Now(),
		Labels:            map[string]string{},
	},
	Spec: core.PodSpec{
		Containers: []core.Container{
			{
				Name:  "nginx",
				Image: "nginx:latest",
				Ports: []uint16{80},
				VolumeMounts: []core.VolumeMount{
					{
						Name:      "test-volume",
						MountPath: "/test",
					},
				},
			},
		},
		Volumes: []string{
			"test-volume",
		},
	},
	Status: core.PodStatus{
		Phase: core.PodPending,
	},
}

var testNode = core.Node{
	Kind: core.NodeType,
	ObjectMeta: core.ObjectMeta{
		Name:              "test-node",
		UUID:              uuid.New(),
		CreationTimestamp: time.Now(),
		Labels:            map[string]string{},
	},
	Spec:   core.NodeSpec{},
	Status: core.NodeStatus{},
}

// TestCreatePod must be run after kubelet is up.
func TestCreatePod(t *testing.T) {
	conn, err := grpc.Dial("localhost:4000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewKubeletApiServerServiceClient(conn)

	// Contact the server and print out its response.
	ctx := context.Background()
	bytes, err := json.Marshal(testPod)
	if err != nil {
		log.Fatal(err)
	}
	_, err = c.CreatePod(ctx, &pb.KubeletCreatePodRequest{Pod: bytes})
	if err != nil {
		log.Fatalf("could not create pod: %v", err)
	}
}

// TestNotifyRegistered must be run after kubelet is up.
func TestNotifyRegistered(t *testing.T) {
	oldServerIP := os.Getenv(api.ApiServerIP)
	assert.NoError(t, os.Setenv(api.ApiServerIP, "localhost"))
	nodeManager = node.NewNodeManager()
	nodeController = node.NewNodeController(nodeManager)
	// ctx simulates ctl rpc to api server.
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.IPAddr{IP: net.ParseIP("127.0.0.1")},
	})
	assert.NoError(t, nodeController.RegisterNode(ctx, &testNode))
	os.Setenv(api.ApiServerIP, oldServerIP)
}
