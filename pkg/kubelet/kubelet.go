package kubelet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"

	"github.com/golang/glog"

	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	dockernat "github.com/docker/go-connections/nat"
	etcd "go.etcd.io/etcd/client/v3"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/kubelet/client"
	kubeletpod "p9t.io/kuberboat/pkg/kubelet/pod"
)

const (
	pauseImage      string = "docker.io/mirrorgooglecontainers/pause-amd64:3.0"
	cadvisorImage   string = "google/cadvisor:v0.33.0"
	cadvisorPort    uint16 = 8080
	cadvisorName    string = "kuberboat-cadvisor"
	Port                   = 4000
	dnsIPKey               = "/ip/coredns"
	etcdPort               = 2379
	etcdDialTimeout        = 2000000000
)

// Kubelet defines public methods of a PodManager.
// All methods are thread safe.
type Kubelet interface {
	// ConnectToServer initializes grpc client to the api server.
	ConnectToServer(cluster *core.ApiserverStatus) error
	// GetPods returns the pods bound to the kubelet and their spec.
	GetPods() []*core.Pod
	// GetPodByName provides the pod that matches name, as well as whether the pod was found.
	GetPodByName(name string) (*core.Pod, bool)
	// AddPod runs a pod based on the pod spec passed in as parameter.
	// The status and metadata of the pod will be managed.
	AddPod(ctx context.Context, pod *core.Pod) error
	// DeletePodByName destroys a pod indexed by name and all its containers.
	DeletePodByName(ctx context.Context, name string) error
	// StartCAdvisor starts cadvisor container in Kubelet, used for monitoring the pods.
	StartCAdvisor() error
}

// Kubelet is the core data structure of the component. It manages pods, containers, monitors.
type dockerKubelet struct {
	// IP address of the DNS name server for all the containers.
	dnsIP string
	// Client to communicate with API server.
	apiClient *client.KubeletClient
	// Ensure concurrent access to inner data structures are safe.
	mtx sync.Mutex
	// Docker client to access docker apis.
	dockerClient *dockerclient.Client
	// Manage pod metadata.
	podMetaManager kubeletpod.MetaManager
	// Manage pod runtime data.
	podRuntimeManager kubeletpod.RuntimeManager
}

// newKubelet creates a new Kubelet object.
func NewKubelet() Kubelet {
	// Create docker client.
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		glog.Fatal(err)
	}
	return &dockerKubelet{
		dockerClient:      cli,
		podMetaManager:    kubeletpod.NewMetaManager(),
		podRuntimeManager: kubeletpod.NewRuntimeManager(),
	}
}

func (kl *dockerKubelet) ConnectToServer(apiserverStatus *core.ApiserverStatus) error {
	if kl.apiClient != nil {
		return errors.New("api server client alreay exists")
	}
	apiClient, err := client.NewKubeletClient(apiserverStatus.IP, apiserverStatus.Port)
	if err != nil {
		return err
	}
	kl.apiClient = apiClient
	glog.Infof("connected to api server at %v:%v", apiserverStatus.IP, apiserverStatus.Port)

	// Get CoreDNS IP from etcd.
	var dnsIP string
	etcdClient, err := etcd.New(etcd.Config{
		Endpoints:   []string{fmt.Sprintf("%v:%v", apiserverStatus.IP, etcdPort)},
		DialTimeout: etcdDialTimeout,
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), etcdDialTimeout)
	resp, err := etcdClient.Get(ctx, dnsIPKey)
	cancel()
	if err != nil {
		return err
	}
	for _, kv := range resp.Kvs {
		if string(kv.Key) == dnsIPKey {
			dnsIP = string(kv.Value)
			if net.ParseIP(dnsIP) == nil {
				glog.Errorf("got invalid DNS server IP from etcd: %v, DNS might not work", dnsIP)
			} else {
				glog.Infof("got DNS server IP: %v", dnsIP)
			}
		}
	}
	kl.dnsIP = dnsIP

	return nil
}

func (kl *dockerKubelet) GetPods() []*core.Pod {
	return kl.podMetaManager.Pods()
}

func (kl *dockerKubelet) GetPodByName(name string) (*core.Pod, bool) {
	return kl.podMetaManager.PodByName(name)
}

func (kl *dockerKubelet) AddPod(ctx context.Context, pod *core.Pod) error {
	if _, ok := kl.podMetaManager.PodByName(pod.Name); ok {
		err := fmt.Errorf("pod exists already: %v", pod.Name)
		glog.Error(err.Error())
		return err
	}

	// Update pod status as Running.
	// Note that running does not mean ready.
	kl.podMetaManager.AddPod(pod)
	// TODO: Defer broadcasting condition variable. CV and mtx should be a member of the kubelet.

	// Start sandbox pause container. If sandbox container fails to start, then no other container
	// could get started and the pod will be marked as failed.
	if err := kl.runPodSandBox(ctx, pod); err != nil {
		glog.Errorf("cannot create sandbox: %v", err.Error())
		pod.Status.Phase = core.PodFailed
		kl.apiClient.UpdatePodStatus(pod)
		return err
	}

	// Start user containers. Here we won't care about whether the container has started successfully.
	// This will be checked by the monitor.
	for _, c := range pod.Spec.Containers {
		err := kl.runPodContainer(ctx, pod, &c)
		if err != nil {
			return err
		}
	}

	// Notify API server.
	pod.Status.Phase = core.PodReady
	kl.apiClient.UpdatePodStatus(pod)

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
			ports[dockernat.Port(fmt.Sprintf("%v/tcp", p))] = struct{}{}
		}
	}

	pauseName := core.GetPodSpecificPauseName(pod)
	resp, err := cli.ContainerCreate(ctx, &dockercontainer.Config{
		Image:        pauseImage,
		ExposedPorts: ports,
	}, &dockercontainer.HostConfig{
		DNS:     []string{kl.dnsIP},
		IpcMode: "shareable",
	}, nil, nil, pauseName)
	if err != nil {
		return err
	}
	kl.podRuntimeManager.AddPodSandBox(pod, resp.ID)

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

