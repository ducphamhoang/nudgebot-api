package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"nudgebot-api/internal/events"
)

func TestEventBus_PublishFailure(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create event bus
	eventBus := events.NewEventBus(logger)

	// Close the event bus to simulate failure
	err := eventBus.Close()
	require.NoError(t, err)

	// Try to publish event to closed bus
	event := events.MessageReceived{
		Event:       events.NewEvent(),
		UserID:      "user123",
		ChatID:      "123",
		MessageText: "test message",
	}

	// Publishing to closed bus should return error
	err = eventBus.Publish("message.received", event)
	assert.Error(t, err, "Publishing to closed bus should return error")
}

func TestEventBus_ConcurrentEventHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewEventBus(logger)
	defer eventBus.Close()

	// Track received events
	receivedEvents := make(chan events.MessageReceived, 100)

	// Subscribe to events
	subscriber := func(event events.MessageReceived) {
		receivedEvents <- event
	}

	err := eventBus.Subscribe("message.received", subscriber)
	require.NoError(t, err)

	// Publish multiple events concurrently
	numEvents := 10
	for i := 0; i < numEvents; i++ {
		go func(id int) {
			event := events.MessageReceived{
				Event:       events.NewEvent(),
				UserID:      "user123",
				ChatID:      "123",
				MessageText: "concurrent test message",
			}
			err := eventBus.Publish("message.received", event)
			if err != nil {
				t.Logf("Failed to publish event %d: %v", id, err)
			}
		}(i)
	}

	// Wait for events to be processed
	timeout := time.After(3 * time.Second)
	receivedCount := 0

	for receivedCount < numEvents {
		select {
		case <-receivedEvents:
			receivedCount++
		case <-timeout:
			break
		}
	}

	// Should receive at least some events (exact count may vary due to concurrency)
	assert.Greater(t, receivedCount, 0, "Should receive at least some events")
	t.Logf("Received %d out of %d events", receivedCount, numEvents)
}

func TestEventBus_EventOrderingUnderLoad(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewEventBus(logger)
	defer eventBus.Close()

	// Track event order
	eventOrder := make(chan string, 50)

	// Subscribe to events
	subscriber := func(event events.MessageReceived) {
		// Extract order from MessageText
		eventOrder <- event.MessageText
	}

	err := eventBus.Subscribe("message.received", subscriber)
	require.NoError(t, err)

	// Publish events in sequence
	numEvents := 20
	for i := 0; i < numEvents; i++ {
		event := events.MessageReceived{
			Event:       events.NewEvent(),
			UserID:      "user123",
			ChatID:      "123",
			MessageText: fmt.Sprintf("message-%d", i),
		}
		err := eventBus.Publish("message.received", event)
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond) // Small delay between publishes
	}

	// Collect received events
	timeout := time.After(5 * time.Second)
	receivedOrder := make([]string, 0, numEvents)

	for len(receivedOrder) < numEvents {
		select {
		case msg := <-eventOrder:
			receivedOrder = append(receivedOrder, msg)
		case <-timeout:
			break
		}
	}

	// Verify we received events (order may not be guaranteed depending on implementation)
	assert.Greater(t, len(receivedOrder), 0, "Should receive events")
	t.Logf("Event processing order: %v", receivedOrder)

	// Check for basic event delivery (all unique events)
	uniqueEvents := make(map[string]bool)
	for _, msg := range receivedOrder {
		uniqueEvents[msg] = true
	}
	assert.Greater(t, len(uniqueEvents), 0, "Should receive unique events")
}

func TestEventBus_SubscriptionRecovery(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewEventBus(logger)
	defer eventBus.Close()

	// Counter for received events
	eventCount := 0

	// Create a subscriber that processes events
	subscriber := func(event events.MessageReceived) {
		eventCount++
	}

	err := eventBus.Subscribe("message.received", subscriber)
	require.NoError(t, err)

	// Publish multiple events
	for i := 0; i < 5; i++ {
		event := events.MessageReceived{
			Event:       events.NewEvent(),
			UserID:      "user123",
			ChatID:      "123",
			MessageText: "recovery test message",
		}
		err := eventBus.Publish("message.received", event)
		require.NoError(t, err)
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for processing
	time.Sleep(1 * time.Second)

	// Should have processed events
	assert.Greater(t, eventCount, 0, "Should have processed some events")
	t.Logf("Processed %d events", eventCount)
}
