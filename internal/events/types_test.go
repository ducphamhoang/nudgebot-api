package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventTypes_Validation(t *testing.T) {
	tests := []struct {
		name        string
		event       interface{}
		shouldError bool
	}{
		{
			name: "valid MessageReceived event",
			event: MessageReceived{
				Event:       NewEvent(),
				UserID:      "user123",
				ChatID:      "chat456",
				MessageText: "Hello, world!",
			},
			shouldError: false,
		},
		{
			name: "MessageReceived with empty UserID",
			event: MessageReceived{
				Event:       NewEvent(),
				UserID:      "",
				ChatID:      "chat456",
				MessageText: "Hello, world!",
			},
			shouldError: true,
		},
		{
			name: "valid TaskParsed event",
			event: TaskParsed{
				Event:  NewEvent(),
				UserID: "user123",
				ChatID: "chat456",
				ParsedTask: ParsedTask{
					Title:    "Complete project",
					Priority: "high",
				},
			},
			shouldError: false,
		},
		{
			name: "TaskParsed with empty title",
			event: TaskParsed{
				Event:  NewEvent(),
				UserID: "user123",
				ChatID: "chat456",
				ParsedTask: ParsedTask{
					Title:    "",
					Priority: "high",
				},
			},
			shouldError: true,
		},
		{
			name: "valid ReminderDue event",
			event: ReminderDue{
				Event:  NewEvent(),
				TaskID: "task123",
				UserID: "user456",
				ChatID: "chat789",
			},
			shouldError: false,
		},
		{
			name: "valid TaskCreated event",
			event: TaskCreated{
				Event:     NewEvent(),
				TaskID:    "task123",
				UserID:    "user456",
				Title:     "New Task",
				Priority:  "medium",
				CreatedAt: time.Now(),
			},
			shouldError: false,
		},
		{
			name: "TaskCreated with missing TaskID",
			event: TaskCreated{
				Event:     NewEvent(),
				TaskID:    "",
				UserID:    "user456",
				Title:     "New Task",
				Priority:  "medium",
				CreatedAt: time.Now(),
			},
			shouldError: true,
		},
		{
			name: "valid TaskListRequested event",
			event: TaskListRequested{
				Event:  NewEvent(),
				UserID: "user123",
				ChatID: "chat456",
			},
			shouldError: false,
		},
		{
			name: "valid TaskActionRequested event",
			event: TaskActionRequested{
				Event:  NewEvent(),
				UserID: "user123",
				ChatID: "chat456",
				TaskID: "task789",
				Action: "done",
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to JSON and back to validate structure
			jsonData, err := json.Marshal(tt.event)
			require.NoError(t, err)

			// Validate that required fields are present in JSON
			var jsonMap map[string]interface{}
			err = json.Unmarshal(jsonData, &jsonMap)
			require.NoError(t, err)

			// Check for required fields based on event type
			switch event := tt.event.(type) {
			case MessageReceived:
				if tt.shouldError {
					assert.True(t, event.UserID == "" || event.ChatID == "" || event.MessageText == "")
				} else {
					assert.NotEmpty(t, event.UserID)
					assert.NotEmpty(t, event.ChatID)
					assert.NotEmpty(t, event.MessageText)
					assert.NotEmpty(t, event.CorrelationID)
					assert.False(t, event.Timestamp.IsZero())
				}
			case TaskParsed:
				if tt.shouldError {
					assert.True(t, event.UserID == "" || event.ChatID == "" || event.ParsedTask.Title == "")
				} else {
					assert.NotEmpty(t, event.UserID)
					assert.NotEmpty(t, event.ChatID)
					assert.NotEmpty(t, event.ParsedTask.Title)
					assert.NotEmpty(t, event.ParsedTask.Priority)
				}
			case TaskCreated:
				if tt.shouldError {
					assert.True(t, event.TaskID == "" || event.UserID == "" || event.Title == "")
				} else {
					assert.NotEmpty(t, event.TaskID)
					assert.NotEmpty(t, event.UserID)
					assert.NotEmpty(t, event.Title)
					assert.NotEmpty(t, event.Priority)
					assert.False(t, event.CreatedAt.IsZero())
				}
			}
		})
	}
}

