package apiserver

var dispatcher eventDispatcher = eventDispatcher{
	subscribers: map[EventType][]EventSubscriber{},
}

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

// SubscribeToEvents provides a means for objects (mainly controllers) to subscribe to a kind of event.
func SubscribeToEvent(obj EventSubscriber, eventType EventType) {
	subscribers, present := dispatcher.subscribers[eventType]
	if present {
		dispatcher.subscribers[eventType] = append(subscribers, obj)
	} else {
		dispatcher.subscribers[eventType] = []EventSubscriber{obj}
	}
}
