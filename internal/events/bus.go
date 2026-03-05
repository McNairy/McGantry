// Package events provides an in-process event bus for Gantry.
// It enables decoupled communication between components via publish/subscribe
// with thread-safe fan-out to all registered handlers.
package events

import (
	"sync"
	"time"
)

// EventType identifies the category of an event.
type EventType string

const (
	EntityCreated   EventType = "entity.created"
	EntityUpdated   EventType = "entity.updated"
	EntityDeleted   EventType = "entity.deleted"
	ActionTriggered EventType = "action.triggered"
	ActionCompleted EventType = "action.completed"
	ActionFailed    EventType = "action.failed"
	UserLogin       EventType = "user.login"
)

// Event represents a single occurrence in the system.
type Event struct {
	Type      EventType      `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

// Handler is a function that processes an event.
type Handler func(Event)

// wildcard is a sentinel EventType used internally for SubscribeAll handlers.
const wildcard EventType = "*"

// Bus is an in-process event bus that dispatches events to registered handlers.
// It is safe for concurrent use by multiple goroutines.
type Bus struct {
	mu       sync.RWMutex
	handlers map[EventType][]Handler
}

// New creates a new event bus ready for use.
func New() *Bus {
	return &Bus{
		handlers: make(map[EventType][]Handler),
	}
}

// Subscribe registers a handler for a specific event type.
// The handler will be called in a new goroutine each time a matching event is published.
func (b *Bus) Subscribe(eventType EventType, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// SubscribeAll registers a handler that receives all published events,
// regardless of their type.
func (b *Bus) SubscribeAll(handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[wildcard] = append(b.handlers[wildcard], handler)
}

// Publish dispatches an event to all handlers registered for the event's type,
// as well as all wildcard (SubscribeAll) handlers. Each handler is invoked
// in its own goroutine so publishing never blocks on handler execution.
// If the event's Timestamp is zero, it is set to the current time.
func (b *Bus) Publish(event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	b.mu.RLock()
	// Collect all applicable handlers under the read lock.
	typed := make([]Handler, len(b.handlers[event.Type]))
	copy(typed, b.handlers[event.Type])

	wild := make([]Handler, len(b.handlers[wildcard]))
	copy(wild, b.handlers[wildcard])
	b.mu.RUnlock()

	for _, h := range typed {
		go h(event)
	}
	for _, h := range wild {
		go h(event)
	}
}
