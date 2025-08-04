package events

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEventBus_PublishSubscribe(t *testing.T) {
	tests := []struct {
		name     string
		topic    string
		event    interface{}
		expected interface{}
	}{
		{
			name:     "publish string event",
			topic:    "test.string",
			event:    "test message",
			expected: "test message",
		},
		{
			name:  "publish message received event",
			topic: TopicMessageReceived,
			event: MessageReceived{
				Event:       NewEvent(),
				UserID:      "user123",
				ChatID:      "chat456",
				MessageText: "Hello, world!",
			},
			expected: MessageReceived{
				UserID:      "user123",
				ChatID:      "chat456",
				MessageText: "Hello, world!",
			},
		},
		{
			name:  "publish task created event",
			topic: TopicTaskCreated,
			event: TaskCreated{
				Event:     NewEvent(),
				TaskID:    "task123",
				UserID:    "user456",
				Title:     "Test Task",
				Priority:  "high",
				CreatedAt: time.Now(),
			},
			expected: TaskCreated{
				TaskID:   "task123",
				UserID:   "user456",
				Title:    "Test Task",
				Priority: "high",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			bus := NewEventBus(logger)
			defer bus.Close()

			var receivedEvent interface{}
			var wg sync.WaitGroup
			wg.Add(1)

			// Subscribe to the topic
			handler := func(event interface{}) {
				receivedEvent = event
				wg.Done()
			}

			err := bus.Subscribe(tt.topic, handler)
			require.NoError(t, err)

			// Publish the event
			err = bus.Publish(tt.topic, tt.event)
			require.NoError(t, err)

			// Wait for the event to be received
			done := make(chan bool)
			go func() {
				wg.Wait()
				done <- true
			}()

			select {
			case <-done:
				// Verify the received event
				assert.NotNil(t, receivedEvent)
				switch expected := tt.expected.(type) {
				case string:
					assert.Equal(t, expected, receivedEvent)
				case MessageReceived:
					received, ok := receivedEvent.(MessageReceived)
					assert.True(t, ok)
					assert.Equal(t, expected.UserID, received.UserID)
					assert.Equal(t, expected.ChatID, received.ChatID)
					assert.Equal(t, expected.MessageText, received.MessageText)
				case TaskCreated:
					received, ok := receivedEvent.(TaskCreated)
					assert.True(t, ok)
					assert.Equal(t, expected.TaskID, received.TaskID)
					assert.Equal(t, expected.UserID, received.UserID)
					assert.Equal(t, expected.Title, received.Title)
					assert.Equal(t, expected.Priority, received.Priority)
				}
			case <-time.After(1 * time.Second):
				t.Error("Timeout waiting for event")
			}
		})
	}
}

func TestEventBus_ConcurrentAccess(t *testing.T) {
	logger := zap.NewNop()
	bus := NewEventBus(logger)
	defer bus.Close()

	const numGoroutines = 10
	const numEvents = 100

	var wg sync.WaitGroup
	var mu sync.Mutex
	receivedEvents := make([]interface{}, 0)

	// Subscribe to the topic
	handler := func(event interface{}) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
	}

	err := bus.Subscribe("test.concurrent", handler)
	require.NoError(t, err)

	// Start multiple publishers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numEvents; j++ {
				event := struct {
					PublisherID int
					EventID     int
				}{
					PublisherID: id,
					EventID:     j,
				}
				err := bus.Publish("test.concurrent", event)
				assert.NoError(t, err)
			}
		}(i)
	}

	// Start multiple subscribers
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			subscriberHandler := func(event interface{}) {
				// Just receive the event
			}
			topic := "test.concurrent.subscriber"
			err := bus.Subscribe(topic, subscriberHandler)
			assert.NoError(t, err)
			time.Sleep(100 * time.Millisecond)
			err = bus.Unsubscribe(topic, subscriberHandler)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Allow some time for all events to be processed
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	totalReceived := len(receivedEvents)
	mu.Unlock()

	// We expect to receive all published events
	expectedTotal := numGoroutines * numEvents
	assert.Equal(t, expectedTotal, totalReceived)
}

func TestEventBus_Unsubscribe(t *testing.T) {
	logger := zap.NewNop()
	bus := NewEventBus(logger)
	defer bus.Close()

	receivedCount := 0
	var mu sync.Mutex

	handler := func(event interface{}) {
		mu.Lock()
		receivedCount++
		mu.Unlock()
	}

	// Subscribe to topic
	err := bus.Subscribe("test.unsubscribe", handler)
	require.NoError(t, err)

	// Publish first event
	err = bus.Publish("test.unsubscribe", "event1")
	require.NoError(t, err)

	// Allow time for processing
	time.Sleep(50 * time.Millisecond)

	// Unsubscribe
	err = bus.Unsubscribe("test.unsubscribe", handler)
	require.NoError(t, err)

	// Publish second event (should not be received)
	err = bus.Publish("test.unsubscribe", "event2")
	require.NoError(t, err)

	// Allow time for processing
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	count := receivedCount
	mu.Unlock()

	// Should have received only the first event
	assert.Equal(t, 1, count)
}

