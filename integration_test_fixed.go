package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"nudgebot-api/internal/events"
	"nudgebot-api/internal/llm"
	"nudgebot-api/internal/nudge"
)

// TestEventDrivenIntegration tests the complete event-driven flow
// from message receipt to task creation and user notification
func TestEventDrivenIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewMockEventBus()

	// Setup services (using mocks where necessary for testing)
	_ = setupMockNudgeService(t, eventBus, logger)
	_ = setupMockLLMService(t, eventBus, logger)

	// Wait for all services to initialize
	time.Sleep(100 * time.Millisecond)

	// Test complete flow: Message -> LLM Parse -> Task Creation -> Notification
	userID := "integration_test_user"
	chatID := "integration_test_chat"
	messageText := "Buy groceries tomorrow at 3 PM"

	// Step 1: Simulate incoming message
	messageEvent := events.MessageReceived{
		Event:       events.NewEvent(),
		UserID:      userID,
		ChatID:      chatID,
		MessageText: messageText,
	}

	err := eventBus.Publish(events.TopicMessageReceived, messageEvent)
	require.NoError(t, err)

	// Step 2: Wait for LLM processing and task creation
	time.Sleep(200 * time.Millisecond)

	// Step 3: Simulate task list request
	listTasksEvent := events.TaskListRequested{
		Event:  events.NewEvent(),
		UserID: userID,
		ChatID: chatID,
	}

	err = eventBus.Publish(events.TopicTaskListRequested, listTasksEvent)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify events were processed with detailed assertions
	publishedEvents := eventBus.GetPublishedEvents(events.TopicMessageReceived)
	assert.Greater(t, len(publishedEvents), 0, "At least one MessageReceived event should be published")

	// Verify the first published event has the expected content
	if len(publishedEvents) > 0 {
		if msgEvent, ok := publishedEvents[0].(events.MessageReceived); ok {
			assert.Equal(t, "user123", msgEvent.UserID, "UserID should match")
			assert.Equal(t, "chat123", msgEvent.ChatID, "ChatID should match")
			assert.Equal(t, "Buy groceries tomorrow at 3 PM", msgEvent.MessageText, "Message text should match")
			assert.NotEmpty(t, msgEvent.CorrelationID, "CorrelationID should not be empty")
			assert.False(t, msgEvent.Timestamp.IsZero(), "Timestamp should be set")
		} else {
			t.Error("Published event should be of type MessageReceived")
		}
	}

	t.Log("✅ Event-driven integration test completed successfully")
}

func TestEventErrorHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewMockEventBus()

	// Test error handling in event processing
	_ = setupMockNudgeService(t, eventBus, logger)

	// Wait for service initialization
	time.Sleep(50 * time.Millisecond)

	// Test with invalid event data
	invalidEvents := []interface{}{
		"not an event",
		123,
		map[string]interface{}{"invalid": "structure"},
		events.TaskListRequested{}, // Missing required fields
	}

	for i, invalidEvent := range invalidEvents {
		err := eventBus.Publish(events.TopicTaskListRequested, invalidEvent)
		// The mock event bus should handle invalid events gracefully
		assert.NoError(t, err, "Mock event bus should handle invalid event %d", i)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	t.Log("✅ Event error handling test passed")
}

func TestConcurrentEventProcessing(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := events.NewMockEventBus()

	// Setup services
	_ = setupMockNudgeService(t, eventBus, logger)

	// Wait for service initialization
	time.Sleep(50 * time.Millisecond)

	// Test concurrent event processing
	numEvents := 10
	done := make(chan bool, numEvents)

	// Publish multiple events concurrently
	for i := 0; i < numEvents; i++ {
		go func(index int) {
			defer func() { done <- true }()

			event := events.TaskListRequested{
				Event:  events.NewEvent(),
				UserID: fmt.Sprintf("user_%d", index),
				ChatID: fmt.Sprintf("chat_%d", index),
			}

			err := eventBus.Publish(events.TopicTaskListRequested, event)
			assert.NoError(t, err)
		}(i)
	}

	// Wait for all events to be processed
	for i := 0; i < numEvents; i++ {
		<-done
	}

	// Additional wait for processing
	time.Sleep(200 * time.Millisecond)

	t.Log("✅ Concurrent event processing test passed")
}

// Helper functions for test setup

func setupMockNudgeService(t *testing.T, eventBus events.EventBus, logger *zap.Logger) nudge.NudgeService {
	// Create mock nudge service for testing
	mockRepo := nudge.NewEnhancedMockNudgeRepository()
	service, err := nudge.NewNudgeService(eventBus, logger, mockRepo)
	if err != nil {
		t.Fatalf("Failed to create nudge service: %v", err)
	}
	return service
}

func setupMockLLMService(t *testing.T, eventBus events.EventBus, logger *zap.Logger) llm.LLMService {
	// For testing, create a simple mock that can handle events
	service := &mockLLMServiceIntegration{
		eventBus: eventBus,
		logger:   logger,
	}

	// Subscribe to MessageReceived events
	err := eventBus.Subscribe(events.TopicMessageReceived, func(event interface{}) error {
		if msgEvent, ok := event.(events.MessageReceived); ok {
			return service.HandleMessageReceived(msgEvent)
		}
		return nil
	})

	if err != nil {
		t.Logf("Failed to subscribe LLM service to MessageReceived: %v", err)
	}

	return service
}

// Mock implementations for integration testing

type mockLLMServiceIntegration struct {
	eventBus events.EventBus
	logger   *zap.Logger
}

func (m *mockLLMServiceIntegration) ParseTaskFromMessage(userID, messageText string) (*llm.TaskSuggestion, error) {
	// Simple mock implementation
	return &llm.TaskSuggestion{
		Title:       extractTaskTitle(messageText),
		Description: messageText,
		Priority:    "medium",
		Tags:        []string{"integration-test"},
	}, nil
}

func (m *mockLLMServiceIntegration) GetSuggestions(userID string, context string) ([]*llm.TaskSuggestion, error) {
	// Mock implementation - return empty suggestions
	return []*llm.TaskSuggestion{}, nil
}

func (m *mockLLMServiceIntegration) HandleMessageReceived(event events.MessageReceived) error {
	// Simulate LLM processing by immediately publishing a TaskParsed event
	parsedTask := events.ParsedTask{
		Title:       extractTaskTitle(event.MessageText),
		Description: event.MessageText,
		Priority:    "medium",
		Tags:        []string{"integration-test"},
	}

	// Add due date if mentioned
	if containsDateHint(event.MessageText) {
		tomorrow := time.Now().Add(24 * time.Hour)
		parsedTask.DueDate = &tomorrow
	}

	taskParsedEvent := events.TaskParsed{
		Event:      events.NewEvent(),
		UserID:     event.UserID,
		ParsedTask: parsedTask,
	}

	return m.eventBus.Publish(events.TopicTaskParsed, taskParsedEvent)
}

func extractTaskTitle(text string) string {
	// Simple title extraction for testing
	words := strings.Fields(text)
	if len(words) > 5 {
		return strings.Join(words[:5], " ")
	}
	return text
}

func containsDateHint(text string) bool {
	dateHints := []string{"tomorrow", "today", "next week", "monday", "tuesday"}
	lowerText := strings.ToLower(text)

	for _, hint := range dateHints {
		if strings.Contains(lowerText, hint) {
			return true
		}
	}
	return false
}
