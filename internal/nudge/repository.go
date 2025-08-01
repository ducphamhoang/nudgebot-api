package nudge

import (
	"errors"
	"time"

	"nudgebot-api/internal/common"
)

// Repository errors
var (
	ErrTaskNotFound     = errors.New("task not found")
	ErrReminderNotFound = errors.New("reminder not found")
	ErrDuplicateTask    = errors.New("duplicate task")
	ErrInvalidFilter    = errors.New("invalid filter parameters")
)

// TaskRepository defines the interface for task data access
type TaskRepository interface {
	Create(task *Task) error
	GetByID(taskID common.TaskID) (*Task, error)
	GetByUserID(userID common.UserID, filter TaskFilter) ([]*Task, error)
	Update(task *Task) error
	Delete(taskID common.TaskID) error
	GetStats(userID common.UserID) (*TaskStats, error)
}

// ReminderRepository defines the interface for reminder data access
type ReminderRepository interface {
	Create(reminder *Reminder) error
	GetDueReminders(before time.Time) ([]*Reminder, error)
	MarkSent(reminderID common.ID) error
	GetByTaskID(taskID common.TaskID) ([]*Reminder, error)
	Delete(reminderID common.ID) error
}

// NudgeSettingsRepository defines the interface for nudge settings data access
type NudgeSettingsRepository interface {
	GetByUserID(userID common.UserID) (*NudgeSettings, error)
	CreateOrUpdate(settings *NudgeSettings) error
	Delete(userID common.UserID) error
}

// NudgeRepository combines all repository interfaces for unified data access
type NudgeRepository interface {
	// Task operations
	CreateTask(task *Task) error
	GetTaskByID(taskID common.TaskID) (*Task, error)
	GetTasksByUserID(userID common.UserID, filter TaskFilter) ([]*Task, error)
	UpdateTask(task *Task) error
	DeleteTask(taskID common.TaskID) error
	GetTaskStats(userID common.UserID) (*TaskStats, error)

	// Reminder operations
	CreateReminder(reminder *Reminder) error
	GetDueReminders(before time.Time) ([]*Reminder, error)
	MarkReminderSent(reminderID common.ID) error
	GetRemindersByTaskID(taskID common.TaskID) ([]*Reminder, error)
	DeleteReminder(reminderID common.ID) error

	// Nudge settings operations
	GetNudgeSettingsByUserID(userID common.UserID) (*NudgeSettings, error)
	CreateOrUpdateNudgeSettings(settings *NudgeSettings) error
	DeleteNudgeSettings(userID common.UserID) error

	// Transaction support
	WithTransaction(fn func(NudgeRepository) error) error
}
