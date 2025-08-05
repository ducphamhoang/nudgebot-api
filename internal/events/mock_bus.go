package events

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

//go:generate mockgen -source=../events/bus.go -destination=event_bus_mocks.go -package=events

// MockEventBus provides an in-memory implementation of EventBus for testing
type MockEventBus struct {
	subscriptions    map[string][]interface{}
	publishedEvents  map[string][]interface{}
	mutex            sync.RWMutex
	callbackHandlers map[string]func(interface{})
	errors           []error
	synchronousMode  bool
}

// NewMockEventBus creates a new MockEventBus instance
func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		subscriptions:    make(map[string][]interface{}),
		publishedEvents:  make(map[string][]interface{}),
		callbackHandlers: make(map[string]func(interface{})),
	}
}

// Subscribe implements the EventBus interface
func (m *MockEventBus) Subscribe(topic string, handler interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.subscriptions[topic] == nil {
		m.subscriptions[topic] = make([]interface{}, 0)
	}
	m.subscriptions[topic] = append(m.subscriptions[topic], handler)

	// Store callback handler for testing
	if callbackHandler, ok := handler.(func(interface{})); ok {
		m.callbackHandlers[topic] = callbackHandler
	}

	return nil
}

// Unsubscribe implements the EventBus interface
func (m *MockEventBus) Unsubscribe(topic string, handler interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if handlers, exists := m.subscriptions[topic]; exists {
		// Collect indices to remove (iterate backwards to avoid index issues)
		var indicesToRemove []int
		for i := len(handlers) - 1; i >= 0; i-- {
			// Simple comparison - in a real implementation this would need better handler matching
			if handlers[i] == handler {
				indicesToRemove = append(indicesToRemove, i)
			}
		}

		// Remove all matching handlers
		for _, idx := range indicesToRemove {
			handlers = append(handlers[:idx], handlers[idx+1:]...)
		}
		m.subscriptions[topic] = handlers
	}

	return nil
}

// Publish implements the EventBus interface
func (m *MockEventBus) Publish(topic string, event interface{}) error {
	m.mutex.Lock()

	// Store published event
	if m.publishedEvents[topic] == nil {
		m.publishedEvents[topic] = make([]interface{}, 0)
	}
	m.publishedEvents[topic] = append(m.publishedEvents[topic], event)

	// Get handlers to invoke
	var handlersToInvoke []interface{}
	if handlers, exists := m.subscriptions[topic]; exists {
		handlersToInvoke = make([]interface{}, len(handlers))
		copy(handlersToInvoke, handlers)
	}

	m.mutex.Unlock()

	// Trigger handlers outside of the mutex to avoid deadlocks
	for _, handler := range handlersToInvoke {
		if m.synchronousMode {
			// Run synchronously for testing
			m.invokeHandler(handler, event)
		} else {
			// Run asynchronously
			go m.invokeHandler(handler, event)
		}
	}

	return nil
}

// Close implements the EventBus interface
func (m *MockEventBus) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Clear all subscriptions and events
	m.subscriptions = make(map[string][]interface{})
	m.publishedEvents = make(map[string][]interface{})
	m.callbackHandlers = make(map[string]func(interface{}))

	return nil
}

// SetSynchronousMode enables or disables synchronous event handling
func (m *MockEventBus) SetSynchronousMode(enabled bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.synchronousMode = enabled
}

// GetPublishedEvents returns published events for a topic
func (m *MockEventBus) GetPublishedEvents(topic string) []interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if events, exists := m.publishedEvents[topic]; exists {
		// Return a copy to prevent race conditions
		result := make([]interface{}, len(events))
		copy(result, events)
		return result
	}

	return []interface{}{}
}

// GetSubscriberCount returns the number of subscribers for a topic
func (m *MockEventBus) GetSubscriberCount(topic string) int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if handlers, exists := m.subscriptions[topic]; exists {
		return len(handlers)
	}

	return 0
}

// ClearEvents resets all published events
func (m *MockEventBus) ClearEvents() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.publishedEvents = make(map[string][]interface{})
}

