package mocks

import (
	"time"

	"nudgebot-api/internal/common"
	"nudgebot-api/internal/nudge"
)

//go:generate mockgen -source=../events/bus.go -destination=./event_bus_mock.go -package=mocks
//go:generate mockgen -source=../chatbot/service.go -destination=./chatbot_service_mock.go -package=mocks
//go:generate mockgen -source=../llm/service.go -destination=./llm_service_mock.go -package=mocks
//go:generate mockgen -source=../llm/provider.go -destination=./llm_provider_mock.go -package=mocks
//go:generate mockgen -source=../nudge/service.go -destination=./nudge_service_mock.go -package=mocks
//go:generate mockgen -source=../nudge/repository.go -destination=./nudge_repository_mock.go -package=mocks

// MockEventBus provides a mock implementation of the EventBus interface for testing
type MockEventBus struct {
	publishedEvents  []MockEvent
	subscriptions    map[string][]interface{}
	closed           bool
	publishError     error
	subscribeError   error
	unsubscribeError error
}

// MockEvent represents a published event for testing verification
type MockEvent struct {
	Topic string
	Data  interface{}
}

// NewMockEventBus creates a new mock event bus
func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		publishedEvents: make([]MockEvent, 0),
		subscriptions:   make(map[string][]interface{}),
	}
}

// Publish implements the EventBus interface
func (m *MockEventBus) Publish(topic string, data interface{}) error {
	if m.publishError != nil {
		return m.publishError
	}

	m.publishedEvents = append(m.publishedEvents, MockEvent{
		Topic: topic,
		Data:  data,
	})

	return nil
}

// Subscribe implements the EventBus interface
func (m *MockEventBus) Subscribe(topic string, handler interface{}) error {
	if m.subscribeError != nil {
		return m.subscribeError
	}

	if m.subscriptions[topic] == nil {
		m.subscriptions[topic] = make([]interface{}, 0)
	}
	m.subscriptions[topic] = append(m.subscriptions[topic], handler)

	return nil
}

