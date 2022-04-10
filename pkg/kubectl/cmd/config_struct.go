package cmd

import (
	pb "p9t.io/kuberboat/pkg/proto"
)

type ConfigKind struct {
	Kind string
}

type VolumeMount struct {
	Name      string
	MountPath string `yaml:"mountPath"`
}

type Container struct {
	Name         string
	Image        string
	Ports        []uint32
	Resources    map[string]uint32
	Commands     []string
	VolumeMounts []VolumeMount `yaml:"volumeMounts"`
}

type Pod struct {
	Name       string
	Labels     map[string]string
	Containers []Container
	Volumes    []string
}

func (pod *Pod) Convert2RPCPod() *pb.Pod {
	rpcPod := &pb.Pod{}
	rpcPod.Name = pod.Name
	labels := make([]*pb.Label, 0, len(pod.Labels))
	for k, v := range pod.Labels {
		labels = append(labels, &pb.Label{Name: k, Value: v})
	}
	rpcPod.Labels = labels
	containers := make([]*pb.Container, 0, len(pod.Containers))
	for _, container := range pod.Containers {
		resouceRequirements := make([]*pb.ResourceRequirement, 0, len(container.Resources))
		for k, v := range container.Resources {
			resouceRequirements = append(
				resouceRequirements,
				&pb.ResourceRequirement{Resource: k, Requirement: v})
		}
		volumeMounts := make([]*pb.VolumeMount, 0, len(container.VolumeMounts))
		for _, volumeMount := range container.VolumeMounts {
			volumeMounts = append(volumeMounts, &pb.VolumeMount{
				Name: volumeMount.Name, MountPath: volumeMount.MountPath,
			})
		}
		containers = append(containers, &pb.Container{
			Name: container.Name, Image: container.Image, Ports: container.Ports,
			ResourceRequirements: resouceRequirements, Command: container.Commands,
			VolumeMounts: volumeMounts})
	}
	rpcPod.Containers = containers
	rpcPod.Volumes = pod.Volumes
	return rpcPod
}
