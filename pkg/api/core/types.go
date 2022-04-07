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
	Resources []ResourceRequirements
	// Entrypoint of the container. Equivalent to `docker run --entrypoint ...`.
	// The container image's ENTRYPOINT is used if this is not provided.
	Command []string
	// Pod volumes to mount into the container's filesystem.
	VolumeMounts []VolumeMount
}

// ContainerPort represents a network port in a single container.
// Only TCP is supported.
type ContainerPort struct {
	// Port number to expose on the pod's IP address.
	ContainerPort uint16
}

// Volume represents a named volume in a pod that may be accessed by any container in the pod.
// Only supports on-disk EmptyDir.
type Volume struct {
	// Name of the volume.
	// Must be unique within a pod.
	// Only supports host path volume.
	Name string
	// VolumeSource represents the location and type of the mounted volume.
	// If not specified, the Volume is implied to be an EmptyDir.
	VolumeSource
}

// VolumeSource represents the source of a volume to mount.
// Only one of its members may be specified.
type VolumeSource struct {
	// EmptyDir represents a temporary directory that shares a pod's lifetime.
	EmptyDir *EmptyDirVolumeSource
}

// EmptyDirVolumeSource is the representation of EmptyDir PV type.
// Currently not configurable.
type EmptyDirVolumeSource struct {
}

type VolumeMount struct {
	// This must match the Name of a Volume.
	Name string
	// Path within the container at which the volume should be mounted.  Must not contain ':'.
	MountPath string
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

// ResourceRequirements describes the compute resource requirements of a single container.
type ResourceRequirements struct {
	// Limits describes the maximum amount of compute resources allowed.
	Limits map[ResourceName]uint64
}

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
	// List of volumes that can be mounted by containers belonging to the pod.
	Volumes []Volume
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
	TypeMeta
	// Standard object's metadata.
	ObjectMeta
	// Specification of the desired behavior of the pod.
	// Entirely populated by the user, though there might be default values..
	// Currently the only source of a PodSpec is a yaml file.
	Spec PodSpec
	// Most recently observed status of the pod.
	// Entirely populated by the system.
	Status PodStatus
}
