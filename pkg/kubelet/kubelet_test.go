package kubelet

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/golang/glog"

	"github.com/google/uuid"
	"p9t.io/kuberboat/pkg/api/core"
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
			// Ensure that shared volume works correctly.
			{
				Name:  "redis",
				Image: "redis:latest",
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

func TestAddAndDeletePod(t *testing.T) {
	err := flag.Set("logtostderr", "true")
	if err != nil {
		return
	}
	flag.Parse()

	ctx := context.Background()
	kl := Instance()
	if err := kl.AddPod(ctx, &testPod); err != nil {
		glog.Fatal(err)
	}
	// Validate pod
	if err := kl.DeletePodByName(ctx, testPod.Name); err != nil {
		glog.Fatal(err)
	}
}
