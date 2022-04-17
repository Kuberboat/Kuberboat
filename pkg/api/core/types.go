package core

import (
	"time"

	"github.com/google/uuid"
)

// Container corresponds to a container entity described in pod spec.
// A single Docker application container that you want to run within a pod.
type Container struct {
	// Name of the container.
	// When a pod is created, Name of all the pods should be checked against duplication.
	Name string
	// Container image name.
	Image string
	// List of ports to expose from the container.
	Ports []ContainerPort
	// Compute Resources required by this container.
	Resources map[ResourceName]uint64
	// Entrypoint of the container. Equivalent to `docker run --entrypoint ...`.
	// The container image's ENTRYPOINT is used if this is not provided.
	Commands []string
	// Pod volumes to mount into the container's filesystem.
	VolumeMounts []VolumeMount `yaml:"volumeMounts"`
}

// ContainerPort represents a network port in a single container.
// Only TCP is supported.
type ContainerPort struct {
	// Port number to expose on the pod's IP address.
	ContainerPort uint16 `yaml:"containerPort"`
}

type VolumeMount struct {
	// This must match the Name of a Volume.
	Name string
	// Path within the container at which the volume should be mounted.  Must not contain ':'.
	MountPath string `yaml:"mountPath"`
}

// ResourceName is the name identifying various resources in a ResourceList that a single container can use.
type ResourceName string

// These are the valid types of resources that a docker container can be confined to.
const (
	// ResourceCPU represents the number of cores a container can use.
	// Using half a core is not supported.
	ResourceCPU ResourceName = "cpu"
	// ResourceMemory represents the memory in bytes that a container can use.
	ResourceMemory ResourceName = "memory"
)

// Kind specified the category of an object.
type Kind string

// These are valid kinds of an object.
const (
	// PodType means the resource is a pod.
	PodType Kind = "Pod"
)

// PodPhase is a label for the condition of a pod at the current time.
type PodPhase string

// These are the valid statuses of pods.
const (
	// PodPending means the pod has been accepted by the system, but one or more of the containers
	// has not been started. This includes time before being bound to a node, as well as time spent
	// pulling images onto the host.
	PodPending PodPhase = "Pending"
	// PodRunning means the pod has been bound to a node and all the containers have been started.
	// At least one container is still running or is in the process of being restarted.
	PodRunning PodPhase = "Running"
	// PodSucceeded means that all containers in the pod have voluntarily terminated
	// with a container exit code of 0, and the system is not going to restart any of these containers.
	PodSucceeded PodPhase = "Succeeded"
	// PodFailed means that all containers in the pod have terminated, and at least one container has
	// terminated in a failure (exited with a non-zero exit code or was stopped by the system).
	PodFailed PodPhase = "Failed"
)

// TypeMeta describes the type of an individual object in an API response or request.
type TypeMeta struct {
	Kind
}

// ObjectMeta is metadata that all persisted resources must have.
type ObjectMeta struct {
	// The name of an object.
	// Must not be empty.
	Name string
	// Unique identifier of the object. Populated by the system when the owning resource is successfully created.
	// User cannot modify this field.
	UUID uuid.UUID
	// A timestamp representing the server time when this object was created.
	// Can be used to compute up time of a pod.
	CreationTimestamp time.Time
	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	Labels map[string]string
}

// PodSpec is the set of properties of a pod that can be specified using a yaml file.
type PodSpec struct {
	// List of containers belonging to the pod.
	// There must be at least one container in a Pod.
	Containers []Container
	// List of named volumes that can be mounted by containers belonging to the pod.
	Volumes []string
}

// PodStatus represents information about the status of a pod.
type PodStatus struct {
	// The phase of a Pod is a simple, high-level summary of where the Pod is in its lifecycle.
	Phase PodPhase
	// IP address of the host to which the pod is assigned. Empty if not yet scheduled.
	HostIP string
	// IPv4 address assigned to the pod. Empty if not yet allocated.
	PodIP string
}

// Pod is a collection of containers that can run on a host. This resource is created
// by clients and scheduled onto hosts.
type Pod struct {
	// The type of a pod is Pod.
	TypeMeta `yaml:",inline"`
	// Standard object's metadata.
	ObjectMeta `yaml:"metadata"`
	// Specification of the desired behavior of the pod.
	// Entirely populated by the user, though there might be default values..
	// Currently the only source of a PodSpec is a yaml file.
	Spec PodSpec `yaml:",inline"`
	// Most recently observed status of the pod.
	// Entirely populated by the system.
	Status PodStatus
}

// NodeSpec describes the attributes of a node.
type NodeSpec struct {
	// PodCIDR represents the IPV4 range assigned to the node for usage by Pods on the node.
	PodCIDR string
}

// NodePhase is a label for the condition of a node at the current time.
type NodePhase string

// These are the valid statuses of node.
const (
	// NodePending means the node has been created/added by the system, but not configured.
	NodePending NodePhase = "Pending"
	// NodeRunning means the node has been configured and has some components running.
	NodeRunning NodePhase = "Running"
	// NodeTerminated means the node has been removed from the cluster.
	NodeTerminated NodePhase = "Terminated"
)

// NodeConditionType defines node's condition.
type NodeCondition string

// These are valid conditions of node.
const (
	// NodeReady means kubelet is healthy and ready to accept pods.
	NodeReady NodeCondition = "Ready"
	// NodeUnavailable means the node is unavailable for use.
	NodeUnavailable NodeCondition = "Unavailable"
)

// NodeStatus represents information about the status of a node.
type NodeStatus struct {
	// NodePhase is a simple, high-level summary of where the node is in its lifecycle.
	Phase NodePhase
	// NodeCondition indicates whether the node is available or not.
	Condition NodeCondition
	// Address is the IPV4 address of the node. Currently, we only support node address
	// represented in IPV4.
	Address string
}

// Node represents a host machine where Pods are actually running.
type Node struct {
	// The type of a node is Node.
	TypeMeta
	// Standard object's metadata.
	ObjectMeta
	// Specification of the desired behavior of the node.
	Spec NodeSpec
	// Most recently observed status of the node.
	Status NodeStatus
}
