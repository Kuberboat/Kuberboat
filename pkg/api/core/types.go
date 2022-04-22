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
	Ports []uint16
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
	// DeploymentType means the resource is a deployment.
	DeploymentType = "Deployment"
	// NodeType means the resource is a node.
	NodeType = "Node"
	// ServiceType means the resource is a service
	ServiceType = "Service"
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
	PodReady PodPhase = "Ready"
	// PodSucceeded means that all containers in the pod have voluntarily terminated
	// with a container exit code of 0, and the system is not going to restart any of these containers.
	PodSucceeded PodPhase = "Succeeded"
	// PodFailed means that all containers in the pod have terminated, and at least one container has
	// terminated in a failure (exited with a non-zero exit code or was stopped by the system).
	PodFailed PodPhase = "Failed"
)

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
	Kind
	// Standard object's metadata.
	ObjectMeta `yaml:"metadata"`
	// Specification of the desired behavior of the pod.
	// Entirely populated by the user, though there might be default values.
	// Currently the only source of a PodSpec is a yaml file.
	Spec PodSpec
	// Most recently observed status of the pod.
	// Entirely populated by the system.
	Status PodStatus
}

/// ServicePort is a set of ports that describes the port mapping of the service.
type ServicePort struct {
	/// The port that will be exposed on the service. Pods in the cluster can find the
	/// service via <ClusterIP>:<Port>.
	Port uint16
	/// The port exposed by pods that are selected by this service. <ClusterIP>:<Port> will
	/// be mapped to this port of the pods in the service. If this is not specified in user
	/// yaml, then default to `Port`.
	TargetPort uint16 `default:"80" yaml:"targetPort"`
}

/// ServiceSpec is the set of properties of a service.
type ServiceSpec struct {
	/// Ports describes the mapping of the port on service cluster IP and the port of inner pods.
	Ports []ServicePort
	/// Selector selects the pods whose labels match with the selector.
	Selector map[string]string
	/// ClusterIP is the virtual IP address of the service and is assigned by the master.
	ClusterIP string
}

/// Service is a named abstraction of software service consisting of several pods. The pods can be
/// found in the cluster through the service abstraction (more specifically, cluster IP).
type Service struct {
	// The type of a service is Service.
	Kind
	// Standard object's metadata.
	ObjectMeta `yaml:"metadata"`
	/// Specification of the desired behavior of the service.
	Spec ServiceSpec
}

// DeploymentSpec is the set of properties of a deployment that can be specified using a yaml file.
type DeploymentSpec struct {
	// Replicas is the desired number of pods.
	Replicas uint32
	// Template is the object that describes the pod that will be created if
	// insufficient replicas are detected.
	Template PodTemplateSpec
}

// PodTemplateSpec describes the data a pod should have when created from a template.
type PodTemplateSpec struct {
	// Standard object's metadata.
	// For PodTemplateSpec, ObjectMeta provides labels and names for the pods created/
	// UUID and CreationTimestamp is unused.
	ObjectMeta `yaml:"metadata"`
	// Specification of the desired behavior of the pod.
	Spec PodSpec
}

// DeploymentStatus holds information about the observed status of a deployment.
type DeploymentStatus struct {
	// Total number of non-terminated pods targeted by this deployment (their labels match the selector).
	Replicas uint32
	// Total number of non-terminated pods targeted by this deployment that have the desired template spec.
	UpdatedReplicas uint32
	// Total number of ready pods targeted by this deployment.
	ReadyReplicas uint32
}

// Deployment is a collection of pods that are monitored. It ensures the number of pods in a deployment is stable.
type Deployment struct {
	// The type of a deployment is Deployment.
	Kind
	// Standard object's metadata.
	// For deployment, Label is unused.
	ObjectMeta `yaml:"metadata"`
	// Specification of the desired behavior of the pod.
	// Entirely populated by the user, though there might be default values..
	// Currently the only source of a PodSpec is a yaml file.
	Spec DeploymentSpec `yaml:"spec"`
	// DeploymentStatus is the most recently observed status of the pod.
	// Entirely populated by the system.
	Status DeploymentStatus
}

// NodeSpec describes the attributes of a node.
type NodeSpec struct {
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
	// Port of the kubelet grpc server on node
	Port uint16
}

// Node represents a host machine where Pods are actually running.
type Node struct {
	// The type of a node is Node.
	Kind
	// Standard object's metadata.
	ObjectMeta `yaml:"metadata"`
	// Specification of the desired behavior of the node.
	Spec NodeSpec
	// Most recently observed status of the node.
	Status NodeStatus
}

// ClusterWithName wraps a cluster with its name
type ClusterWithName struct {
	// Server is the URL of the apiserver, default is localhost
	Server string `default:"localhost"`
	// Port is the port of the apiserver, default is 6443
	Port uint16 `default:"6443"`
	// context use name to specifiy cluster
	Name string
}

// ContextWithName wraps a context with its name
type ContextWithName struct {
	// cluster's name
	Context string
	// context's name
	Name string
}

// Config describes cluster information for kubectl to connect
type Config struct {
	// The type of a config is Config
	Kind string
	// timeout of grpc client
	Duration uint16 `default:"1"`
	// clusters can connect to
	Clusters []ClusterWithName
	// contexts can use
	Contexts []ContextWithName
	// current context
	CurrentContext ContextWithName `yaml:"currentContext"`
}
