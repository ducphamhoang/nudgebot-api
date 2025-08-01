package nudge

import (
	"fmt"
	"strings"
	"time"

	"nudgebot-api/internal/common"
)

// Business rule constants
const (
	MinTaskTitleLength     = 1
	MaxTaskTitleLength     = 255
	MaxTaskDescLength      = 2000
	MaxNudgeInterval       = 24 * time.Hour
	MinNudgeInterval       = 15 * time.Minute
	DefaultNudgeInterval   = time.Hour
	MaxNudgesPerTask       = 10
	DefaultMaxNudges       = 3
	ReminderLeadTime       = time.Hour // Default lead time before due date
	NudgeBackoffMultiplier = 2.0       // Exponential backoff for nudges
)

// TaskValidator provides validation for task operations
type TaskValidator struct{}

// NewTaskValidator creates a new TaskValidator
func NewTaskValidator() *TaskValidator {
	return &TaskValidator{}
}

// ValidateTask performs comprehensive validation on a task
func (v *TaskValidator) ValidateTask(task *Task) error {
	if task == nil {
		return NewTaskValidationError("task", nil, "task cannot be nil")
	}

	// Validate ID
	if task.ID == "" {
		return NewTaskValidationError("id", task.ID, "task ID is required")
	}
	if !common.ID(task.ID).IsValid() {
		return NewTaskValidationError("id", task.ID, "task ID must be a valid UUID")
	}

	// Validate UserID
	if task.UserID == "" {
		return NewTaskValidationError("user_id", task.UserID, "user ID is required")
	}
	if !common.ID(task.UserID).IsValid() {
		return NewTaskValidationError("user_id", task.UserID, "user ID must be a valid UUID")
	}

	// Validate Title
	if strings.TrimSpace(task.Title) == "" {
		return NewTaskValidationError("title", task.Title, "title is required")
	}
	if len(task.Title) < MinTaskTitleLength {
		return NewTaskValidationError("title", task.Title, fmt.Sprintf("title must be at least %d characters", MinTaskTitleLength))
	}
	if len(task.Title) > MaxTaskTitleLength {
		return NewTaskValidationError("title", task.Title, fmt.Sprintf("title cannot exceed %d characters", MaxTaskTitleLength))
	}

	// Validate Description
	if len(task.Description) > MaxTaskDescLength {
		return NewTaskValidationError("description", task.Description, fmt.Sprintf("description cannot exceed %d characters", MaxTaskDescLength))
	}

	// Validate Priority
	if !task.Priority.IsValid() {
		return NewTaskValidationError("priority", task.Priority, "invalid priority value")
	}

	// Validate Status
	if !task.Status.IsValid() {
		return NewTaskValidationError("status", task.Status, "invalid status value")
	}

	// Validate Due Date
	if task.DueDate != nil && task.DueDate.Before(time.Now().Add(-24*time.Hour)) {
		return NewTaskValidationError("due_date", task.DueDate, "due date cannot be more than 24 hours in the past")
	}

	// Validate CompletedAt
	if task.CompletedAt != nil && task.Status != common.TaskStatusCompleted {
		return NewTaskValidationError("completed_at", task.CompletedAt, "completed_at can only be set when status is completed")
	}

	if task.Status == common.TaskStatusCompleted && task.CompletedAt == nil {
		return NewTaskValidationError("completed_at", task.CompletedAt, "completed_at is required when status is completed")
	}

	return nil
}

// ValidateStatusTransition ensures valid status transitions
func (v *TaskValidator) ValidateStatusTransition(from, to common.TaskStatus) error {
	if from == to {
		return nil // No transition needed
	}

	switch from {
	case common.TaskStatusActive:
		// Active tasks can transition to any other status
		if to == common.TaskStatusCompleted || to == common.TaskStatusSnoozed || to == common.TaskStatusDeleted {
			return nil
		}
	case common.TaskStatusSnoozed:
		// Snoozed tasks can be activated, completed, or deleted
		if to == common.TaskStatusActive || to == common.TaskStatusCompleted || to == common.TaskStatusDeleted {
			return nil
		}
	case common.TaskStatusCompleted:
		// Completed tasks can only be deleted or reactivated
		if to == common.TaskStatusDeleted || to == common.TaskStatusActive {
			return nil
		}
	case common.TaskStatusDeleted:
		// Deleted tasks can only be reactivated
		if to == common.TaskStatusActive {
			return nil
		}
	}

	return NewStatusTransitionError(from, to, "invalid status transition")
}

