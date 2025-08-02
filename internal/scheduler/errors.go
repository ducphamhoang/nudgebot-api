package scheduler

import (
	"fmt"
)

// SchedulerError defines the interface for scheduler-specific errors
type SchedulerError interface {
	error
	Code() string
	Message() string
	Temporary() bool
}

// schedulerError implements the SchedulerError interface
type schedulerError struct {
	code      string
	message   string
	temporary bool
}

func (e *schedulerError) Error() string {
	return fmt.Sprintf("scheduler error [%s]: %s", e.code, e.message)
}

func (e *schedulerError) Code() string {
	return e.code
}

func (e *schedulerError) Message() string {
	return e.message
}

func (e *schedulerError) Temporary() bool {
	return e.temporary
}

// Error constants
const (
	ErrSchedulerNotRunning      = "scheduler_not_running"
	ErrSchedulerAlreadyRunning  = "scheduler_already_running"
	ErrInvalidConfiguration     = "invalid_configuration"
	ErrReminderProcessingFailed = "reminder_processing_failed"
	ErrNudgeCreationFailed      = "nudge_creation_failed"
	ErrWorkerPanic              = "worker_panic"
)

// Specific error types
type ReminderProcessingError struct {
	schedulerError
	ReminderID string
	Operation  string
}

type NudgeCreationError struct {
	schedulerError
	TaskID    string
	Operation string
}

type WorkerError struct {
	schedulerError
	WorkerID  int
	Operation string
}

type ShutdownError struct {
	schedulerError
	TimeoutSeconds int
}

type ConfigurationError struct {
	schedulerError
	Field string
	Value interface{}
}

// Constructor functions
func NewSchedulerError(code, message string) error {
	return &schedulerError{
		code:      code,
		message:   message,
		temporary: false,
	}
}

func NewTemporarySchedulerError(code, message string) error {
	return &schedulerError{
		code:      code,
		message:   message,
		temporary: true,
	}
}

func NewReminderProcessingError(reminderID, operation string, err error) error {
	return &ReminderProcessingError{
		schedulerError: schedulerError{
			code:      ErrReminderProcessingFailed,
			message:   fmt.Sprintf("failed to process reminder %s during %s: %v", reminderID, operation, err),
			temporary: true,
		},
		ReminderID: reminderID,
		Operation:  operation,
	}
}

func NewNudgeCreationError(taskID, operation string, err error) error {
	return &NudgeCreationError{
		schedulerError: schedulerError{
			code:      ErrNudgeCreationFailed,
			message:   fmt.Sprintf("failed to create nudge for task %s during %s: %v", taskID, operation, err),
			temporary: true,
		},
		TaskID:    taskID,
		Operation: operation,
	}
}

func NewWorkerError(workerID int, operation string, err error) error {
	return &WorkerError{
		schedulerError: schedulerError{
			code:      ErrWorkerPanic,
			message:   fmt.Sprintf("worker %d failed during %s: %v", workerID, operation, err),
			temporary: true,
		},
		WorkerID:  workerID,
		Operation: operation,
	}
}

func NewShutdownError(message string, timeoutSeconds int) error {
	return &ShutdownError{
		schedulerError: schedulerError{
			code:      "shutdown_timeout",
			message:   message,
			temporary: false,
		},
		TimeoutSeconds: timeoutSeconds,
	}
}

func NewConfigurationError(field string, value interface{}, message string) error {
	return &ConfigurationError{
		schedulerError: schedulerError{
			code:      ErrInvalidConfiguration,
			message:   fmt.Sprintf("invalid configuration for field %s (value: %v): %s", field, value, message),
			temporary: false,
		},
		Field: field,
		Value: value,
	}
}

// Error classification helpers
func IsRetryableError(err error) bool {
	if schedErr, ok := err.(SchedulerError); ok {
		return schedErr.Temporary()
	}
	return false
}

func IsTemporaryError(err error) bool {
	if schedErr, ok := err.(SchedulerError); ok {
		return schedErr.Temporary()
	}
	return false
}

func IsConfigurationError(err error) bool {
	if schedErr, ok := err.(SchedulerError); ok {
		return schedErr.Code() == ErrInvalidConfiguration
	}
	return false
}

// Error wrapping utilities
func WrapReminderError(err error, reminderID, operation string) error {
	return NewReminderProcessingError(reminderID, operation, err)
}

func WrapWorkerError(err error, workerID int, operation string) error {
	return NewWorkerError(workerID, operation, err)
}
