package nudge

import (
	"time"

	"nudgebot-api/internal/common"
)

// Task represents a task in the nudge system
type Task struct {
	ID          common.TaskID     `json:"id" gorm:"primaryKey;type:varchar(36)" validate:"required"`
	UserID      common.UserID     `json:"user_id" gorm:"type:varchar(36);not null;index" validate:"required"`
	ChatID      common.ChatID     `json:"chat_id" gorm:"type:varchar(36);index"`
	Title       string            `json:"title" gorm:"type:varchar(255);not null" validate:"required"`
	Description string            `json:"description" gorm:"type:text"`
	DueDate     *time.Time        `json:"due_date" gorm:"type:timestamp"`
	Priority    common.Priority   `json:"priority" gorm:"type:varchar(20);not null;default:'medium'" validate:"required"`
	Status      common.TaskStatus `json:"status" gorm:"type:varchar(20);not null;default:'active'" validate:"required"`
	CreatedAt   time.Time         `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time         `json:"updated_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	CompletedAt *time.Time        `json:"completed_at" gorm:"type:timestamp"`
}

// Reminder represents a reminder for a task
type Reminder struct {
	ID           common.ID     `json:"id" gorm:"primaryKey;type:varchar(36)" validate:"required"`
	TaskID       common.TaskID `json:"task_id" gorm:"type:varchar(36);not null;index" validate:"required"`
	UserID       common.UserID `json:"user_id" gorm:"type:varchar(36);not null;index" validate:"required"`
	ChatID       common.ChatID `json:"chat_id" gorm:"type:varchar(36);not null;index" validate:"required"`
	ScheduledAt  time.Time     `json:"scheduled_at" gorm:"type:timestamp;not null" validate:"required"`
	SentAt       *time.Time    `json:"sent_at" gorm:"type:timestamp"`
	ReminderType ReminderType  `json:"reminder_type" gorm:"type:varchar(20);not null" validate:"required"`
}

// ReminderType represents the type of reminder
type ReminderType string

const (
	ReminderTypeInitial ReminderType = "initial"
	ReminderTypeNudge   ReminderType = "nudge"
)

// TaskFilter represents filtering options for querying tasks
type TaskFilter struct {
	UserID    common.UserID      `json:"user_id"`
	Status    *common.TaskStatus `json:"status,omitempty"`
	Priority  *common.Priority   `json:"priority,omitempty"`
	DueBefore *time.Time         `json:"due_before,omitempty"`
	DueAfter  *time.Time         `json:"due_after,omitempty"`
	Limit     int                `json:"limit,omitempty"`
	Offset    int                `json:"offset,omitempty"`
}

// TaskStats represents statistics about a user's tasks
type TaskStats struct {
	TotalTasks     int64 `json:"total_tasks"`
	CompletedTasks int64 `json:"completed_tasks"`
	OverdueTasks   int64 `json:"overdue_tasks"`
	ActiveTasks    int64 `json:"active_tasks"`
}

// NudgeSettings represents user-specific nudge settings
type NudgeSettings struct {
	UserID        common.UserID `json:"user_id" gorm:"primaryKey;type:varchar(36)" validate:"required"`
	NudgeInterval time.Duration `json:"nudge_interval" gorm:"type:bigint;not null;default:3600000000000"` // 1 hour in nanoseconds
	MaxNudges     int           `json:"max_nudges" gorm:"type:int;not null;default:3"`
	Enabled       bool          `json:"enabled" gorm:"type:boolean;not null;default:true"`
	CreatedAt     time.Time     `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time     `json:"updated_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
}

// IsValid checks if the reminder type is valid
func (rt ReminderType) IsValid() bool {
	switch rt {
	case ReminderTypeInitial, ReminderTypeNudge:
		return true
	default:
		return false
	}
}

// IsOverdue checks if the task is overdue
func (t Task) IsOverdue() bool {
	if t.DueDate == nil {
		return false
	}
	return time.Now().After(*t.DueDate) && t.Status == common.TaskStatusActive
}

// IsCompleted checks if the task is completed
func (t Task) IsCompleted() bool {
	return t.Status == common.TaskStatusCompleted
}

// CanBeNudged checks if the task can receive nudges
func (t Task) CanBeNudged() bool {
	return t.Status == common.TaskStatusActive && t.DueDate != nil
}

// TableName returns the table name for the Task model
func (Task) TableName() string {
	return "tasks"
}

// TableName returns the table name for the Reminder model
func (Reminder) TableName() string {
	return "reminders"
}

// TableName returns the table name for the NudgeSettings model
func (NudgeSettings) TableName() string {
	return "nudge_settings"
}
