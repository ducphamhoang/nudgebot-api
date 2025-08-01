package nudge

import (
	"time"

	"nudgebot-api/internal/common"
)

// MockTaskRepository provides a mock implementation for testing
type MockTaskRepository struct {
	tasks       map[common.TaskID]*Task
	reminders   map[common.ID]*Reminder
	settings    map[common.UserID]*NudgeSettings
	createError error
	getError    error
	updateError error
	deleteError error
}

// NewMockTaskRepository creates a new mock repository
func NewMockTaskRepository() *MockTaskRepository {
	return &MockTaskRepository{
		tasks:     make(map[common.TaskID]*Task),
		reminders: make(map[common.ID]*Reminder),
		settings:  make(map[common.UserID]*NudgeSettings),
	}
}

// Task repository methods
func (m *MockTaskRepository) CreateTask(task *Task) error {
	if m.createError != nil {
		return m.createError
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *MockTaskRepository) GetTaskByID(taskID common.TaskID) (*Task, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	if task, exists := m.tasks[taskID]; exists {
		return task, nil
	}
	return nil, ErrTaskNotFound
}

func (m *MockTaskRepository) GetTasksByUserID(userID common.UserID, filter TaskFilter) ([]*Task, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	var tasks []*Task
	for _, task := range m.tasks {
		if task.UserID == userID {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (m *MockTaskRepository) UpdateTask(task *Task) error {
	if m.updateError != nil {
		return m.updateError
	}
	if _, exists := m.tasks[task.ID]; !exists {
		return ErrTaskNotFound
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *MockTaskRepository) DeleteTask(taskID common.TaskID) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	if _, exists := m.tasks[taskID]; !exists {
		return ErrTaskNotFound
	}
	delete(m.tasks, taskID)
	return nil
}

func (m *MockTaskRepository) GetTaskStats(userID common.UserID) (*TaskStats, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	stats := &TaskStats{}
	for _, task := range m.tasks {
		if task.UserID == userID {
			stats.TotalTasks++
			if task.Status == common.TaskStatusCompleted {
				stats.CompletedTasks++
			} else if task.Status == common.TaskStatusActive {
				stats.ActiveTasks++
				if task.IsOverdue() {
					stats.OverdueTasks++
				}
			}
		}
	}

	return stats, nil
}

// Reminder repository methods
func (m *MockTaskRepository) CreateReminder(reminder *Reminder) error {
	if m.createError != nil {
		return m.createError
	}
	m.reminders[reminder.ID] = reminder
	return nil
}

func (m *MockTaskRepository) GetDueReminders(before time.Time) ([]*Reminder, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	var dueReminders []*Reminder
	for _, reminder := range m.reminders {
		if reminder.ScheduledAt.Before(before) && reminder.SentAt == nil {
			dueReminders = append(dueReminders, reminder)
		}
	}

	return dueReminders, nil
}

func (m *MockTaskRepository) MarkReminderSent(reminderID common.ID) error {
	if m.updateError != nil {
		return m.updateError
	}

	if reminder, exists := m.reminders[reminderID]; exists {
		now := time.Now()
		reminder.SentAt = &now
		return nil
	}

	return ErrReminderNotFound
}

func (m *MockTaskRepository) GetRemindersByTaskID(taskID common.TaskID) ([]*Reminder, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	var reminders []*Reminder
	for _, reminder := range m.reminders {
		if reminder.TaskID == taskID {
			reminders = append(reminders, reminder)
		}
	}

	return reminders, nil
}

func (m *MockTaskRepository) DeleteReminder(reminderID common.ID) error {
	if m.deleteError != nil {
		return m.deleteError
	}

	if _, exists := m.reminders[reminderID]; !exists {
		return ErrReminderNotFound
	}

	delete(m.reminders, reminderID)
	return nil
}

// Nudge settings repository methods
func (m *MockTaskRepository) GetNudgeSettingsByUserID(userID common.UserID) (*NudgeSettings, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	if settings, exists := m.settings[userID]; exists {
		return settings, nil
	}

	return nil, common.NotFoundError{Resource: "NudgeSettings", ID: string(userID)}
}

func (m *MockTaskRepository) CreateOrUpdateNudgeSettings(settings *NudgeSettings) error {
	if m.createError != nil {
		return m.createError
	}

	m.settings[settings.UserID] = settings
	return nil
}

func (m *MockTaskRepository) DeleteNudgeSettings(userID common.UserID) error {
	if m.deleteError != nil {
		return m.deleteError
	}

	if _, exists := m.settings[userID]; !exists {
		return common.NotFoundError{Resource: "NudgeSettings", ID: string(userID)}
	}

	delete(m.settings, userID)
	return nil
}

// Transaction support
func (m *MockTaskRepository) WithTransaction(fn func(NudgeRepository) error) error {
	// For mock, just execute the function with the same repository
	return fn(m)
}

// Test helper methods
func (m *MockTaskRepository) SetCreateError(err error) {
	m.createError = err
}

func (m *MockTaskRepository) SetGetError(err error) {
	m.getError = err
}

func (m *MockTaskRepository) SetUpdateError(err error) {
	m.updateError = err
}

func (m *MockTaskRepository) SetDeleteError(err error) {
	m.deleteError = err
}

func (m *MockTaskRepository) GetTaskCount() int {
	return len(m.tasks)
}

func (m *MockTaskRepository) GetReminderCount() int {
	return len(m.reminders)
}

func (m *MockTaskRepository) GetSettingsCount() int {
	return len(m.settings)
}
