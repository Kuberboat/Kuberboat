package kubelet

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/golang/glog"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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

var invalidPod = core.Pod{
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
			},
			{
				Name:  "ngin",
				Image: "ngin:latest",
				Ports: []uint16{80},
			},
		},
	},
	Status: core.PodStatus{
		Phase: core.PodPending,
	},
}

func validateCleanUp(t *testing.T, kl Kubelet) {
	dockerkl := kl.(*dockerKubelet)
	assert.Empty(t, dockerkl.podMetaManager.Pods())
	containers, ok := dockerkl.podRuntimeManager.ContainersByPod(&testPod)
	assert.Empty(t, containers)
	assert.False(t, ok)
	volumes, ok := dockerkl.podRuntimeManager.VolumesByPod(&testPod)
	assert.Empty(t, volumes)
	assert.False(t, ok)
}

// AddPod might fail due to network issues, but we need to make sure metadata is properly cleaned up.
func TestAddAndDeletePod(t *testing.T) {
	err := flag.Set("logtostderr", "true")
	if err != nil {
		return
	}
	flag.Parse()

	ctx := context.Background()
	kl := KubeletInstance()
	if err := kl.AddPod(ctx, &testPod); err != nil {
		validateCleanUp(t, kl)
		glog.Fatal(err)
	}
	// Validate pod
	if err := kl.DeletePodByName(ctx, testPod.Name); err != nil {
		glog.Fatal(err)
	}
	validateCleanUp(t, kl)
}

func TestAddInvalidPod(t *testing.T) {
	err := flag.Set("logtostderr", "true")
	if err != nil {
		return
	}
	flag.Parse()

	ctx := context.Background()
	kl := KubeletInstance()
	err = kl.AddPod(ctx, &invalidPod)
	assert.NotNil(t, err)
	assert.NotEmpty(t, kl.GetPods())
}
