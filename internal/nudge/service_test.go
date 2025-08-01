package nudge

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"nudgebot-api/internal/common"
	"nudgebot-api/internal/events"
	"nudgebot-api/internal/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// Simple mock event bus for testing
type mockEventBus struct {
	subscribers     map[string][]func(interface{}) error
	publishedEvents map[string][]interface{}
	mu              sync.RWMutex
}

func newMockEventBus() *mockEventBus {
	return &mockEventBus{
		subscribers:     make(map[string][]func(interface{}) error),
		publishedEvents: make(map[string][]interface{}),
	}
}

func (m *mockEventBus) Publish(topic string, data interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store published event
	m.publishedEvents[topic] = append(m.publishedEvents[topic], data)

	// Call subscribers
	if handlers, exists := m.subscribers[topic]; exists {
		for _, handler := range handlers {
			go handler(data) // Run async like real event bus
		}
	}

	return nil
}

func (m *mockEventBus) Subscribe(topic string, handler interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Convert handler to our function type
	handlerFunc, ok := handler.(func(interface{}) error)
	if !ok {
		return fmt.Errorf("invalid handler type")
	}

	m.subscribers[topic] = append(m.subscribers[topic], handlerFunc)
	return nil
}

func (m *mockEventBus) Unsubscribe(topic string, handler interface{}) error {
	// For testing, we don't need to implement this
	return nil
}

func (m *mockEventBus) Close() error {
	return nil
}

func (m *mockEventBus) GetPublishedEvents(topic string) []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.publishedEvents[topic]
}

func (m *mockEventBus) GetSubscriberCount(topic string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.subscribers[topic])
}

func TestNudgeService_HandleTaskListRequested(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		chatID        string
		existingTasks []*Task
		expectedTasks int
		expectedError bool
	}{
		{
			name:          "successful task list retrieval with tasks",
			userID:        "user123",
			chatID:        "chat123",
			existingTasks: createTestTasks("user123", 3),
			expectedTasks: 3,
			expectedError: false,
		},
		{
			name:          "successful task list retrieval with no tasks",
			userID:        "user456",
			chatID:        "chat456",
			existingTasks: []*Task{},
			expectedTasks: 0,
			expectedError: false,
		},
		{
			name:          "task list for different user",
			userID:        "user789",
			chatID:        "chat789",
			existingTasks: createTestTasks("other_user", 2),
			expectedTasks: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			logger := zaptest.NewLogger(t)
			mockEventBus := newMockEventBus()
			mockRepository := NewMockTaskRepository()

			// Add existing tasks to mock repository
			for _, task := range tt.existingTasks {
				err := mockRepository.CreateTask(task)
				require.NoError(t, err)
			}

			service := NewNudgeService(mockEventBus, logger, mockRepository)

			// Wait for service to initialize subscriptions
			time.Sleep(50 * time.Millisecond)

			// Create and publish TaskListRequested event
			event := events.TaskListRequested{
				Event:  events.NewEvent(),
				UserID: tt.userID,
				ChatID: tt.chatID,
			}

			err := mockEventBus.Publish(events.TopicTaskListRequested, event)
			require.NoError(t, err)

			// Wait for event processing
			time.Sleep(100 * time.Millisecond)

			// Verify TaskListResponse was published
			publishedEvents := mockEventBus.GetPublishedEvents(events.TopicTaskListResponse)
			assert.Len(t, publishedEvents, 1, "Expected one TaskListResponse event")

			if len(publishedEvents) > 0 {
				responseEvent, ok := publishedEvents[0].(events.TaskListResponse)
				require.True(t, ok, "Published event should be TaskListResponse")

				assert.Equal(t, tt.userID, responseEvent.UserID)
				assert.Equal(t, tt.chatID, responseEvent.ChatID)
				assert.Len(t, responseEvent.Tasks, tt.expectedTasks)
				assert.Equal(t, tt.expectedTasks, responseEvent.TotalCount)
			}
		})
	}
}