func TestEventBus_ClosedBus(t *testing.T) {
	logger := zap.NewNop()
	bus := NewEventBus(logger)

	// Close the bus
	err := bus.Close()
	require.NoError(t, err)

	// Attempt to publish to closed bus
	err = bus.Publish("test.closed", "event")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event bus is closed")

	// Attempt to subscribe to closed bus
	handler := func(event interface{}) {}
	err = bus.Subscribe("test.closed", handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event bus is closed")

	// Attempt to unsubscribe from closed bus
	err = bus.Unsubscribe("test.closed", handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event bus is closed")

	// Closing again should not error
	err = bus.Close()
	assert.NoError(t, err)
}

func TestEventBus_HandlerErrors(t *testing.T) {
	logger := zap.NewNop()
	bus := NewEventBus(logger)
	defer bus.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	// Handler that panics
	panicHandler := func(event interface{}) {
		defer wg.Done()
		panic("handler panic")
	}

	// Normal handler
	normalHandler := func(event interface{}) {
		defer wg.Done()
		// Normal processing
	}

	err := bus.Subscribe("test.errors", panicHandler)
	require.NoError(t, err)

	err = bus.Subscribe("test.errors", normalHandler)
	require.NoError(t, err)

	// Publish event that will trigger both handlers
	err = bus.Publish("test.errors", "test event")
	require.NoError(t, err)

	// Wait for handlers to complete
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Both handlers should have been called despite the panic
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for handlers")
	}
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	logger := zap.NewNop()
	bus := NewEventBus(logger)
	defer bus.Close()

	const numSubscribers = 5
	var wg sync.WaitGroup
	wg.Add(numSubscribers)

	receivedEvents := make([]interface{}, 0, numSubscribers)
	var mu sync.Mutex

	// Create multiple subscribers for the same topic
	for i := 0; i < numSubscribers; i++ {
		handler := func(event interface{}) {
			mu.Lock()
			receivedEvents = append(receivedEvents, event)
			mu.Unlock()
			wg.Done()
		}

		err := bus.Subscribe("test.multiple", handler)
		require.NoError(t, err)
	}

	// Publish one event
	testEvent := "broadcast event"
	err := bus.Publish("test.multiple", testEvent)
	require.NoError(t, err)

	// Wait for all subscribers to receive the event
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		mu.Lock()
		count := len(receivedEvents)
		mu.Unlock()
		assert.Equal(t, numSubscribers, count)
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for all subscribers")
	}
}

func TestEventBus_EventDelivery(t *testing.T) {
	logger := zap.NewNop()
	bus := NewEventBus(logger)
	defer bus.Close()

	events := []interface{}{
		"string event",
		123,
		map[string]interface{}{"key": "value"},
		MessageReceived{
			Event:       NewEvent(),
			UserID:      "test_user",
			ChatID:      "test_chat",
			MessageText: "test message",
		},
	}

	receivedEvents := make([]interface{}, 0, len(events))
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(events))

	handler := func(event interface{}) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
		wg.Done()
	}

	err := bus.Subscribe("test.delivery", handler)
	require.NoError(t, err)

	// Publish all events
	for _, event := range events {
		err = bus.Publish("test.delivery", event)
		require.NoError(t, err)
	}

	// Wait for all events to be received
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		mu.Lock()
		count := len(receivedEvents)
		mu.Unlock()
		assert.Equal(t, len(events), count)
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for event delivery")
	}
}

func TestEventBus_SubscriptionManagement(t *testing.T) {
	logger := zap.NewNop()
	bus := NewEventBus(logger)
	defer bus.Close()

	handler1 := func(event interface{}) {}
	handler2 := func(event interface{}) {}
	handler3 := func(event interface{}) {}

	// Subscribe multiple handlers
	err := bus.Subscribe("test.mgmt", handler1)
	require.NoError(t, err)

	err = bus.Subscribe("test.mgmt", handler2)
	require.NoError(t, err)

	err = bus.Subscribe("test.mgmt", handler3)
	require.NoError(t, err)

	// Subscribe to different topics
	err = bus.Subscribe("test.other", handler1)
	require.NoError(t, err)

	// Unsubscribe one handler
	err = bus.Unsubscribe("test.mgmt", handler2)
	require.NoError(t, err)

	// Test that we can still publish to the topic
	err = bus.Publish("test.mgmt", "test event")
	assert.NoError(t, err)

	err = bus.Publish("test.other", "other event")
	assert.NoError(t, err)

	// Unsubscribe remaining handlers
	err = bus.Unsubscribe("test.mgmt", handler1)
	require.NoError(t, err)

	err = bus.Unsubscribe("test.mgmt", handler3)
	require.NoError(t, err)

	// Publishing should still work even with no subscribers
	err = bus.Publish("test.mgmt", "lonely event")
	assert.NoError(t, err)
}

func TestEventBus_TopicIsolation(t *testing.T) {
	logger := zap.NewNop()
	bus := NewEventBus(logger)
	defer bus.Close()

	topic1Events := make([]interface{}, 0)
	topic2Events := make([]interface{}, 0)
	var mu sync.Mutex

	handler1 := func(event interface{}) {
		mu.Lock()
		topic1Events = append(topic1Events, event)
		mu.Unlock()
	}

	handler2 := func(event interface{}) {
		mu.Lock()
		topic2Events = append(topic2Events, event)
		mu.Unlock()
	}

	err := bus.Subscribe("topic1", handler1)
	require.NoError(t, err)

	err = bus.Subscribe("topic2", handler2)
	require.NoError(t, err)

	// Publish to topic1
	err = bus.Publish("topic1", "event for topic1")
	require.NoError(t, err)

	// Publish to topic2
	err = bus.Publish("topic2", "event for topic2")
	require.NoError(t, err)

	// Allow time for processing
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	topic1Count := len(topic1Events)
	topic2Count := len(topic2Events)
	mu.Unlock()

	// Each topic should have received only its own event
	assert.Equal(t, 1, topic1Count)
	assert.Equal(t, 1, topic2Count)

	mu.Lock()
	if len(topic1Events) > 0 {
		assert.Equal(t, "event for topic1", topic1Events[0])
	}
	if len(topic2Events) > 0 {
		assert.Equal(t, "event for topic2", topic2Events[0])
	}
	mu.Unlock()
}
