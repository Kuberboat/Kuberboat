package etcd

import (
	"container/list"
	"fmt"
	"testing"
	"time"

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

func TestCrud(t *testing.T) {
	assert := assert.New(t)
	err := InitializeClient("localhost:2379")
	assert.Nil(err)
	err = Put(fmt.Sprintf("/Pods/%s", testPod.Name), testPod)
	assert.Nil(err)
	var pod core.Pod
	rawPods, err := Get(fmt.Sprintf("/Pods/%s", testPod.Name), pod)
	assert.Nil(err)
	retrivePod := rawPods[0].(core.Pod)
	assert.True(testPod.CreationTimestamp.Equal(retrivePod.CreationTimestamp))
	retrivePod.CreationTimestamp = testPod.CreationTimestamp
	assert.Equal(testPod, retrivePod)
	err = Delete(fmt.Sprintf("/Pods/%s", testPod.Name))
	assert.Nil(err)
	rawPods, err = Get(fmt.Sprintf("/Pods/%s", testPod.Name), pod)
	assert.Nil(err)
	assert.Len(rawPods, 0)
}

func TestGetNames(t *testing.T) {
	assert := assert.New(t)
	err := InitializeClient("localhost:2379")
	assert.Nil(err)
	pods := list.New()
	for i := 0; i < 3; i++ {
		newPod := testPod
		newPod.ObjectMeta.Name = fmt.Sprintf("test-pod-%v", i)
		pods.PushBack(&newPod)
	}
	k := "/Services/Pods/test-service"
	err = Put(k, core.GetPodNames(pods))
	assert.Nil(err)
	var podNames []string
	rawPodNames, err := Get(k, podNames)
	assert.Nil(err)
	podNames = rawPodNames[0].([]string)
	assert.Len(podNames, 3)
	for i, podName := range podNames {
		assert.Equal(fmt.Sprintf("test-pod-%v", i), podName)
	}
}