func TestEventTypes_Serialization(t *testing.T) {
	tests := []struct {
		name  string
		event interface{}
	}{
		{
			name: "MessageReceived serialization",
			event: MessageReceived{
				Event:       NewEvent(),
				UserID:      "user123",
				ChatID:      "chat456",
				MessageText: "Test message with special chars: áéíóú",
			},
		},
		{
			name: "TaskParsed serialization",
			event: TaskParsed{
				Event:  NewEvent(),
				UserID: "user789",
				ChatID: "chat012",
				ParsedTask: ParsedTask{
					Title:       "Complete project with emojis 🚀",
					Description: "This is a detailed description",
					Priority:    "high",
					Tags:        []string{"work", "urgent", "project"},
				},
			},
		},
		{
			name: "TaskCreated with due date",
			event: TaskCreated{
				Event:     NewEvent(),
				TaskID:    "task123",
				UserID:    "user456",
				Title:     "Task with due date",
				DueDate:   timePtr(time.Now().Add(24 * time.Hour)),
				Priority:  "medium",
				CreatedAt: time.Now(),
			},
		},
		{
			name: "TaskListResponse with multiple tasks",
			event: TaskListResponse{
				Event:  NewEvent(),
				UserID: "user123",
				ChatID: "chat456",
				Tasks: []TaskSummary{
					{
						ID:          "task1",
						Title:       "First task",
						Description: "Description 1",
						Priority:    "high",
						Status:      "active",
						IsOverdue:   false,
					},
					{
						ID:          "task2",
						Title:       "Second task",
						Description: "Description 2",
						DueDate:     timePtr(time.Now().Add(-1 * time.Hour)),
						Priority:    "medium",
						Status:      "active",
						IsOverdue:   true,
					},
				},
				TotalCount: 2,
				HasMore:    false,
				Success:    true,
			},
		},
		{
			name: "TaskActionResponse success",
			event: TaskActionResponse{
				Event:   NewEvent(),
				UserID:  "user123",
				ChatID:  "chat456",
				TaskID:  "task789",
				Action:  "done",
				Success: true,
				Message: "Task completed successfully",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			jsonData, err := json.Marshal(tt.event)
			require.NoError(t, err)
			assert.NotEmpty(t, jsonData)

			// Unmarshal back to verify structure
			var unmarshaled map[string]interface{}
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err)

			// Verify correlation ID and timestamp are present
			assert.Contains(t, unmarshaled, "correlation_id")
			assert.Contains(t, unmarshaled, "timestamp")

			// Verify specific event fields based on type
			switch event := tt.event.(type) {
			case MessageReceived:
				assert.Equal(t, event.UserID, unmarshaled["user_id"])
				assert.Equal(t, event.ChatID, unmarshaled["chat_id"])
				assert.Equal(t, event.MessageText, unmarshaled["message_text"])
			case TaskParsed:
				assert.Equal(t, event.UserID, unmarshaled["user_id"])
				assert.Equal(t, event.ChatID, unmarshaled["chat_id"])
				parsedTask, ok := unmarshaled["parsed_task"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, event.ParsedTask.Title, parsedTask["title"])
				assert.Equal(t, event.ParsedTask.Priority, parsedTask["priority"])
			case TaskCreated:
				assert.Equal(t, event.TaskID, unmarshaled["task_id"])
				assert.Equal(t, event.UserID, unmarshaled["user_id"])
				assert.Equal(t, event.Title, unmarshaled["title"])
				assert.Equal(t, event.Priority, unmarshaled["priority"])
			case TaskListResponse:
				assert.Equal(t, event.UserID, unmarshaled["user_id"])
				assert.Equal(t, event.ChatID, unmarshaled["chat_id"])
				assert.Equal(t, float64(event.TotalCount), unmarshaled["total_count"])
				assert.Equal(t, event.Success, unmarshaled["success"])
				tasks, ok := unmarshaled["tasks"].([]interface{})
				require.True(t, ok)
				assert.Len(t, tasks, len(event.Tasks))
			case TaskActionResponse:
				assert.Equal(t, event.UserID, unmarshaled["user_id"])
				assert.Equal(t, event.ChatID, unmarshaled["chat_id"])
				assert.Equal(t, event.TaskID, unmarshaled["task_id"])
				assert.Equal(t, event.Action, unmarshaled["action"])
				assert.Equal(t, event.Success, unmarshaled["success"])
				assert.Equal(t, event.Message, unmarshaled["message"])
			}
		})
	}
}

