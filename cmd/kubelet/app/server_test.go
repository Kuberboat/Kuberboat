package app

import (
	"context"
	"encoding/json"
	"log"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"p9t.io/kuberboat/pkg/api/core"
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