// runPodContainer runs a container and joins it to pod's pause container.
func (kl *dockerKubelet) runPodContainer(ctx context.Context, pod *core.Pod, c *core.Container) error {
	pauseContainerName := core.GetPodSpecificPauseName(pod)
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
		vBinds = append(vBinds, fmt.Sprintf("%v:%v", core.GetPodSpecificName(pod, m.Name), m.MountPath))
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
	}, nil, nil, core.GetPodSpecificName(pod, c.Name))
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

func (kl *dockerKubelet) DeletePodByName(ctx context.Context, name string) (err error) {
	pod, ok := kl.podMetaManager.PodByName(name)
	defer func(err error) {
		var success bool = err == nil
		if _, err := kl.apiClient.NotifyPodDeletion(success, pod); err != nil {
			glog.Errorf("failed to notify apiserver of pod deletion: %v", err.Error())
		}
	}(err)

	if !ok {
		err := fmt.Errorf("pod does not exist: %v", name)
		glog.Error(err.Error())
		return err
	}

	// TODO: Wait until pod is done adding. By doing while () { cv.Wait() }
	kl.podMetaManager.DeletePodByName(name)

	// Remove user containers.
	containers, _ := kl.podRuntimeManager.ContainersByPod(pod)
	for _, c := range containers {
		err := kl.dockerClient.ContainerStop(ctx, c, nil)
		if err != nil {
			glog.Errorf("cannot stop container: %v", err.Error())
			return err
		}
		err = kl.dockerClient.ContainerRemove(ctx, c, dockertypes.ContainerRemoveOptions{})
		if err != nil {
			glog.Errorf("cannot remove container: %v", err.Error())
			return err
		}
	}

	// Remove pause container.
	pauseName, ok := kl.podRuntimeManager.SandBoxByPod(pod)
	if !ok {
		// This is not necessarily an internal error.
		// Pause container may fail to launch for all sorts of reasons.
		glog.Warningf("cannot find sandbox for pod: %v", pod.Name)
	} else {
		err := kl.dockerClient.ContainerStop(ctx, pauseName, nil)
		if err != nil {
			glog.Errorf("cannot stop pause container: %v", err.Error())
			return err
		}
		err = kl.dockerClient.ContainerRemove(ctx, pauseName, dockertypes.ContainerRemoveOptions{})
		if err != nil {
			glog.Errorf("cannot remove pause container: %v", err.Error())
			return err
		}
	}

	kl.podRuntimeManager.DeletePodContainers(pod)

	// Remove volumes.
	volumes, _ := kl.podRuntimeManager.VolumesByPod(pod)
	for _, v := range volumes {
		err := kl.dockerClient.VolumeRemove(ctx, v, true)
		if err != nil {
			glog.Errorf("cannot remove volume: %v", err.Error())
			return err
		}
	}

	kl.podRuntimeManager.DeletePodVolumes(pod)

	return nil
}

func (kl *dockerKubelet) StartCAdvisor() error {
	cli := kl.dockerClient
	ctx := context.Background()

	// Pull image
	out, err := cli.ImagePull(ctx, cadvisorImage, dockertypes.ImagePullOptions{})
	if err != nil {
		glog.Errorf("fail to create cadvisor container: %v", err)
		return err
	}
	_, err = io.ReadAll(out)
	if err != nil {
		glog.Errorf("fail to create cadvisor container: %v", err)
		return nil
	}
	defer func(out io.ReadCloser) {
		err := out.Close()
		if err != nil {
			glog.Error(err)
		}
	}(out)

	// Create cadvisor container
	vBinds := []string{
		"/:/rootfs:ro",
		"/var/run:/var/run:ro",
		"/sys:/sys:ro",
		"/var/lib/docker/:/var/lib/docker:ro",
		"/dev/disk/:/dev/disk:ro",
	}
	exposedPort := dockernat.Port(fmt.Sprintf("%d/tcp", cadvisorPort))
	resp, err := cli.ContainerCreate(ctx, &dockercontainer.Config{
		Image: cadvisorImage,
		ExposedPorts: dockernat.PortSet{
			exposedPort: struct{}{},
		},
	}, &dockercontainer.HostConfig{
		Binds:      vBinds,
		Privileged: true,
		PortBindings: dockernat.PortMap{
			exposedPort: []dockernat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: fmt.Sprint(cadvisorPort),
				},
			},
		},
	}, nil, nil, cadvisorName)
	if err != nil {
		return err
	}

	// Start cadvisor container
	err = cli.ContainerStart(ctx, resp.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		glog.Errorf("fail to start cadvisor container: %v", err)
		return err
	}

	glog.Infoln("successfully starts cadvisor container")
	return nil
}
