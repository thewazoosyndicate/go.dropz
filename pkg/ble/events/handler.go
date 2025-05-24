package events

import (
	"sync"

	"github.com/dropz/dropz/pkg/logger"
)

// EventHandler is a function that handles BLE events
type EventHandler func(event BLEEvent)

// EventEmitter manages BLE event emission and handling
type EventEmitter struct {
	handlers map[EventType][]EventHandler
	mutex    sync.RWMutex
	log      logger.Logger
}

// NewEventEmitter creates a new event emitter
func NewEventEmitter(log logger.Logger) *EventEmitter {
	return &EventEmitter{
		handlers: make(map[EventType][]EventHandler),
		log:      log,
	}
}

// RegisterHandler registers an event handler for a specific event type
func (e *EventEmitter) RegisterHandler(eventType EventType, handler EventHandler) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.handlers[eventType] = append(e.handlers[eventType], handler)
	e.log.Debugf("Registered handler for event type: %s", eventType)
}

// EmitEvent emits an event to all registered handlers
func (e *EventEmitter) EmitEvent(event BLEEvent) {
	e.mutex.RLock()
	handlers, exists := e.handlers[event.Type]
	e.mutex.RUnlock()

	if !exists {
		return
	}

	// Execute handlers in goroutines to avoid blocking
	for _, handler := range handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					e.log.Errorf("Event handler panic: %v", r)
				}
			}()
			h(event)
		}(handler)
	}
}

// UnregisterAllHandlers removes all handlers for a specific event type
func (e *EventEmitter) UnregisterAllHandlers(eventType EventType) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	delete(e.handlers, eventType)
	e.log.Debugf("Unregistered all handlers for event type: %s", eventType)
}