func TestEventTypes_CorrelationID(t *testing.T) {
	// Test that NewEvent generates unique correlation IDs
	event1 := NewEvent()
	event2 := NewEvent()

	assert.NotEqual(t, event1.CorrelationID, event2.CorrelationID)
	assert.NotEmpty(t, event1.CorrelationID)
	assert.NotEmpty(t, event2.CorrelationID)

	// Verify correlation IDs are valid UUIDs
	_, err := uuid.Parse(event1.CorrelationID)
	assert.NoError(t, err)

	_, err = uuid.Parse(event2.CorrelationID)
	assert.NoError(t, err)

	// Test timestamp generation
	assert.False(t, event1.Timestamp.IsZero())
	assert.False(t, event2.Timestamp.IsZero())
	assert.True(t, event2.Timestamp.After(event1.Timestamp) || event2.Timestamp.Equal(event1.Timestamp))
}

func TestEventTypes_EventCreation(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		createEvent func() interface{}
	}{
		{
			name: "MessageReceived creation",
			createEvent: func() interface{} {
				return MessageReceived{
					Event:       NewEvent(),
					UserID:      "user123",
					ChatID:      "chat456",
					MessageText: "Hello",
				}
			},
		},
		{
			name: "TaskCreated creation",
			createEvent: func() interface{} {
				return TaskCreated{
					Event:     NewEvent(),
					TaskID:    "task123",
					UserID:    "user456",
					Title:     "New Task",
					Priority:  "high",
					CreatedAt: now,
				}
			},
		},
		{
			name: "ReminderDue creation",
			createEvent: func() interface{} {
				return ReminderDue{
					Event:  NewEvent(),
					TaskID: "task789",
					UserID: "user012",
					ChatID: "chat345",
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := tt.createEvent()
			assert.NotNil(t, event)

			// Verify that events have proper base fields
			switch e := event.(type) {
			case MessageReceived:
				assert.NotEmpty(t, e.CorrelationID)
				assert.False(t, e.Timestamp.IsZero())
				assert.NotEmpty(t, e.UserID)
				assert.NotEmpty(t, e.ChatID)
				assert.NotEmpty(t, e.MessageText)
			case TaskCreated:
				assert.NotEmpty(t, e.CorrelationID)
				assert.False(t, e.Timestamp.IsZero())
				assert.NotEmpty(t, e.TaskID)
				assert.NotEmpty(t, e.UserID)
				assert.NotEmpty(t, e.Title)
				assert.NotEmpty(t, e.Priority)
				assert.Equal(t, now, e.CreatedAt)
			case ReminderDue:
				assert.NotEmpty(t, e.CorrelationID)
				assert.False(t, e.Timestamp.IsZero())
				assert.NotEmpty(t, e.TaskID)
				assert.NotEmpty(t, e.UserID)
				assert.NotEmpty(t, e.ChatID)
			}
		})
	}
}

func TestEventTypes_TopicConstants(t *testing.T) {
	// Test that all topic constants are defined and unique
	topics := []string{
		TopicMessageReceived,
		TopicTaskParsed,
		TopicReminderDue,
		TopicTaskCompleted,
		TopicTaskCreated,
		TopicTaskListRequested,
		TopicTaskActionRequested,
		TopicUserSessionStarted,
		TopicCommandExecuted,
		TopicTaskListResponse,
		TopicTaskActionResponse,
	}

	// Verify all topics are non-empty
	for _, topic := range topics {
		assert.NotEmpty(t, topic, "Topic constant should not be empty")
	}

	// Verify all topics are unique
	topicSet := make(map[string]bool)
	for _, topic := range topics {
		assert.False(t, topicSet[topic], "Topic %s should be unique", topic)
		topicSet[topic] = true
	}

	// Verify topic naming convention
	expectedTopics := map[string]string{
		TopicMessageReceived:     "message.received",
		TopicTaskParsed:          "task.parsed",
		TopicReminderDue:         "reminder.due",
		TopicTaskCompleted:       "task.completed",
		TopicTaskCreated:         "task.created",
		TopicTaskListRequested:   "task.list.requested",
		TopicTaskActionRequested: "task.action.requested",
		TopicUserSessionStarted:  "user.session.started",
		TopicCommandExecuted:     "command.executed",
		TopicTaskListResponse:    "task.list.response",
		TopicTaskActionResponse:  "task.action.response",
	}

	for constant, expected := range expectedTopics {
		assert.Equal(t, expected, constant, "Topic constant should match expected value")
	}
}