// Unsubscribe implements the EventBus interface
func (m *MockEventBus) Unsubscribe(topic string, handler interface{}) error {
	if m.unsubscribeError != nil {
		return m.unsubscribeError
	}

	// Remove handler from subscriptions (simplified implementation)
	if handlers, exists := m.subscriptions[topic]; exists {
		for i, h := range handlers {
			if h == handler {
				m.subscriptions[topic] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
	}

	return nil
}

// Close implements the EventBus interface
func (m *MockEventBus) Close() error {
	m.closed = true
	return nil
}

// Test helper methods
func (m *MockEventBus) GetPublishedEvents() []MockEvent {
	return m.publishedEvents
}

func (m *MockEventBus) GetSubscriptions() map[string][]interface{} {
	return m.subscriptions
}

func (m *MockEventBus) SetPublishError(err error) {
	m.publishError = err
}

func (m *MockEventBus) SetSubscribeError(err error) {
	m.subscribeError = err
}

func (m *MockEventBus) SetUnsubscribeError(err error) {
	m.unsubscribeError = err
}

func (m *MockEventBus) IsClosed() bool {
	return m.closed
}

// MockNudgeRepository provides a mock implementation of the NudgeRepository interface
type MockNudgeRepository struct {
	tasks       map[common.TaskID]*nudge.Task
	reminders   map[common.ID]*nudge.Reminder
	settings    map[common.UserID]*nudge.NudgeSettings
	createError error
	getError    error
	updateError error
	deleteError error
}

// NewMockNudgeRepository creates a new mock nudge repository
func NewMockNudgeRepository() *MockNudgeRepository {
	return &MockNudgeRepository{
		tasks:     make(map[common.TaskID]*nudge.Task),
		reminders: make(map[common.ID]*nudge.Reminder),
		settings:  make(map[common.UserID]*nudge.NudgeSettings),
	}
}

// Task repository methods
func (m *MockNudgeRepository) CreateTask(task *nudge.Task) error {
	if m.createError != nil {
		return m.createError
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *MockNudgeRepository) GetTaskByID(taskID common.TaskID) (*nudge.Task, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	if task, exists := m.tasks[taskID]; exists {
		return task, nil
	}
	return nil, nudge.ErrTaskNotFound
}

func (m *MockNudgeRepository) GetTasksByUserID(userID common.UserID, filter nudge.TaskFilter) ([]*nudge.Task, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	var tasks []*nudge.Task
	for _, task := range m.tasks {
		if task.UserID == userID {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (m *MockNudgeRepository) UpdateTask(task *nudge.Task) error {
	if m.updateError != nil {
		return m.updateError
	}
	if _, exists := m.tasks[task.ID]; !exists {
		return nudge.ErrTaskNotFound
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *MockNudgeRepository) DeleteTask(taskID common.TaskID) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	if _, exists := m.tasks[taskID]; !exists {
		return nudge.ErrTaskNotFound
	}
	delete(m.tasks, taskID)
	return nil
}

func (m *MockNudgeRepository) GetTaskStats(userID common.UserID) (*nudge.TaskStats, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	stats := &nudge.TaskStats{}
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
func (m *MockNudgeRepository) CreateReminder(reminder *nudge.Reminder) error {
	if m.createError != nil {
		return m.createError
	}
	m.reminders[reminder.ID] = reminder
	return nil
}

func (m *MockNudgeRepository) GetDueReminders(before time.Time) ([]*nudge.Reminder, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	var dueReminders []*nudge.Reminder
	for _, reminder := range m.reminders {
		if reminder.ScheduledAt.Before(before) && reminder.SentAt == nil {
			dueReminders = append(dueReminders, reminder)
		}
	}

	return dueReminders, nil
}

func (m *MockNudgeRepository) MarkReminderSent(reminderID common.ID) error {
	if m.updateError != nil {
		return m.updateError
	}

	if reminder, exists := m.reminders[reminderID]; exists {
		now := time.Now()
		reminder.SentAt = &now
		return nil
	}

	return nudge.ErrReminderNotFound
}

func (m *MockNudgeRepository) GetRemindersByTaskID(taskID common.TaskID) ([]*nudge.Reminder, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	var reminders []*nudge.Reminder
	for _, reminder := range m.reminders {
		if reminder.TaskID == taskID {
			reminders = append(reminders, reminder)
		}
	}

	return reminders, nil
}

func (m *MockNudgeRepository) DeleteReminder(reminderID common.ID) error {
	if m.deleteError != nil {
		return m.deleteError
	}

	if _, exists := m.reminders[reminderID]; !exists {
		return nudge.ErrReminderNotFound
	}

	delete(m.reminders, reminderID)
	return nil
}

// Nudge settings repository methods
func (m *MockNudgeRepository) GetNudgeSettingsByUserID(userID common.UserID) (*nudge.NudgeSettings, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	if settings, exists := m.settings[userID]; exists {
		return settings, nil
	}

	return nil, common.NotFoundError{Resource: "NudgeSettings", ID: string(userID)}
}

func (m *MockNudgeRepository) CreateOrUpdateNudgeSettings(settings *nudge.NudgeSettings) error {
	if m.createError != nil {
		return m.createError
	}

	m.settings[settings.UserID] = settings
	return nil
}

func (m *MockNudgeRepository) DeleteNudgeSettings(userID common.UserID) error {
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
func (m *MockNudgeRepository) WithTransaction(fn func(nudge.NudgeRepository) error) error {
	// For mock, just execute the function with the same repository
	return fn(m)
}

// Test helper methods
func (m *MockNudgeRepository) SetCreateError(err error) {
	m.createError = err
}

func (m *MockNudgeRepository) SetGetError(err error) {
	m.getError = err
}

func (m *MockNudgeRepository) SetUpdateError(err error) {
	m.updateError = err
}

func (m *MockNudgeRepository) SetDeleteError(err error) {
	m.deleteError = err
}

func (m *MockNudgeRepository) GetTaskCount() int {
	return len(m.tasks)
}

func (m *MockNudgeRepository) GetReminderCount() int {
	return len(m.reminders)
}

func (m *MockNudgeRepository) GetSettingsCount() int {
	return len(m.settings)
}