// WaitForEvent waits for an event to be published on a topic
func (m *MockEventBus) WaitForEvent(topic string, timeout time.Duration) (interface{}, error) {
	startTime := time.Now()

	for {
		events := m.GetPublishedEvents(topic)
		if len(events) > 0 {
			return events[len(events)-1], nil // Return latest event
		}

		if time.Since(startTime) > timeout {
			return nil, &TimeoutError{Topic: topic, Timeout: timeout}
		}

		time.Sleep(10 * time.Millisecond) // Small delay to prevent busy waiting
	}
}

// SimulateEventDelivery manually triggers event handlers for testing
func (m *MockEventBus) SimulateEventDelivery(topic string, event interface{}) {
	m.mutex.RLock()
	handlers := m.subscriptions[topic]
	m.mutex.RUnlock()

	for _, handler := range handlers {
		m.invokeHandler(handler, event)
	}
}

// invokeHandler safely invokes an event handler
func (m *MockEventBus) invokeHandler(handler interface{}, event interface{}) {
	defer func() {
		if r := recover(); r != nil {
			// Log panic details for debugging
			m.mutex.Lock()
			m.errors = append(m.errors, fmt.Errorf("handler panic: %v", r))
			m.mutex.Unlock()
		}
	}()

	handlerInvoked := false
	switch h := handler.(type) {
	case func(MessageReceived):
		if e, ok := event.(MessageReceived); ok {
			h(e)
			handlerInvoked = true
		}
	case func(TaskParsed):
		if e, ok := event.(TaskParsed); ok {
			h(e)
			handlerInvoked = true
		}
	case func(TaskCreated):
		if e, ok := event.(TaskCreated); ok {
			h(e)
			handlerInvoked = true
		}
	case func(TaskListRequested):
		if e, ok := event.(TaskListRequested); ok {
			h(e)
			handlerInvoked = true
		}
	case func(TaskListResponse):
		if e, ok := event.(TaskListResponse); ok {
			h(e)
			handlerInvoked = true
		}
	case func(TaskActionRequested):
		if e, ok := event.(TaskActionRequested); ok {
			h(e)
			handlerInvoked = true
		}
	case func(TaskActionResponse):
		if e, ok := event.(TaskActionResponse); ok {
			h(e)
			handlerInvoked = true
		}
	case func(ReminderDue):
		if e, ok := event.(ReminderDue); ok {
			h(e)
			handlerInvoked = true
		}
	case func(interface{}):
		h(event)
		handlerInvoked = true
	}

	// Log type mismatches for debugging
	if !handlerInvoked {
		m.mutex.Lock()
		m.errors = append(m.errors, fmt.Errorf("type mismatch: handler type does not match event type %T", event))
		m.mutex.Unlock()
	}
}

// MockEventHandler provides configurable event handling for testing
type MockEventHandler struct {
	CallCount       int
	LastEvent       interface{}
	ShouldError     bool
	ErrorMessage    string
	ProcessingDelay time.Duration
	mutex           sync.Mutex
}

// NewMockEventHandler creates a new MockEventHandler
func NewMockEventHandler() *MockEventHandler {
	return &MockEventHandler{}
}

// Handle processes an event with configurable behavior
func (h *MockEventHandler) Handle(event interface{}) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.CallCount++
	h.LastEvent = event

	if h.ProcessingDelay > 0 {
		time.Sleep(h.ProcessingDelay)
	}

	if h.ShouldError {
		return &HandlerError{Message: h.ErrorMessage}
	}

	return nil
}

// GetCallCount returns the number of times the handler was called
func (h *MockEventHandler) GetCallCount() int {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.CallCount
}

// GetLastEvent returns the last event processed
func (h *MockEventHandler) GetLastEvent() interface{} {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.LastEvent
}

// Reset resets the handler state
func (h *MockEventHandler) Reset() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.CallCount = 0
	h.LastEvent = nil
	h.ShouldError = false
	h.ErrorMessage = ""
	h.ProcessingDelay = 0
}

// Factory methods for creating test events

// CreateMessageReceivedEvent creates a test MessageReceived event
func CreateMessageReceivedEvent(userID, chatID, text string) MessageReceived {
	return MessageReceived{
		Event:       NewEvent(),
		UserID:      userID,
		ChatID:      chatID,
		MessageText: text,
	}
}

// CreateTaskParsedEvent creates a test TaskParsed event
func CreateTaskParsedEvent(userID string, taskData ParsedTask) TaskParsed {
	return TaskParsed{
		Event:      NewEvent(),
		UserID:     userID,
		ParsedTask: taskData,
	}
}

