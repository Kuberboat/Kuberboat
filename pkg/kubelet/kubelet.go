package kubelet

import (
	"context"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"

	"github.com/golang/glog"

	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockerfilters "github.com/docker/docker/api/types/filters"
	dockervolume "github.com/docker/docker/api/types/volume"
	dockerclient "github.com/docker/docker/client"
	dockernat "github.com/docker/go-connections/nat"
	"p9t.io/kuberboat/pkg/api/core"
	kubepod "p9t.io/kuberboat/pkg/kubelet/pod"
)

const (
	pauseImage         string = "docker.io/mirrorgooglecontainers/pause-amd64:3.0"
	pauseContainerName string = "pause"
)

// Kubelet defines public methods of a PodManager.
// All methods are thread safe.
type Kubelet interface {
	// GetPods returns the pods bound to the kubelet and their spec.
	GetPods() []*core.Pod
	// GetPodByName provides the pod that matches name, as well as whether the pod was found.
	GetPodByName(name string) (*core.Pod, bool)
	// AddPod runs a pod based on the pod spec passed in as parameter.
	// The status and metadata of the pod will be managed.
	AddPod(ctx context.Context, pod *core.Pod) error
	// DeletePodByName destroys a pod indexed by name and all its containers.
	DeletePodByName(ctx context.Context, name string) error
}

// Kubelet is the core data structure of the component. It manages pods, containers, monitors.
type dockerKubelet struct {
	// Ensure concurrent access to inner data structures are safe.
	mtx sync.Mutex
	// Docker client to access docker apis.
	dockerClient *dockerclient.Client
	// Manage pod metadata.
	podMetaManager kubepod.MetaManager
	// Manage pod runtime data.
	podRuntimeManager kubepod.RuntimeManager
}

var kubelet Kubelet

// Instance is the access point of the singleton kubelet instance.
// NOT thread safe.
func Instance() Kubelet {
	if kubelet == nil {
		kubelet = newKubelet()
	}
	return kubelet
}

// newKubelet creates a new Kubelet object.
func newKubelet() Kubelet {
	// Create docker client.
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		glog.Fatal(err)
	}
	return &dockerKubelet{
		dockerClient:      cli,
		podMetaManager:    kubepod.NewMetaManager(),
		podRuntimeManager: kubepod.NewRuntimeManager(),
	}
}

func (kl *dockerKubelet) GetPods() []*core.Pod {
	return kl.podMetaManager.Pods()
}

func (kl *dockerKubelet) GetPodByName(name string) (*core.Pod, bool) {
	return kl.podMetaManager.PodByName(name)
}

func (kl *dockerKubelet) AddPod(ctx context.Context, pod *core.Pod) error {
	if _, ok := kl.podMetaManager.PodByName(pod.Name); ok {
		return fmt.Errorf("pod exists already: %v", pod.Name)
	}

	// Update pod status as Running.
	// Note that running does not mean ready.
	kl.podMetaManager.AddPod(pod)
	pod.Status.Phase = core.PodRunning
	// TODO: Defer broadcasting condition variable. CV and mtx should be a member of the kubelet.

	// Start sandbox pause container.
	if err := kl.runPodSandBox(ctx, pod); err != nil {
		return err
	}

	// Create volumes for the pod.
	if err := kl.createPodVolumes(ctx, pod); err != nil {
		return err
	}

	// Start user containers.
	for _, c := range pod.Spec.Containers {
		if err := kl.runPodContainer(ctx, pod, &c); err != nil {
			return err
		}
	}

	// TODO(yuanxin.cao): Start a monitor to monitor pod status.

	return nil
}

// runPodSandBox pulls pause image and runs pause container.
// The name of the pause container will be "<pod UUID>_pause"
// User pods will share network and PID space with this container.
func (kl *dockerKubelet) runPodSandBox(ctx context.Context, pod *core.Pod) error {
	cli := kl.dockerClient

	// Pull pause image.
	out, err := cli.ImagePull(ctx, pauseImage, dockertypes.ImagePullOptions{})
	if err != nil {
		return err
	}
	pullRes, err := io.ReadAll(out)
	if err != nil {
		return nil
	}
	glog.Info(string(pullRes[:]))
	defer func(out io.ReadCloser) {
		err := out.Close()
		if err != nil {
			glog.Error(err)
		}
	}(out)

	// Populate exposed ports.
	ports := make(map[dockernat.Port]struct{})
	for _, c := range pod.Spec.Containers {
		for _, p := range c.Ports {
			ports[dockernat.Port(fmt.Sprintf("%v/tcp", p.ContainerPort))] = struct{}{}
		}
	}

	pauseName := GetPodSpecificName(pod, pauseContainerName)
	resp, err := cli.ContainerCreate(ctx, &dockercontainer.Config{
		Image:        pauseImage,
		ExposedPorts: ports,
	}, &dockercontainer.HostConfig{
		IpcMode: "shareable",
	}, nil, nil, pauseName)
	if err != nil {
		return err
	}

	// Start pause container.
	if err := cli.ContainerStart(ctx, resp.ID, dockertypes.ContainerStartOptions{}); err != nil {
		return err
	}

	// Inspect the container to get pod IP, and update pod status.
	containerJson, err := cli.ContainerInspect(ctx, pauseName)
	if err != nil {
		return fmt.Errorf("cannot get pod IP: %v", err.Error())
	}
	podIP := containerJson.NetworkSettings.DefaultNetworkSettings.IPAddress
	if net.ParseIP(podIP) == nil {
		return fmt.Errorf("invalid pod IP: %v", podIP)
	}
	pod.Status.PodIP = podIP

	return nil
}