func TestNudgeService_HandleTaskActionRequested(t *testing.T) {
	tests := []struct {
		name            string
		action          string
		taskExists      bool
		expectedStatus  common.TaskStatus
		expectedSuccess bool
	}{
		{
			name:            "complete task successfully",
			action:          "done",
			taskExists:      true,
			expectedStatus:  common.TaskStatusCompleted,
			expectedSuccess: true,
		},
		{
			name:            "complete task with alternative action",
			action:          "complete",
			taskExists:      true,
			expectedStatus:  common.TaskStatusCompleted,
			expectedSuccess: true,
		},
		{
			name:            "delete task successfully",
			action:          "delete",
			taskExists:      true,
			expectedStatus:  common.TaskStatusActive, // Status before deletion
			expectedSuccess: true,
		},
		{
			name:            "snooze task successfully",
			action:          "snooze",
			taskExists:      true,
			expectedStatus:  common.TaskStatusSnoozed,
			expectedSuccess: true,
		},
		{
			name:            "invalid action",
			action:          "invalid_action",
			taskExists:      true,
			expectedStatus:  common.TaskStatusActive,
			expectedSuccess: false,
		},
		{
			name:            "action on non-existent task",
			action:          "done",
			taskExists:      false,
			expectedStatus:  common.TaskStatusActive,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			logger := zaptest.NewLogger(t)
			mockEventBus := mocks.NewMockEventBus()
			mockRepository := NewMockTaskRepository()

			taskID := "test_task_123"
			userID := "user123"
			chatID := "chat123"

			// Create task if it should exist
			if tt.taskExists {
				task := &Task{
					ID:     common.TaskID(taskID),
					UserID: common.UserID(userID),
					Title:  "Test Task",
					Status: common.TaskStatusActive,
				}
				err := mockRepository.CreateTask(task)
				require.NoError(t, err)
			}

			service := NewNudgeService(mockEventBus, logger, mockRepository)

			// Wait for service to initialize subscriptions
			time.Sleep(50 * time.Millisecond)

			// Create and publish TaskActionRequested event
			event := events.TaskActionRequested{
				Event:  events.NewEvent(),
				UserID: userID,
				ChatID: chatID,
				TaskID: taskID,
				Action: tt.action,
			}

			err := mockEventBus.Publish(events.TopicTaskActionRequested, event)
			require.NoError(t, err)

			// Wait for event processing
			time.Sleep(100 * time.Millisecond)

			// Verify TaskActionResponse was published
			publishedEvents := mockEventBus.GetPublishedEvents(events.TopicTaskActionResponse)
			assert.Len(t, publishedEvents, 1, "Expected one TaskActionResponse event")

			if len(publishedEvents) > 0 {
				responseEvent, ok := publishedEvents[0].(events.TaskActionResponse)
				require.True(t, ok, "Published event should be TaskActionResponse")

				assert.Equal(t, userID, responseEvent.UserID)
				assert.Equal(t, chatID, responseEvent.ChatID)
				assert.Equal(t, taskID, responseEvent.TaskID)
				assert.Equal(t, tt.action, responseEvent.Action)
				assert.Equal(t, tt.expectedSuccess, responseEvent.Success)

				if tt.expectedSuccess {
					assert.NotEmpty(t, responseEvent.Message)
				}
			}

			// For successful actions, verify additional events
			if tt.expectedSuccess && tt.taskExists {
				switch tt.action {
				case "done", "complete":
					// Should publish TaskCompleted event
					completedEvents := mockEventBus.GetPublishedEvents(events.TopicTaskCompleted)
					assert.Len(t, completedEvents, 1, "Expected TaskCompleted event for done action")

				case "delete":
					// Task should be deleted from repository
					_, err := mockRepository.GetTaskByID(common.TaskID(taskID))
					assert.Error(t, err, "Task should be deleted from repository")
				}
			}
		})
	}
}

func TestNudgeService_HandleTaskParsed(t *testing.T) {
	// Setup
	logger := zaptest.NewLogger(t)
	mockEventBus := mocks.NewMockEventBus()
	mockRepository := NewMockTaskRepository()

	service := NewNudgeService(mockEventBus, logger, mockRepository)

	// Wait for service to initialize subscriptions
	time.Sleep(50 * time.Millisecond)

	// Create and publish TaskParsed event
	dueDate := time.Now().Add(24 * time.Hour)
	event := events.TaskParsed{
		Event:  events.NewEvent(),
		UserID: "user123",
		ParsedTask: events.ParsedTask{
			Title:       "Buy groceries",
			Description: "Get milk, bread, and eggs",
			DueDate:     &dueDate,
			Priority:    "high",
			Tags:        []string{"shopping", "food"},
		},
	}

	err := mockEventBus.Publish(events.TopicTaskParsed, event)
	require.NoError(t, err)

	// Wait for event processing
	time.Sleep(100 * time.Millisecond)

	// Verify TaskCreated event was published
	publishedEvents := mockEventBus.GetPublishedEvents(events.TopicTaskCreated)
	assert.Len(t, publishedEvents, 1, "Expected one TaskCreated event")

	if len(publishedEvents) > 0 {
		createdEvent, ok := publishedEvents[0].(events.TaskCreated)
		require.True(t, ok, "Published event should be TaskCreated")

		assert.Equal(t, event.UserID, createdEvent.UserID)
		assert.Equal(t, event.ParsedTask.Title, createdEvent.Title)
		assert.Equal(t, event.ParsedTask.Priority, createdEvent.Priority)
		assert.NotEmpty(t, createdEvent.TaskID)
	}

	// Verify task was created in repository
	assert.Equal(t, 1, mockRepository.GetTaskCount())
}

