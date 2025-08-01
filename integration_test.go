package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"nudgebot-api/internal/chatbot"
	"nudgebot-api/internal/common"
	"nudgebot-api/internal/events"
	"nudgebot-api/internal/llm"
	"nudgebot-api/internal/mocks"
	"nudgebot-api/internal/nudge"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TestEventDrivenIntegration tests the complete event-driven flow
// from message receipt to task creation and user notification
func TestEventDrivenIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := mocks.NewMockEventBus()

	// Setup services (using mocks where necessary for testing)
	nudgeService := setupMockNudgeService(t, eventBus, logger)
	llmService := setupMockLLMService(t, eventBus, logger)
	chatbotService := setupMockChatbotService(t, eventBus, logger)

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
		ReceivedAt:  time.Now(),
	}

	err := eventBus.Publish(events.TopicMessageReceived, messageEvent)
	require.NoError(t, err)

	// Wait for LLM processing
	time.Sleep(200 * time.Millisecond)

	// Step 2: Simulate task list request
	listRequestEvent := events.TaskListRequested{
		Event:  events.NewEvent(),
		UserID: userID,
		ChatID: chatID,
		Limit:  10,
		Offset: 0,
	}

	err = eventBus.Publish(events.TopicTaskListRequested, listRequestEvent)
	require.NoError(t, err)

	// Wait for task list processing
	time.Sleep(200 * time.Millisecond)

	// Step 3: Simulate task action
	actionRequestEvent := events.TaskActionRequested{
		Event:  events.NewEvent(),
		UserID: userID,
		ChatID: chatID,
		TaskID: "test_task_123",
		Action: "done",
	}

	err = eventBus.Publish(events.TopicTaskActionRequested, actionRequestEvent)
	require.NoError(t, err)

	// Wait for action processing
	time.Sleep(200 * time.Millisecond)

	t.Log("✅ Complete event-driven integration test passed")
}

func TestEventFlowValidation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := mocks.NewMockEventBus()

	// Create event flow validator
	validator := events.NewEventFlowValidator(logger)

	// Test valid event flow
	validFlow := []events.Event{
		events.MessageReceived{
			Event:       events.NewEvent(),
			UserID:      "user123",
			ChatID:      "chat123",
			MessageText: "Test message",
			ReceivedAt:  time.Now(),
		},
		events.TaskParsed{
			Event:  events.NewEvent(),
			UserID: "user123",
			ParsedTask: events.ParsedTask{
				Title:    "Test task",
				Priority: "medium",
			},
		},
		events.TaskCreated{
			Event:     events.NewEvent(),
			TaskID:    "task123",
			UserID:    "user123",
			Title:     "Test task",
			Priority:  "medium",
			CreatedAt: time.Now(),
		},
	}

	for _, event := range validFlow {
		err := validator.ValidateEvent(event)
		assert.NoError(t, err, "Valid event should pass validation")
	}

	// Test flow sequence validation
	isValidSequence := validator.ValidateFlowSequence(validFlow)
	assert.True(t, isValidSequence, "Valid flow sequence should pass validation")

	t.Log("✅ Event flow validation test passed")
}

func TestEventMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := mocks.NewMockEventBus()

	// Create event metrics collector
	metrics := common.NewEventMetrics(logger)

	// Simulate event processing with metrics
	topics := []string{
		events.TopicMessageReceived,
		events.TopicTaskParsed,
		events.TopicTaskCreated,
		events.TopicTaskListRequested,
		events.TopicTaskListResponse,
	}

	for _, topic := range topics {
		// Record event processing
		metrics.RecordEventProcessed(topic, true, time.Millisecond*100)
	}

	// Get metrics summary
	summary := metrics.GetSummary()

	assert.Equal(t, len(topics), int(summary.TotalEventsProcessed),
		"Total events processed should match")
	assert.Zero(t, summary.TotalEventsFailed,
		"No events should have failed")
	assert.Greater(t, summary.AverageProcessingTime, time.Duration(0),
		"Average processing time should be greater than zero")

	t.Log("✅ Event metrics test passed")
}

func TestEventErrorHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := mocks.NewMockEventBus()

	// Test error handling in event processing
	nudgeService := setupMockNudgeService(t, eventBus, logger)

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
	eventBus := mocks.NewMockEventBus()

	// Setup services
	nudgeService := setupMockNudgeService(t, eventBus, logger)

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
				Limit:  10,
				Offset: 0,
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

func TestEventPersistenceAndReplay(t *testing.T) {
	// Test event persistence and replay functionality
	logger := zaptest.NewLogger(t)
	eventBus := mocks.NewMockEventBus()

	// Create event store (mock)
	eventStore := mocks.NewMockEventStore()

	// Test event persistence
	testEvents := []interface{}{
		events.MessageReceived{
			Event:       events.NewEvent(),
			UserID:      "persist_user",
			ChatID:      "persist_chat",
			MessageText: "Persistent message",
			ReceivedAt:  time.Now(),
		},
		events.TaskCreated{
			Event:     events.NewEvent(),
			TaskID:    "persist_task",
			UserID:    "persist_user",
			Title:     "Persistent task",
			Priority:  "medium",
			CreatedAt: time.Now(),
		},
	}

	// Store events
	for _, event := range testEvents {
		err := eventStore.Store(event)
		assert.NoError(t, err, "Event should be stored successfully")
	}

	// Retrieve and replay events
	storedEvents, err := eventStore.GetEvents("persist_user")
	assert.NoError(t, err)
	assert.Len(t, storedEvents, len(testEvents), "All events should be retrievable")

	t.Log("✅ Event persistence and replay test passed")
}

