package apiserver

import "p9t.io/kuberboat/pkg/api/core"

type EventType int

const (
	PodDeletion EventType = iota
	PodReady
)

var dispatcher eventDispatcher = eventDispatcher{}

// Event is an event that happens on any kind of resources, and can be handled by EventSubscriber.
// An event can be handled by multiple subscribers, but each subscriber can only handle it once.
type Event interface {
	Type() EventType
}

// PodDeletionEvent marks the deletion of a pod.
type PodDeletionEvent struct {
	Pod *core.Pod
}

func (*PodDeletionEvent) Type() EventType {
	return PodDeletion
}

// PodReadyEvent means the a pod has entered phase PodReady.
type PodReadyEvent struct {
	Pod *core.Pod
}

func (*PodReadyEvent) Type() EventType {
	return PodReady
}

// More events...

// EventSubscriber is an object that subscribes to eventDispatcher and handles events that are dispatched to it.
type EventSubscriber interface {
	HandleEvent(event Event)
}

// eventDispatcher keeps track of what objects have subscribed what kind of events.
type eventDispatcher struct {
	subscribers map[EventType][]EventSubscriber
}

// Dispatch dispatches an event to its subscribers.
func Dispatch(event Event) {
	subscribers, present := dispatcher.subscribers[event.Type()]
	if present {
		for _, subscriber := range subscribers {
			subscriber.HandleEvent(event)
		}
	}
}

// SubscribeToEvents provides a means for objects (mainly contronllers) to scribe to a kind of event.
func SubscribeToEvent(obj EventSubscriber, eventType EventType) {
	subscribers, present := dispatcher.subscribers[eventType]
	if present {
		dispatcher.subscribers[eventType] = append(subscribers, obj)
	} else {
		dispatcher.subscribers[eventType] = []EventSubscriber{obj}
	}
}