// createVolumes creates docker volumes for the pod.
func (kl *dockerKubelet) createPodVolumes(ctx context.Context, pod *core.Pod) error {
	for _, vName := range pod.Spec.Volumes {
		// Check if target volume already exists.
		resp, err := kl.dockerClient.VolumeList(
			ctx,
			dockerfilters.NewArgs(
				dockerfilters.KeyValuePair{
					Key: "name",
					// Use regex for exact name matching.
					Value: fmt.Sprintf("^%v$", vName),
				}))
		if err != nil {
			return err
		}

		// If so, remove it.
		if len(resp.Volumes) > 0 {
			if len(resp.Volumes) > 1 {
				return fmt.Errorf("more than 1 volumes have the name: %v", vName)
			}
			err = kl.dockerClient.VolumeRemove(ctx, vName, true)
			if err != nil {
				return err
			}
		}

		// Create the new volume.
		dockerVolume, err := kl.dockerClient.VolumeCreate(ctx, dockervolume.VolumeCreateBody{
			Driver: "local",
			Name:   vName,
		})
		if err != nil {
			return err
		}
		kl.podRuntimeManager.AddPodVolume(pod, dockerVolume.Name)
	}
	return nil
}

// runPodContainer runs a container and joins it to pod's pause container.
func (kl *dockerKubelet) runPodContainer(ctx context.Context, pod *core.Pod, c *core.Container) error {
	pauseContainerName := GetPodSpecificName(pod, pauseContainerName)
	cli := kl.dockerClient

	// Pull image.
	out, err := cli.ImagePull(ctx, c.Image, dockertypes.ImagePullOptions{})
	if err != nil {
		return err
	}
	pullRes, err := io.ReadAll(out)
	if err != nil {
		return nil
	}
	glog.Info(string(pullRes[:]))
	defer func(out io.ReadCloser) {
		err := out.Close()
		if err != nil {
			glog.Error(err)
		}
	}(out)

	// Populate volume bindings.
	vBinds := make([]string, 0, len(c.VolumeMounts))
	for _, m := range c.VolumeMounts {
		vBinds = append(vBinds, fmt.Sprintf("%v:%v", GetPodSpecificName(pod, m.Name), m.MountPath))
	}

	// Populate resources.
	resources := dockercontainer.Resources{}
	if bytes, ok := c.Resources["memory"]; ok {
		if int64(bytes) < 0 {
			return fmt.Errorf("memory limit overflow: %v", bytes)
		} else {
			resources.Memory = int64(bytes)
		}
	}
	if cpu, ok := c.Resources["cpu"]; ok {
		if int64(cpu) < 0 {
			return fmt.Errorf("cpu limit overflow: %v", cpu)
		}
		if int(cpu) > runtime.NumCPU() {
			cpu = uint64(runtime.NumCPU())
		}
		resources.NanoCPUs = 1000000000 * int64(cpu)
	}

	// Create container.
	mode := fmt.Sprintf("container:%v", pauseContainerName)
	resp, err := cli.ContainerCreate(ctx, &dockercontainer.Config{
		Image: c.Image,
		Cmd:   c.Commands,
	}, &dockercontainer.HostConfig{
		Binds:       vBinds,
		NetworkMode: dockercontainer.NetworkMode(mode),
		IpcMode:     dockercontainer.IpcMode(mode),
		PidMode:     dockercontainer.PidMode(mode),
		Resources:   resources,
	}, nil, nil, GetPodSpecificName(pod, c.Name))
	if err != nil {
		return err
	}
	kl.podRuntimeManager.AddPodContainer(pod, resp.ID)

	// Start container.
	if err := cli.ContainerStart(ctx, resp.ID, dockertypes.ContainerStartOptions{}); err != nil {
		return err
	}

	return nil
}

func (kl *dockerKubelet) DeletePodByName(ctx context.Context, name string) error {
	pod, ok := kl.podMetaManager.PodByName(name)
	if !ok {
		return fmt.Errorf("pod does not exist: %v", name)
	}

	// TODO: Wait until pod is done adding. By doing while () { cv.Wait() }
	kl.podMetaManager.DeletePodByName(name)

	// Remove user containers.
	containers, _ := kl.podRuntimeManager.ContainersByPod(pod)
	for _, c := range containers {
		err := kl.dockerClient.ContainerStop(ctx, c, nil)
		if err != nil {
			return err
		}
		err = kl.dockerClient.ContainerRemove(ctx, c, dockertypes.ContainerRemoveOptions{})
		if err != nil {
			return err
		}
	}

	// Remove pause container.
	pauseName := GetPodSpecificName(pod, pauseContainerName)
	err := kl.dockerClient.ContainerStop(ctx, pauseName, nil)
	if err != nil {
		return err
	}
	err = kl.dockerClient.ContainerRemove(ctx, pauseName, dockertypes.ContainerRemoveOptions{})
	if err != nil {
		return err
	}

	// Remove volumes.
	volumes, _ := kl.podRuntimeManager.VolumesByPod(pod)
	for _, v := range volumes {
		err = kl.dockerClient.VolumeRemove(ctx, v, true)
		if err != nil {
			return err
		}
	}

	return nil
}