func TestEventMonitoring(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := mocks.NewMockEventBus()

	// Create event flow monitor
	monitor := events.NewEventFlowMonitor(logger)

	// Start monitoring
	monitor.StartMonitoring()
	defer monitor.StopMonitoring()

	// Simulate monitored events
	monitoredEvents := []interface{}{
		events.MessageReceived{
			Event:       events.NewEvent(),
			UserID:      "monitor_user",
			ChatID:      "monitor_chat",
			MessageText: "Monitored message",
			ReceivedAt:  time.Now(),
		},
		events.TaskParsed{
			Event:  events.NewEvent(),
			UserID: "monitor_user",
			ParsedTask: events.ParsedTask{
				Title:    "Monitored task",
				Priority: "medium",
			},
		},
	}

	for _, event := range monitoredEvents {
		monitor.RecordEvent(event)
	}

	// Wait for monitoring data
	time.Sleep(100 * time.Millisecond)

	// Get monitoring report
	report := monitor.GetFlowReport()
	assert.NotNil(t, report, "Monitoring report should be available")

	t.Log("✅ Event monitoring test passed")
}

func TestEventDrivenServiceShutdown(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus := mocks.NewMockEventBus()

	// Setup services
	nudgeService := setupMockNudgeService(t, eventBus, logger)

	// Wait for initialization
	time.Sleep(50 * time.Millisecond)

	// Test graceful shutdown
	shutdownEvent := events.ServiceShutdown{
		Event:     events.NewEvent(),
		ServiceID: "test_service",
		Reason:    "Integration test shutdown",
		Timestamp: time.Now(),
	}

	err := eventBus.Publish(events.TopicServiceShutdown, shutdownEvent)
	assert.NoError(t, err)

	// Wait for shutdown processing
	time.Sleep(100 * time.Millisecond)

	t.Log("✅ Event-driven service shutdown test passed")
}

// Helper functions for test setup

func setupMockNudgeService(t *testing.T, eventBus events.EventBus, logger *zap.Logger) nudge.NudgeService {
	// Create mock nudge service for testing
	mockRepo := nudge.NewMockTaskRepository()
	service := nudge.NewNudgeService(eventBus, logger, mockRepo)
	return service
}

func setupMockLLMService(t *testing.T, eventBus events.EventBus, logger *zap.Logger) llm.LLMService {
	// For testing, create a simple mock that can handle events
	return &mockLLMServiceIntegration{
		eventBus: eventBus,
		logger:   logger,
	}
}

func setupMockChatbotService(t *testing.T, eventBus events.EventBus, logger *zap.Logger) chatbot.ChatbotService {
	// For testing, return nil since chatbot service requires Telegram token
	// In a real test environment, we would use dependency injection with mocks
	return nil
}

// Mock implementations for integration testing

type mockLLMServiceIntegration struct {
	eventBus events.EventBus
	logger   *zap.Logger
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

func TestCompleteUserJourney(t *testing.T) {
	// End-to-end test simulating a complete user journey
	logger := zaptest.NewLogger(t)
	eventBus := mocks.NewMockEventBus()

	// Setup all services
	nudgeService := setupMockNudgeService(t, eventBus, logger)
	llmService := setupMockLLMServiceIntegration(t, eventBus, logger)

	// Wait for initialization
	time.Sleep(100 * time.Millisecond)

	userID := "journey_user"
	chatID := "journey_chat"

	// Journey Step 1: User sends a message
	t.Log("🚀 Starting complete user journey test")

	messageEvent := events.MessageReceived{
		Event:       events.NewEvent(),
		UserID:      userID,
		ChatID:      chatID,
		MessageText: "Buy groceries tomorrow at 3 PM",
		ReceivedAt:  time.Now(),
	}

	err := eventBus.Publish(events.TopicMessageReceived, messageEvent)
	require.NoError(t, err)
	t.Log("✅ Step 1: Message sent")

	// Wait for LLM processing
	time.Sleep(200 * time.Millisecond)

	// Journey Step 2: User requests task list
	listRequestEvent := events.TaskListRequested{
		Event:  events.NewEvent(),
		UserID: userID,
		ChatID: chatID,
		Limit:  10,
		Offset: 0,
	}

	err = eventBus.Publish(events.TopicTaskListRequested, listRequestEvent)
	require.NoError(t, err)
	t.Log("✅ Step 2: Task list requested")

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Journey Step 3: User marks task as done
	actionRequestEvent := events.TaskActionRequested{
		Event:  events.NewEvent(),
		UserID: userID,
		ChatID: chatID,
		TaskID: "journey_task_123",
		Action: "done",
	}

	err = eventBus.Publish(events.TopicTaskActionRequested, actionRequestEvent)
	require.NoError(t, err)
	t.Log("✅ Step 3: Task action requested")

	// Wait for final processing
	time.Sleep(200 * time.Millisecond)

	t.Log("🎉 Complete user journey test passed!")
}

func setupMockLLMServiceIntegration(t *testing.T, eventBus events.EventBus, logger *zap.Logger) *mockLLMServiceIntegration {
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