// CreateReminderDueEvent creates a test ReminderDue event
func CreateReminderDueEvent(taskID, userID, chatID string) ReminderDue {
	return ReminderDue{
		Event:  NewEvent(),
		TaskID: taskID,
		UserID: userID,
		ChatID: chatID,
	}
}

// CreateTaskActionEvent creates a test TaskActionRequested event
func CreateTaskActionEvent(userID, chatID, taskID, action string) TaskActionRequested {
	return TaskActionRequested{
		Event:  NewEvent(),
		UserID: userID,
		ChatID: chatID,
		TaskID: taskID,
		Action: action,
	}
}

// CreateTaskListRequestedEvent creates a test TaskListRequested event
func CreateTaskListRequestedEvent(userID, chatID string) TaskListRequested {
	return TaskListRequested{
		Event:  NewEvent(),
		UserID: userID,
		ChatID: chatID,
	}
}

// CreateTaskCreatedEvent creates a test TaskCreated event
func CreateTaskCreatedEvent(taskID, userID, title, priority string) TaskCreated {
	return TaskCreated{
		Event:     NewEvent(),
		TaskID:    taskID,
		UserID:    userID,
		Title:     title,
		Priority:  priority,
		CreatedAt: time.Now(),
	}
}

// Assertion helpers for testing

// AssertEventPublished verifies that an event was published
func AssertEventPublished(t *testing.T, mockBus *MockEventBus, topic string, expectedEvent interface{}) {
	events := mockBus.GetPublishedEvents(topic)
	if len(events) == 0 {
		t.Errorf("Expected event to be published on topic %s, but no events found", topic)
		return
	}

	// Check if any of the published events match the expected event
	found := false
	for _, event := range events {
		if eventsEqual(event, expectedEvent) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected event %+v not found in published events for topic %s", expectedEvent, topic)
	}
}

// AssertEventCount verifies the number of events published on a topic
func AssertEventCount(t *testing.T, mockBus *MockEventBus, topic string, expectedCount int) {
	events := mockBus.GetPublishedEvents(topic)
	if len(events) != expectedCount {
		t.Errorf("Expected %d events on topic %s, but got %d", expectedCount, topic, len(events))
	}
}

// AssertEventHandlerCalled verifies that a mock handler was called the expected number of times
func AssertEventHandlerCalled(t *testing.T, handler *MockEventHandler, expectedCalls int) {
	if handler.GetCallCount() != expectedCalls {
		t.Errorf("Expected handler to be called %d times, but was called %d times", expectedCalls, handler.GetCallCount())
	}
}

// AssertSubscriberCount verifies the number of subscribers for a topic
func AssertSubscriberCount(t *testing.T, mockBus *MockEventBus, topic string, expectedCount int) {
	count := mockBus.GetSubscriberCount(topic)
	if count != expectedCount {
		t.Errorf("Expected %d subscribers for topic %s, but got %d", expectedCount, topic, count)
	}
}

// Error types for testing

// TimeoutError represents a timeout waiting for an event
type TimeoutError struct {
	Topic   string
	Timeout time.Duration
}

func (e *TimeoutError) Error() string {
	return "timeout waiting for event on topic " + e.Topic
}

// HandlerError represents an error from a mock handler
type HandlerError struct {
	Message string
}

func (e *HandlerError) Error() string {
	return e.Message
}

// Helper functions

// eventsEqual compares two events for equality (basic comparison)
func eventsEqual(a, b interface{}) bool {
	// This is a simple implementation - in a real scenario you might want
	// to use reflection or implement custom comparison logic
	switch eventA := a.(type) {
	case MessageReceived:
		if eventB, ok := b.(MessageReceived); ok {
			return eventA.UserID == eventB.UserID &&
				eventA.ChatID == eventB.ChatID &&
				eventA.MessageText == eventB.MessageText
		}
	case TaskParsed:
		if eventB, ok := b.(TaskParsed); ok {
			return eventA.UserID == eventB.UserID &&
				eventA.ParsedTask.Title == eventB.ParsedTask.Title
		}
	case TaskActionRequested:
		if eventB, ok := b.(TaskActionRequested); ok {
			return eventA.UserID == eventB.UserID &&
				eventA.TaskID == eventB.TaskID &&
				eventA.Action == eventB.Action
		}
		// Add more event type comparisons as needed
	}

	return false
}