func TestNudgeService_EventSubscriptions(t *testing.T) {
	// Setup
	logger := zaptest.NewLogger(t)
	mockEventBus := mocks.NewMockEventBus()
	mockRepository := NewMockTaskRepository()

	// Create service (this should set up subscriptions)
	_ = NewNudgeService(mockEventBus, logger, mockRepository)

	// Wait for service to initialize subscriptions
	time.Sleep(50 * time.Millisecond)

	// Verify subscriptions were set up
	expectedSubscriptions := []string{
		events.TopicTaskParsed,
		events.TopicTaskListRequested,
		events.TopicTaskActionRequested,
	}

	for _, topic := range expectedSubscriptions {
		subscriberCount := mockEventBus.GetSubscriberCount(topic)
		assert.Greater(t, subscriberCount, 0, "Expected at least one subscriber for topic: %s", topic)
	}
}

// Helper functions

func createTestTasks(userID string, count int) []*Task {
	tasks := make([]*Task, count)
	for i := 0; i < count; i++ {
		tasks[i] = &Task{
			ID:          common.TaskID(common.NewID()),
			UserID:      common.UserID(userID),
			Title:       fmt.Sprintf("Test Task %d", i+1),
			Description: fmt.Sprintf("Description for task %d", i+1),
			Priority:    common.PriorityMedium,
			Status:      common.TaskStatusActive,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}
	return tasks
}

func TestNudgeService_Integration_CompleteEventFlow(t *testing.T) {
	// This test validates the complete event flow from message to task creation
	logger := zaptest.NewLogger(t)
	mockEventBus := mocks.NewMockEventBus()
	mockRepository := NewMockTaskRepository()

	service := NewNudgeService(mockEventBus, logger, mockRepository)

	// Wait for service to initialize
	time.Sleep(50 * time.Millisecond)

	userID := "integration_user"
	chatID := "integration_chat"

	// Step 1: Simulate TaskParsed event (from LLM service)
	dueDate := time.Now().Add(24 * time.Hour)
	taskParsedEvent := events.TaskParsed{
		Event:  events.NewEvent(),
		UserID: userID,
		ParsedTask: events.ParsedTask{
			Title:       "Integration Test Task",
			Description: "A task created during integration testing",
			DueDate:     &dueDate,
			Priority:    "high",
			Tags:        []string{"test", "integration"},
		},
	}

	err := mockEventBus.Publish(events.TopicTaskParsed, taskParsedEvent)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify task was created and TaskCreated event published
	createdEvents := mockEventBus.GetPublishedEvents(events.TopicTaskCreated)
	require.Len(t, createdEvents, 1)

	createdEvent := createdEvents[0].(events.TaskCreated)
	taskID := createdEvent.TaskID

	// Step 2: Simulate task list request
	listRequestEvent := events.TaskListRequested{
		Event:  events.NewEvent(),
		UserID: userID,
		ChatID: chatID,
	}

	err = mockEventBus.Publish(events.TopicTaskListRequested, listRequestEvent)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify task list response
	listResponseEvents := mockEventBus.GetPublishedEvents(events.TopicTaskListResponse)
	require.Len(t, listResponseEvents, 1)

	listResponse := listResponseEvents[0].(events.TaskListResponse)
	assert.Equal(t, userID, listResponse.UserID)
	assert.Equal(t, chatID, listResponse.ChatID)
	assert.Len(t, listResponse.Tasks, 1)
	assert.Equal(t, 1, listResponse.TotalCount)

	// Step 3: Simulate task completion
	actionRequestEvent := events.TaskActionRequested{
		Event:  events.NewEvent(),
		UserID: userID,
		ChatID: chatID,
		TaskID: taskID,
		Action: "done",
	}

	err = mockEventBus.Publish(events.TopicTaskActionRequested, actionRequestEvent)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify action response and completion events
	actionResponseEvents := mockEventBus.GetPublishedEvents(events.TopicTaskActionResponse)
	require.Len(t, actionResponseEvents, 1)

	actionResponse := actionResponseEvents[0].(events.TaskActionResponse)
	assert.Equal(t, userID, actionResponse.UserID)
	assert.Equal(t, taskID, actionResponse.TaskID)
	assert.Equal(t, "done", actionResponse.Action)
	assert.True(t, actionResponse.Success)

	completedEvents := mockEventBus.GetPublishedEvents(events.TopicTaskCompleted)
	require.Len(t, completedEvents, 1)

	t.Log("✅ Complete event flow integration test passed")
}
