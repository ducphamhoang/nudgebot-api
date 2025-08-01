package events

import (
	"context"
	"fmt"
	"sync"

	eventbus "github.com/asaskevich/EventBus"
	"go.uber.org/zap"
)

// EventBus defines the interface for publishing and subscribing to events
type EventBus interface {
	Publish(topic string, data interface{}) error
	Subscribe(topic string, handler interface{}) error
	Unsubscribe(topic string, handler interface{}) error
	Close() error
}

// eventBus wraps the EventBus library with additional functionality
type eventBus struct {
	bus    eventbus.Bus
	logger *zap.Logger
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
	closed bool
}

// NewEventBus creates a new event bus instance
func NewEventBus(logger *zap.Logger) EventBus {
	ctx, cancel := context.WithCancel(context.Background())

	return &eventBus{
		bus:    eventbus.New(),
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Publish publishes an event to the specified topic
func (eb *eventBus) Publish(topic string, data interface{}) error {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if eb.closed {
		return fmt.Errorf("event bus is closed")
	}

	eb.logger.Debug("Publishing event",
		zap.String("topic", topic),
		zap.Any("data", data))

	eb.bus.Publish(topic, data)
	return nil
}

// Subscribe subscribes to events on the specified topic
func (eb *eventBus) Subscribe(topic string, handler interface{}) error {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if eb.closed {
		return fmt.Errorf("event bus is closed")
	}

	eb.logger.Debug("Subscribing to topic", zap.String("topic", topic))

	return eb.bus.Subscribe(topic, handler)
}

// Unsubscribe unsubscribes from events on the specified topic
func (eb *eventBus) Unsubscribe(topic string, handler interface{}) error {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if eb.closed {
		return fmt.Errorf("event bus is closed")
	}

	eb.logger.Debug("Unsubscribing from topic", zap.String("topic", topic))

	return eb.bus.Unsubscribe(topic, handler)
}

// Close gracefully shuts down the event bus
func (eb *eventBus) Close() error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return nil
	}

	eb.logger.Info("Closing event bus")
	eb.closed = true
	eb.cancel()
	eb.wg.Wait()

	return nil
}