// ValidateTaskFilter validates filter parameters for task queries
func (v *TaskValidator) ValidateTaskFilter(filter TaskFilter) error {
	// Validate UserID
	if filter.UserID == "" {
		return NewTaskValidationError("user_id", filter.UserID, "user ID is required in filter")
	}
	if !common.ID(filter.UserID).IsValid() {
		return NewTaskValidationError("user_id", filter.UserID, "user ID must be a valid UUID")
	}

	// Validate Status
	if filter.Status != nil && !filter.Status.IsValid() {
		return NewTaskValidationError("status", filter.Status, "invalid status in filter")
	}

	// Validate Priority
	if filter.Priority != nil && !filter.Priority.IsValid() {
		return NewTaskValidationError("priority", filter.Priority, "invalid priority in filter")
	}

	// Validate date range
	if filter.DueBefore != nil && filter.DueAfter != nil {
		if filter.DueBefore.Before(*filter.DueAfter) {
			return NewTaskValidationError("date_range", nil, "due_before must be after due_after")
		}
	}

	// Validate pagination
	if filter.Limit < 0 {
		return NewTaskValidationError("limit", filter.Limit, "limit cannot be negative")
	}
	if filter.Offset < 0 {
		return NewTaskValidationError("offset", filter.Offset, "offset cannot be negative")
	}
	if filter.Limit > 1000 {
		return NewTaskValidationError("limit", filter.Limit, "limit cannot exceed 1000")
	}

	return nil
}

// ReminderManager handles reminder scheduling and nudge logic
type ReminderManager struct{}

// NewReminderManager creates a new ReminderManager
func NewReminderManager() *ReminderManager {
	return &ReminderManager{}
}

// CalculateReminderTime calculates when to send the initial reminder
func (rm *ReminderManager) CalculateReminderTime(task *Task, settings *NudgeSettings) time.Time {
	if task.DueDate == nil {
		// For tasks without due dates, schedule reminder for immediate processing
		return time.Now().Add(5 * time.Minute)
	}

	// Calculate lead time based on priority
	var leadTime time.Duration
	switch task.Priority {
	case common.PriorityUrgent:
		leadTime = 2 * time.Hour
	case common.PriorityHigh:
		leadTime = 4 * time.Hour
	case common.PriorityMedium:
		leadTime = ReminderLeadTime
	case common.PriorityLow:
		leadTime = 30 * time.Minute
	default:
		leadTime = ReminderLeadTime
	}

	reminderTime := task.DueDate.Add(-leadTime)

	// Ensure reminder is not in the past
	if reminderTime.Before(time.Now()) {
		return time.Now().Add(1 * time.Minute)
	}

	return reminderTime
}

// ShouldCreateNudge determines if a nudge should be created based on task and settings
func (rm *ReminderManager) ShouldCreateNudge(task *Task, reminderCount int, settings *NudgeSettings) bool {
	// Don't nudge if nudging is disabled
	if !settings.Enabled {
		return false
	}

	// Don't nudge if we've reached the maximum nudge count
	if reminderCount >= settings.MaxNudges {
		return false
	}

	// Don't nudge if task is not in active status
	if task.Status != common.TaskStatusActive {
		return false
	}

	// Don't nudge if task doesn't have a due date
	if task.DueDate == nil {
		return false
	}

	// Create nudge if task is overdue or approaching due date
	now := time.Now()
	if task.DueDate.Before(now) || task.DueDate.Sub(now) <= settings.NudgeInterval {
		return true
	}

	return false
}

// GetNextNudgeTime calculates the next nudge time with exponential backoff
func (rm *ReminderManager) GetNextNudgeTime(lastNudge time.Time, settings *NudgeSettings) time.Time {
	// Use exponential backoff for subsequent nudges
	backoffInterval := time.Duration(float64(settings.NudgeInterval) * NudgeBackoffMultiplier)

	// Cap the maximum interval
	if backoffInterval > MaxNudgeInterval {
		backoffInterval = MaxNudgeInterval
	}

	return lastNudge.Add(backoffInterval)
}

// TaskStatusManager handles task status transitions and business rules
type TaskStatusManager struct{}

// NewTaskStatusManager creates a new TaskStatusManager
func NewTaskStatusManager() *TaskStatusManager {
	return &TaskStatusManager{}
}

