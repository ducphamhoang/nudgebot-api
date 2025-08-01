package common

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// ID represents a unique identifier
type ID string

// NewID generates a new unique identifier
func NewID() ID {
	return ID(uuid.New().String())
}

// IsValid checks if the ID is a valid UUID
func (id ID) IsValid() bool {
	_, err := uuid.Parse(string(id))
	return err == nil
}

// String returns the string representation of the ID
func (id ID) String() string {
	return string(id)
}

// MarshalJSON implements json.Marshaler
func (id ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(id))
}

// UnmarshalJSON implements json.Unmarshaler
func (id *ID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*id = ID(s)
	return nil
}

// Typed aliases for different ID types
type (
	UserID ID
	ChatID ID
	TaskID ID
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusActive    TaskStatus = "active"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusSnoozed   TaskStatus = "snoozed"
	TaskStatusDeleted   TaskStatus = "deleted"
)

// String returns the string representation of TaskStatus
func (ts TaskStatus) String() string {
	return string(ts)
}

// IsValid checks if the TaskStatus is valid
func (ts TaskStatus) IsValid() bool {
	switch ts {
	case TaskStatusActive, TaskStatusCompleted, TaskStatusSnoozed, TaskStatusDeleted:
		return true
	default:
		return false
	}
}

// Priority represents the priority level of a task
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

// String returns the string representation of Priority
func (p Priority) String() string {
	return string(p)
}

// IsValid checks if the Priority is valid
func (p Priority) IsValid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent:
		return true
	default:
		return false
	}
}

// Common error types
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

type NotFoundError struct {
	Resource string
	ID       string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("%s with ID '%s' not found", e.Resource, e.ID)
}

type InternalError struct {
	Message string
	Cause   error
}

func (e InternalError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("internal error: %s (caused by: %v)", e.Message, e.Cause)
	}
	return fmt.Sprintf("internal error: %s", e.Message)
}

func (e InternalError) Unwrap() error {
	return e.Cause
}