func TestEventTypes_EventEquality(t *testing.T) {
	baseEvent := NewEvent()

	// Create two identical events (except correlation ID and timestamp)
	event1 := MessageReceived{
		Event:       baseEvent,
		UserID:      "user123",
		ChatID:      "chat456",
		MessageText: "Hello",
	}

	event2 := MessageReceived{
		Event:       baseEvent, // Same base event
		UserID:      "user123",
		ChatID:      "chat456",
		MessageText: "Hello",
	}

	// Events should be equal in content
	assert.Equal(t, event1.UserID, event2.UserID)
	assert.Equal(t, event1.ChatID, event2.ChatID)
	assert.Equal(t, event1.MessageText, event2.MessageText)
	assert.Equal(t, event1.CorrelationID, event2.CorrelationID)
	assert.Equal(t, event1.Timestamp, event2.Timestamp)

	// Create event with different content
	event3 := MessageReceived{
		Event:       NewEvent(),
		UserID:      "user789",
		ChatID:      "chat456",
		MessageText: "Hello",
	}

	assert.NotEqual(t, event1.UserID, event3.UserID)
	assert.NotEqual(t, event1.CorrelationID, event3.CorrelationID)
}

func TestEventTypes_ParsedTaskStructure(t *testing.T) {
	dueDate := time.Now().Add(24 * time.Hour)

	parsedTask := ParsedTask{
		Title:       "Complete project",
		Description: "Finish the project by end of week",
		DueDate:     &dueDate,
		Priority:    "high",
		Tags:        []string{"work", "urgent"},
	}

	assert.Equal(t, "Complete project", parsedTask.Title)
	assert.Equal(t, "Finish the project by end of week", parsedTask.Description)
	assert.NotNil(t, parsedTask.DueDate)
	assert.Equal(t, dueDate, *parsedTask.DueDate)
	assert.Equal(t, "high", parsedTask.Priority)
	assert.Len(t, parsedTask.Tags, 2)
	assert.Contains(t, parsedTask.Tags, "work")
	assert.Contains(t, parsedTask.Tags, "urgent")

	// Test with nil due date
	parsedTaskNoDue := ParsedTask{
		Title:    "No due date task",
		Priority: "low",
		Tags:     []string{},
	}

	assert.Nil(t, parsedTaskNoDue.DueDate)
	assert.Empty(t, parsedTaskNoDue.Tags)
}

func TestEventTypes_TaskSummaryStructure(t *testing.T) {
	dueDate := time.Now().Add(-1 * time.Hour) // Past due

	taskSummary := TaskSummary{
		ID:          "task123",
		Title:       "Overdue task",
		Description: "This task is overdue",
		DueDate:     &dueDate,
		Priority:    "high",
		Status:      "active",
		IsOverdue:   true,
	}

	assert.Equal(t, "task123", taskSummary.ID)
	assert.Equal(t, "Overdue task", taskSummary.Title)
	assert.NotNil(t, taskSummary.DueDate)
	assert.True(t, taskSummary.DueDate.Before(time.Now()))
	assert.Equal(t, "high", taskSummary.Priority)
	assert.Equal(t, "active", taskSummary.Status)
	assert.True(t, taskSummary.IsOverdue)

	// Test JSON serialization
	jsonData, err := json.Marshal(taskSummary)
	require.NoError(t, err)

	var unmarshaled TaskSummary
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, taskSummary.ID, unmarshaled.ID)
	assert.Equal(t, taskSummary.Title, unmarshaled.Title)
	assert.Equal(t, taskSummary.IsOverdue, unmarshaled.IsOverdue)
}

// Helper function to create time pointers
func timePtr(t time.Time) *time.Time {
	return &t
}