// TransitionStatus handles status transitions with business rules
func (tsm *TaskStatusManager) TransitionStatus(task *Task, newStatus common.TaskStatus) error {
	validator := NewTaskValidator()

	// Validate the transition
	if err := validator.ValidateStatusTransition(task.Status, newStatus); err != nil {
		return err
	}

	// Apply business rules for specific transitions
	switch newStatus {
	case common.TaskStatusCompleted:
		return tsm.CompleteTask(task)
	case common.TaskStatusSnoozed:
		// For snoozing, we need a snooze time (handled separately)
		return NewBusinessRuleError("snooze_transition", "use SnoozeTask method for snoozing")
	case common.TaskStatusDeleted:
		return tsm.DeleteTask(task)
	case common.TaskStatusActive:
		return tsm.reactivateTask(task)
	}

	// For other transitions, just update the status
	task.Status = newStatus
	task.UpdatedAt = time.Now()

	return nil
}

// CompleteTask completes a task and sets completion timestamp
func (tsm *TaskStatusManager) CompleteTask(task *Task) error {
	if task.Status == common.TaskStatusCompleted {
		return NewBusinessRuleError("already_completed", "task is already completed")
	}

	now := time.Now()
	task.Status = common.TaskStatusCompleted
	task.CompletedAt = &now
	task.UpdatedAt = now

	return nil
}

// SnoozeTask snoozes a task and reschedules reminders
func (tsm *TaskStatusManager) SnoozeTask(task *Task, snoozeUntil time.Time) error {
	if task.Status != common.TaskStatusActive {
		return NewBusinessRuleError("invalid_snooze", "only active tasks can be snoozed")
	}

	if snoozeUntil.Before(time.Now()) {
		return NewTaskValidationError("snooze_until", snoozeUntil, "snooze time must be in the future")
	}

	// Update task status and due date
	task.Status = common.TaskStatusSnoozed
	task.DueDate = &snoozeUntil
	task.UpdatedAt = time.Now()

	return nil
}

// DeleteTask performs soft delete on a task
func (tsm *TaskStatusManager) DeleteTask(task *Task) error {
	if task.Status == common.TaskStatusDeleted {
		return NewBusinessRuleError("already_deleted", "task is already deleted")
	}

	task.Status = common.TaskStatusDeleted
	task.UpdatedAt = time.Now()

	return nil
}

// reactivateTask reactivates a task from snoozed, completed, or deleted status
func (tsm *TaskStatusManager) reactivateTask(task *Task) error {
	if task.Status == common.TaskStatusActive {
		return NewBusinessRuleError("already_active", "task is already active")
	}

	task.Status = common.TaskStatusActive
	task.UpdatedAt = time.Now()

	// Clear completion timestamp if reactivating a completed task
	if task.CompletedAt != nil {
		task.CompletedAt = nil
	}

	return nil
}

// Helper functions for common operations

// IsTaskOverdue checks if a task is overdue
func IsTaskOverdue(task *Task) bool {
	return task.IsOverdue()
}

// GetTaskPriorityWeight returns a numeric weight for sorting by priority
func GetTaskPriorityWeight(priority common.Priority) int {
	switch priority {
	case common.PriorityUrgent:
		return 4
	case common.PriorityHigh:
		return 3
	case common.PriorityMedium:
		return 2
	case common.PriorityLow:
		return 1
	default:
		return 2 // Default to medium
	}
}

// ValidateNudgeSettings validates nudge settings
func ValidateNudgeSettings(settings *NudgeSettings) error {
	if settings == nil {
		return NewTaskValidationError("settings", nil, "settings cannot be nil")
	}

	if settings.UserID == "" {
		return NewTaskValidationError("user_id", settings.UserID, "user ID is required")
	}

	if !common.ID(settings.UserID).IsValid() {
		return NewTaskValidationError("user_id", settings.UserID, "user ID must be a valid UUID")
	}

	if settings.NudgeInterval < MinNudgeInterval {
		return NewTaskValidationError("nudge_interval", settings.NudgeInterval, fmt.Sprintf("nudge interval must be at least %v", MinNudgeInterval))
	}

	if settings.NudgeInterval > MaxNudgeInterval {
		return NewTaskValidationError("nudge_interval", settings.NudgeInterval, fmt.Sprintf("nudge interval cannot exceed %v", MaxNudgeInterval))
	}

	if settings.MaxNudges < 0 {
		return NewTaskValidationError("max_nudges", settings.MaxNudges, "max nudges cannot be negative")
	}

	if settings.MaxNudges > MaxNudgesPerTask {
		return NewTaskValidationError("max_nudges", settings.MaxNudges, fmt.Sprintf("max nudges cannot exceed %d", MaxNudgesPerTask))
	}

	return nil
}
