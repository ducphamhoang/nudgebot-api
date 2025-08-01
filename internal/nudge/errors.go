package nudge

import (
	"fmt"

	"nudgebot-api/internal/common"
)

// Error codes for nudge module
const (
	ErrCodeTaskNotFound      = "TASK_NOT_FOUND"
	ErrCodeInvalidStatus     = "INVALID_STATUS"
	ErrCodeInvalidTransition = "INVALID_TRANSITION"
	ErrCodeValidationFailed  = "VALIDATION_FAILED"
	ErrCodeDuplicateTask     = "DUPLICATE_TASK"
	ErrCodeReminderFailed    = "REMINDER_FAILED"
	ErrCodeBusinessRule      = "BUSINESS_RULE_VIOLATION"
	ErrCodeRepository        = "REPOSITORY_ERROR"
)

// NudgeError interface for nudge-specific errors
type NudgeError interface {
	error
	Code() string
	Message() string
	Temporary() bool
}

// TaskValidationError represents validation failures for tasks
type TaskValidationError struct {
	Field      string
	Value      interface{}
	ErrMessage string
}

func (e TaskValidationError) Error() string {
	return fmt.Sprintf("task validation failed for field '%s': %s (value: %v)", e.Field, e.ErrMessage, e.Value)
}

func (e TaskValidationError) Code() string {
	return ErrCodeValidationFailed
}

func (e TaskValidationError) Message() string {
	return e.ErrMessage
}

func (e TaskValidationError) Temporary() bool {
	return false
}

// StatusTransitionError represents invalid status transition attempts
type StatusTransitionError struct {
	CurrentStatus common.TaskStatus
	TargetStatus  common.TaskStatus
	Reason        string
}

func (e StatusTransitionError) Error() string {
	return fmt.Sprintf("invalid status transition from '%s' to '%s': %s", e.CurrentStatus, e.TargetStatus, e.Reason)
}

func (e StatusTransitionError) Code() string {
	return ErrCodeInvalidTransition
}

func (e StatusTransitionError) Message() string {
	return e.Reason
}

func (e StatusTransitionError) Temporary() bool {
	return false
}

// ReminderSchedulingError represents failures in reminder creation or scheduling
type ReminderSchedulingError struct {
	TaskID     common.TaskID
	ErrMessage string
	Cause      error
}

func (e ReminderSchedulingError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("reminder scheduling failed for task '%s': %s (caused by: %v)", e.TaskID, e.ErrMessage, e.Cause)
	}
	return fmt.Sprintf("reminder scheduling failed for task '%s': %s", e.TaskID, e.ErrMessage)
}

func (e ReminderSchedulingError) Code() string {
	return ErrCodeReminderFailed
}

func (e ReminderSchedulingError) Message() string {
	return e.ErrMessage
}

func (e ReminderSchedulingError) Temporary() bool {
	return true // Reminder scheduling can often be retried
}

func (e ReminderSchedulingError) Unwrap() error {
	return e.Cause
}

// BusinessRuleError represents violations of business logic rules
type BusinessRuleError struct {
	Rule    string
	Details string
}

func (e BusinessRuleError) Error() string {
	return fmt.Sprintf("business rule violation: %s - %s", e.Rule, e.Details)
}

func (e BusinessRuleError) Code() string {
	return ErrCodeBusinessRule
}

func (e BusinessRuleError) Message() string {
	return e.Details
}

func (e BusinessRuleError) Temporary() bool {
	return false
}

// RepositoryError represents database operation failures
type RepositoryError struct {
	Operation string
	Details   string
	Cause     error
}

func (e RepositoryError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("repository error during %s: %s (caused by: %v)", e.Operation, e.Details, e.Cause)
	}
	return fmt.Sprintf("repository error during %s: %s", e.Operation, e.Details)
}

func (e RepositoryError) Code() string {
	return ErrCodeRepository
}

func (e RepositoryError) Message() string {
	return e.Details
}

func (e RepositoryError) Temporary() bool {
	return true // Database errors can often be retried
}

func (e RepositoryError) Unwrap() error {
	return e.Cause
}

// Error wrapping utilities

// WrapRepositoryError wraps an error as a RepositoryError
func WrapRepositoryError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return RepositoryError{
		Operation: operation,
		Details:   "database operation failed",
		Cause:     err,
	}
}

// WrapValidationError wraps an error as a TaskValidationError
func WrapValidationError(err error, field string) error {
	if err == nil {
		return nil
	}
	return TaskValidationError{
		Field:      field,
		ErrMessage: err.Error(),
	}
}

// NewBusinessRuleError creates a new BusinessRuleError
func NewBusinessRuleError(rule string, details string) error {
	return BusinessRuleError{
		Rule:    rule,
		Details: details,
	}
}

// NewTaskValidationError creates a new TaskValidationError
func NewTaskValidationError(field string, value interface{}, message string) error {
	return TaskValidationError{
		Field:      field,
		Value:      value,
		ErrMessage: message,
	}
}

// NewStatusTransitionError creates a new StatusTransitionError
func NewStatusTransitionError(current, target common.TaskStatus, reason string) error {
	return StatusTransitionError{
		CurrentStatus: current,
		TargetStatus:  target,
		Reason:        reason,
	}
}

// NewReminderSchedulingError creates a new ReminderSchedulingError
func NewReminderSchedulingError(taskID common.TaskID, message string, cause error) error {
	return ReminderSchedulingError{
		TaskID:     taskID,
		ErrMessage: message,
		Cause:      cause,
	}
}

// Error classification helpers

// IsNotFoundError checks if the error is a not found error
func IsNotFoundError(err error) bool {
	if nudgeErr, ok := err.(NudgeError); ok {
		return nudgeErr.Code() == ErrCodeTaskNotFound
	}
	if _, ok := err.(common.NotFoundError); ok {
		return true
	}
	return false
}

// IsValidationError checks if the error is a validation error
func IsValidationError(err error) bool {
	if nudgeErr, ok := err.(NudgeError); ok {
		return nudgeErr.Code() == ErrCodeValidationFailed
	}
	if _, ok := err.(common.ValidationError); ok {
		return true
	}
	return false
}

// IsTemporaryError checks if the error is temporary and can be retried
func IsTemporaryError(err error) bool {
	if nudgeErr, ok := err.(NudgeError); ok {
		return nudgeErr.Temporary()
	}
	return false
}

// IsBusinessRuleError checks if the error is a business rule violation
func IsBusinessRuleError(err error) bool {
	if nudgeErr, ok := err.(NudgeError); ok {
		return nudgeErr.Code() == ErrCodeBusinessRule
	}
	return false
}

// IsRepositoryError checks if the error is a repository error
func IsRepositoryError(err error) bool {
	if nudgeErr, ok := err.(NudgeError); ok {
		return nudgeErr.Code() == ErrCodeRepository
	}
	return false
}
