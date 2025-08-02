package events

import (
	"time"

	"github.com/google/uuid"
)

// Event represents the base event structure with common fields
type Event struct {
	CorrelationID string    `json:"correlation_id" validate:"required"`
	Timestamp     time.Time `json:"timestamp" validate:"required"`
}

// NewEvent creates a new base event with generated correlation ID
func NewEvent() Event {
	return Event{
		CorrelationID: uuid.New().String(),
		Timestamp:     time.Now(),
	}
}

// MessageReceived represents an event when a message is received from a user
type MessageReceived struct {
	Event
	UserID      string `json:"user_id" validate:"required"`
	ChatID      string `json:"chat_id" validate:"required"`
	MessageText string `json:"message_text" validate:"required"`
}

// ParsedTask represents a task that has been parsed from natural language
type ParsedTask struct {
	Title       string     `json:"title" validate:"required"`
	Description string     `json:"description"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Priority    string     `json:"priority" validate:"required"`
	Tags        []string   `json:"tags"`
}

// TaskParsed represents an event when a task has been successfully parsed
type TaskParsed struct {
	Event
	UserID     string     `json:"user_id" validate:"required"`
	ChatID     string     `json:"chat_id" validate:"required"`
	ParsedTask ParsedTask `json:"parsed_task" validate:"required"`
}

// ReminderDue represents an event when a reminder is due to be sent
type ReminderDue struct {
	Event
	TaskID string `json:"task_id" validate:"required"`
	UserID string `json:"user_id" validate:"required"`
	ChatID string `json:"chat_id" validate:"required"`
}

// TaskCompleted represents an event when a task has been completed
type TaskCompleted struct {
	Event
	TaskID      string    `json:"task_id" validate:"required"`
	UserID      string    `json:"user_id" validate:"required"`
	CompletedAt time.Time `json:"completed_at" validate:"required"`
}

// TaskCreated represents an event when a new task has been created
type TaskCreated struct {
	Event
	TaskID    string     `json:"task_id" validate:"required"`
	UserID    string     `json:"user_id" validate:"required"`
	Title     string     `json:"title" validate:"required"`
	DueDate   *time.Time `json:"due_date,omitempty"`
	Priority  string     `json:"priority" validate:"required"`
	CreatedAt time.Time  `json:"created_at" validate:"required"`
}

// TaskListRequested represents an event when a user requests their task list
type TaskListRequested struct {
	Event
	UserID string `json:"user_id" validate:"required"`
	ChatID string `json:"chat_id" validate:"required"`
}

// TaskActionRequested represents an event when a user requests a task action
type TaskActionRequested struct {
	Event
	UserID string `json:"user_id" validate:"required"`
	ChatID string `json:"chat_id" validate:"required"`
	TaskID string `json:"task_id" validate:"required"`
	Action string `json:"action" validate:"required"` // done, delete, snooze
}

// UserSessionStarted represents an event when a user starts a session
type UserSessionStarted struct {
	Event
	UserID      string `json:"user_id" validate:"required"`
	ChatID      string `json:"chat_id" validate:"required"`
	SessionType string `json:"session_type" validate:"required"`
}

// CommandExecuted represents an event when a command is executed
type CommandExecuted struct {
	Event
	UserID       string `json:"user_id" validate:"required"`
	ChatID       string `json:"chat_id" validate:"required"`
	Command      string `json:"command" validate:"required"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// TaskSummary represents a lightweight task representation for responses
type TaskSummary struct {
	ID          string     `json:"id" validate:"required"`
	Title       string     `json:"title" validate:"required"`
	Description string     `json:"description"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Priority    string     `json:"priority" validate:"required"`
	Status      string     `json:"status" validate:"required"`
	IsOverdue   bool       `json:"is_overdue"`
}

// TaskListResponse represents an event response to task list requests
type TaskListResponse struct {
	Event
	UserID     string        `json:"user_id" validate:"required"`
	ChatID     string        `json:"chat_id" validate:"required"`
	Tasks      []TaskSummary `json:"tasks"`
	TotalCount int           `json:"total_count"`
	HasMore    bool          `json:"has_more"`
}

// TaskActionResponse represents an event response to task action requests
type TaskActionResponse struct {
	Event
	UserID  string `json:"user_id" validate:"required"`
	ChatID  string `json:"chat_id" validate:"required"`
	TaskID  string `json:"task_id" validate:"required"`
	Action  string `json:"action" validate:"required"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Event topics constants
const (
	TopicMessageReceived     = "message.received"
	TopicTaskParsed          = "task.parsed"
	TopicReminderDue         = "reminder.due"
	TopicTaskCompleted       = "task.completed"
	TopicTaskCreated         = "task.created"
	TopicTaskListRequested   = "task.list.requested"
	TopicTaskActionRequested = "task.action.requested"
	TopicUserSessionStarted  = "user.session.started"
	TopicCommandExecuted     = "command.executed"
	TopicTaskListResponse    = "task.list.response"
	TopicTaskActionResponse  = "task.action.response"
)
