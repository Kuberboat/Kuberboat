package apiserver

import "p9t.io/kuberboat/pkg/api/core"

type EventType int

const (
	PodDeletion EventType = iota
	PodReady
	PodFail
	PodSucceed
)

// Event is an event that happens on any kind of resources, and can be handled by EventSubscriber.
// An event can be handled by multiple subscribers, but each subscriber can only handle it once.
type Event interface {
	Type() EventType
}

// PodDeletionEvent marks the deletion of a pod.
type PodDeletionEvent struct {
	// Pod is a snapshot of the deleted pod's last observed state before deletion.
	Pod *core.Pod
	// PodLegacy provides information such as which deployment this pod belonged to, etc.
	PodLegacy *PodLegacy
}

func (*PodDeletionEvent) Type() EventType {
	return PodDeletion
}

// PodReadyEvent means the a pod has entered phase PodReady.
type PodReadyEvent struct {
	// PodName is the name of the pod that entered
	PodName string
}

func (*PodReadyEvent) Type() EventType {
	return PodReady
}

// PodFailEvent means a pod has entered phase PodFailed.
type PodFailEvent struct {
	PodName string
}

func (*PodFailEvent) Type() EventType {
	return PodFail
}

// PodSucceedEvent means a pod successfully exit.
type PodSucceedEvent struct {
	PodName string
}

func (*PodSucceedEvent) Type() EventType {
	return PodSucceed
}

// More events...
