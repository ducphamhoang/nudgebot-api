//go:generate mockgen -source=../nudge/repository.go -destination=nudge_repository_mocks.go -package=mocks

package mocks

import (
	"sync"
	"time"

	"nudgebot-api/internal/common"
	"nudgebot-api/internal/nudge"
)

// EnhancedMockNudgeRepository provides an advanced in-memory implementation for testing
type EnhancedMockNudgeRepository struct {
	tasks     map[string]*nudge.Task
	reminders map[string]*nudge.Reminder
	settings  map[string]*nudge.NudgeSettings
	mutex     sync.RWMutex
	errors    map[string]error
	callCount map[string]int
}

// NewEnhancedMockNudgeRepository creates a new enhanced mock repository
func NewEnhancedMockNudgeRepository() *EnhancedMockNudgeRepository {
	return &EnhancedMockNudgeRepository{
		tasks:     make(map[string]*nudge.Task),
		reminders: make(map[string]*nudge.Reminder),
		settings:  make(map[string]*nudge.NudgeSettings),
		errors:    make(map[string]error),
		callCount: make(map[string]int),
	}
}

// Helper methods for test setup

// SetupTestData populates the mock with sample data
func (m *EnhancedMockNudgeRepository) SetupTestData() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Create test tasks
	task1 := m.createTestTask("user1", "Test Task 1")
	task2 := m.createOverdueTask("user1")
	task3 := m.createCompletedTask("user1")

	m.tasks[string(task1.ID)] = task1
	m.tasks[string(task2.ID)] = task2
	m.tasks[string(task3.ID)] = task3

	// Create test reminders
	reminder1 := m.createTestReminder(task1.ID, task1.UserID)
	m.reminders[string(reminder1.ID)] = reminder1

	// Create test settings
	settings := &nudge.NudgeSettings{
		UserID:        "user1",
		NudgeInterval: time.Hour,
		MaxNudges:     3,
		Enabled:       true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	m.settings["user1"] = settings
}

// ClearData resets all data
func (m *EnhancedMockNudgeRepository) ClearData() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.tasks = make(map[string]*nudge.Task)
	m.reminders = make(map[string]*nudge.Reminder)
	m.settings = make(map[string]*nudge.NudgeSettings)
	m.callCount = make(map[string]int)
}

// SetError configures the mock to return an error for a specific operation
func (m *EnhancedMockNudgeRepository) SetError(operation string, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.errors[operation] = err
}

// GetCallCount returns the number of times an operation was called
func (m *EnhancedMockNudgeRepository) GetCallCount(operation string) int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.callCount[operation]
}

// incrementCallCount increments the call count for an operation
func (m *EnhancedMockNudgeRepository) incrementCallCount(operation string) {
	m.callCount[operation]++
}

// checkError returns any configured error for an operation
func (m *EnhancedMockNudgeRepository) checkError(operation string) error {
	if err, exists := m.errors[operation]; exists {
		return err
	}
	return nil
}

// Task operations - implementing the NudgeRepository interface

// CreateTask creates a new task
func (m *EnhancedMockNudgeRepository) CreateTask(task *nudge.Task) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.incrementCallCount("CreateTask")

	if err := m.checkError("CreateTask"); err != nil {
		return err
	}

	// Check for duplicates
	for _, existingTask := range m.tasks {
		if existingTask.UserID == task.UserID && existingTask.Title == task.Title &&
			(existingTask.Status == common.TaskStatusActive || existingTask.Status == common.TaskStatusSnoozed) {
			return nudge.NewTaskValidationError("title", task.Title, "task with this title already exists")
		}
	}

	// Set timestamps
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now

	// Generate ID if not set
	if task.ID == "" {
		task.ID = common.TaskID(common.NewID())
	}

	m.tasks[string(task.ID)] = task
	return nil
}

// GetTaskByID retrieves a task by ID
func (m *EnhancedMockNudgeRepository) GetTaskByID(taskID common.TaskID) (*nudge.Task, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	m.incrementCallCount("GetTaskByID")

	if err := m.checkError("GetTaskByID"); err != nil {
		return nil, err
	}

	task, exists := m.tasks[string(taskID)]
	if !exists {
		return nil, common.NotFoundError{Resource: "Task", ID: string(taskID)}
	}

	// Return a copy
	taskCopy := *task
	return &taskCopy, nil
}

// GetTasksByUserID retrieves tasks for a user with filtering
func (m *EnhancedMockNudgeRepository) GetTasksByUserID(userID common.UserID, filter nudge.TaskFilter) ([]*nudge.Task, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	m.incrementCallCount("GetTasksByUserID")

	if err := m.checkError("GetTasksByUserID"); err != nil {
		return nil, err
	}

	var result []*nudge.Task
	for _, task := range m.tasks {
		if task.UserID != userID {
			continue
		}

		// Apply filters
		if filter.Status != nil && task.Status != *filter.Status {
			continue
		}
		if filter.Priority != nil && task.Priority != *filter.Priority {
			continue
		}
		if filter.DueBefore != nil && (task.DueDate == nil || task.DueDate.After(*filter.DueBefore)) {
			continue
		}
		if filter.DueAfter != nil && (task.DueDate == nil || task.DueDate.Before(*filter.DueAfter)) {
			continue
		}

		// Create a copy
		taskCopy := *task
		result = append(result, &taskCopy)
	}

	// Apply pagination
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result, nil
}

// UpdateTask updates an existing task
func (m *EnhancedMockNudgeRepository) UpdateTask(task *nudge.Task) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.incrementCallCount("UpdateTask")

	if err := m.checkError("UpdateTask"); err != nil {
		return err
	}

	if _, exists := m.tasks[string(task.ID)]; !exists {
		return common.NotFoundError{Resource: "Task", ID: string(task.ID)}
	}

	task.UpdatedAt = time.Now()
	m.tasks[string(task.ID)] = task
	return nil
}

// DeleteTask performs soft delete on a task
func (m *EnhancedMockNudgeRepository) DeleteTask(taskID common.TaskID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.incrementCallCount("DeleteTask")

	if err := m.checkError("DeleteTask"); err != nil {
		return err
	}

	task, exists := m.tasks[string(taskID)]
	if !exists {
		return common.NotFoundError{Resource: "Task", ID: string(taskID)}
	}

	task.Status = common.TaskStatusDeleted
	task.UpdatedAt = time.Now()
	return nil
}

// GetTaskStats retrieves task statistics for a user
func (m *EnhancedMockNudgeRepository) GetTaskStats(userID common.UserID) (*nudge.TaskStats, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	m.incrementCallCount("GetTaskStats")

	if err := m.checkError("GetTaskStats"); err != nil {
		return nil, err
	}

	stats := &nudge.TaskStats{}
	now := time.Now()

	for _, task := range m.tasks {
		if task.UserID != userID || task.Status == common.TaskStatusDeleted {
			continue
		}

		stats.TotalTasks++

		switch task.Status {
		case common.TaskStatusCompleted:
			stats.CompletedTasks++
		case common.TaskStatusActive:
			stats.ActiveTasks++
			if task.DueDate != nil && task.DueDate.Before(now) {
				stats.OverdueTasks++
			}
		}
	}

	return stats, nil
}

// Reminder operations

// CreateReminder creates a new reminder
func (m *EnhancedMockNudgeRepository) CreateReminder(reminder *nudge.Reminder) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.incrementCallCount("CreateReminder")

	if err := m.checkError("CreateReminder"); err != nil {
		return err
	}

	// Verify task exists
	if _, exists := m.tasks[string(reminder.TaskID)]; !exists {
		return common.NotFoundError{Resource: "Task", ID: string(reminder.TaskID)}
	}

	// Generate ID if not set
	if reminder.ID == "" {
		reminder.ID = common.ID(common.NewID())
	}

	m.reminders[string(reminder.ID)] = reminder
	return nil
}

// GetDueReminders retrieves reminders due before the specified time
func (m *EnhancedMockNudgeRepository) GetDueReminders(before time.Time) ([]*nudge.Reminder, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	m.incrementCallCount("GetDueReminders")

	if err := m.checkError("GetDueReminders"); err != nil {
		return nil, err
	}

	var result []*nudge.Reminder
	for _, reminder := range m.reminders {
		if reminder.ScheduledAt.Before(before) && reminder.SentAt == nil {
			reminderCopy := *reminder
			result = append(result, &reminderCopy)
		}
	}

	return result, nil
}

// MarkReminderSent marks a reminder as sent
func (m *EnhancedMockNudgeRepository) MarkReminderSent(reminderID common.ID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.incrementCallCount("MarkReminderSent")

	if err := m.checkError("MarkReminderSent"); err != nil {
		return err
	}

	reminder, exists := m.reminders[string(reminderID)]
	if !exists || reminder.SentAt != nil {
		return common.NotFoundError{Resource: "Reminder", ID: string(reminderID)}
	}

	now := time.Now()
	reminder.SentAt = &now
	return nil
}

// GetRemindersByTaskID retrieves reminders for a specific task
func (m *EnhancedMockNudgeRepository) GetRemindersByTaskID(taskID common.TaskID) ([]*nudge.Reminder, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	m.incrementCallCount("GetRemindersByTaskID")

	if err := m.checkError("GetRemindersByTaskID"); err != nil {
		return nil, err
	}

	var result []*nudge.Reminder
	for _, reminder := range m.reminders {
		if reminder.TaskID == taskID {
			reminderCopy := *reminder
			result = append(result, &reminderCopy)
		}
	}

	return result, nil
}

// DeleteReminder deletes a reminder
func (m *EnhancedMockNudgeRepository) DeleteReminder(reminderID common.ID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.incrementCallCount("DeleteReminder")

	if err := m.checkError("DeleteReminder"); err != nil {
		return err
	}

	if _, exists := m.reminders[string(reminderID)]; !exists {
		return common.NotFoundError{Resource: "Reminder", ID: string(reminderID)}
	}

	delete(m.reminders, string(reminderID))
	return nil
}

// Nudge settings operations

// GetNudgeSettingsByUserID retrieves nudge settings for a user
func (m *EnhancedMockNudgeRepository) GetNudgeSettingsByUserID(userID common.UserID) (*nudge.NudgeSettings, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	m.incrementCallCount("GetNudgeSettingsByUserID")

	if err := m.checkError("GetNudgeSettingsByUserID"); err != nil {
		return nil, err
	}

	settings, exists := m.settings[string(userID)]
	if !exists {
		// Return default settings
		now := time.Now()
		return &nudge.NudgeSettings{
			UserID:        userID,
			NudgeInterval: time.Hour,
			MaxNudges:     3,
			Enabled:       true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}, nil
	}

	// Return a copy
	settingsCopy := *settings
	return &settingsCopy, nil
}

// CreateOrUpdateNudgeSettings creates or updates nudge settings
func (m *EnhancedMockNudgeRepository) CreateOrUpdateNudgeSettings(settings *nudge.NudgeSettings) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.incrementCallCount("CreateOrUpdateNudgeSettings")

	if err := m.checkError("CreateOrUpdateNudgeSettings"); err != nil {
		return err
	}

	now := time.Now()
	if settings.CreatedAt.IsZero() {
		settings.CreatedAt = now
	}
	settings.UpdatedAt = now

	m.settings[string(settings.UserID)] = settings
	return nil
}

// DeleteNudgeSettings deletes nudge settings for a user
func (m *EnhancedMockNudgeRepository) DeleteNudgeSettings(userID common.UserID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.incrementCallCount("DeleteNudgeSettings")

	if err := m.checkError("DeleteNudgeSettings"); err != nil {
		return err
	}

	if _, exists := m.settings[string(userID)]; !exists {
		return common.NotFoundError{Resource: "NudgeSettings", ID: string(userID)}
	}

	delete(m.settings, string(userID))
	return nil
}

// WithTransaction executes a function within a simulated transaction
func (m *EnhancedMockNudgeRepository) WithTransaction(fn func(nudge.NudgeRepository) error) error {
	m.incrementCallCount("WithTransaction")

	if err := m.checkError("WithTransaction"); err != nil {
		return err
	}

	// For mock, we don't simulate actual transactions, just call the function
	// In a real test scenario, you might want to implement rollback simulation
	return fn(m)
}

// Factory methods for creating test data

// CreateTestTask creates a test task with common defaults
func (m *EnhancedMockNudgeRepository) CreateTestTask(userID, title string) *nudge.Task {
	now := time.Now()
	dueDate := now.Add(24 * time.Hour)

	return &nudge.Task{
		ID:          common.TaskID(common.NewID()),
		UserID:      common.UserID(userID),
		Title:       title,
		Description: "Test description",
		DueDate:     &dueDate,
		Priority:    common.PriorityMedium,
		Status:      common.TaskStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// createTestTask is the internal version that doesn't lock
func (m *EnhancedMockNudgeRepository) createTestTask(userID, title string) *nudge.Task {
	now := time.Now()
	dueDate := now.Add(24 * time.Hour)

	return &nudge.Task{
		ID:          common.TaskID(common.NewID()),
		UserID:      common.UserID(userID),
		Title:       title,
		Description: "Test description",
		DueDate:     &dueDate,
		Priority:    common.PriorityMedium,
		Status:      common.TaskStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// CreateOverdueTask creates an overdue test task
func (m *EnhancedMockNudgeRepository) CreateOverdueTask(userID string) *nudge.Task {
	now := time.Now()
	dueDate := now.Add(-2 * time.Hour) // 2 hours ago

	return &nudge.Task{
		ID:          common.TaskID(common.NewID()),
		UserID:      common.UserID(userID),
		Title:       "Overdue Task",
		Description: "This task is overdue",
		DueDate:     &dueDate,
		Priority:    common.PriorityHigh,
		Status:      common.TaskStatusActive,
		CreatedAt:   now.Add(-48 * time.Hour),
		UpdatedAt:   now.Add(-48 * time.Hour),
	}
}

// createOverdueTask is the internal version that doesn't lock
func (m *EnhancedMockNudgeRepository) createOverdueTask(userID string) *nudge.Task {
	now := time.Now()
	dueDate := now.Add(-2 * time.Hour) // 2 hours ago

	return &nudge.Task{
		ID:          common.TaskID(common.NewID()),
		UserID:      common.UserID(userID),
		Title:       "Overdue Task",
		Description: "This task is overdue",
		DueDate:     &dueDate,
		Priority:    common.PriorityHigh,
		Status:      common.TaskStatusActive,
		CreatedAt:   now.Add(-48 * time.Hour),
		UpdatedAt:   now.Add(-48 * time.Hour),
	}
}

// CreateCompletedTask creates a completed test task
func (m *EnhancedMockNudgeRepository) CreateCompletedTask(userID string) *nudge.Task {
	now := time.Now()
	completedAt := now.Add(-1 * time.Hour)

	return &nudge.Task{
		ID:          common.TaskID(common.NewID()),
		UserID:      common.UserID(userID),
		Title:       "Completed Task",
		Description: "This task is completed",
		DueDate:     &completedAt,
		Priority:    common.PriorityLow,
		Status:      common.TaskStatusCompleted,
		CreatedAt:   now.Add(-24 * time.Hour),
		UpdatedAt:   completedAt,
		CompletedAt: &completedAt,
	}
}

// createCompletedTask is the internal version that doesn't lock
func (m *EnhancedMockNudgeRepository) createCompletedTask(userID string) *nudge.Task {
	now := time.Now()
	completedAt := now.Add(-1 * time.Hour)

	return &nudge.Task{
		ID:          common.TaskID(common.NewID()),
		UserID:      common.UserID(userID),
		Title:       "Completed Task",
		Description: "This task is completed",
		DueDate:     &completedAt,
		Priority:    common.PriorityLow,
		Status:      common.TaskStatusCompleted,
		CreatedAt:   now.Add(-24 * time.Hour),
		UpdatedAt:   completedAt,
		CompletedAt: &completedAt,
	}
}

// createTestReminder creates a test reminder
func (m *EnhancedMockNudgeRepository) createTestReminder(taskID common.TaskID, userID common.UserID) *nudge.Reminder {
	now := time.Now()

	return &nudge.Reminder{
		ID:           common.ID(common.NewID()),
		TaskID:       taskID,
		UserID:       userID,
		ScheduledAt:  now.Add(30 * time.Minute),
		ReminderType: nudge.ReminderTypeInitial,
	}
}

// Mock business logic components

// MockTaskValidator provides mock validation for testing
type MockTaskValidator struct {
	shouldFail   bool
	failureError error
}

// NewMockTaskValidator creates a new mock validator
func NewMockTaskValidator() *MockTaskValidator {
	return &MockTaskValidator{}
}

// SetShouldFail configures the validator to fail
func (v *MockTaskValidator) SetShouldFail(shouldFail bool, err error) {
	v.shouldFail = shouldFail
	v.failureError = err
}

// ValidateTask validates a task (mock implementation)
func (v *MockTaskValidator) ValidateTask(task *nudge.Task) error {
	if v.shouldFail {
		return v.failureError
	}
	return nil
}

// ValidateStatusTransition validates status transitions (mock implementation)
func (v *MockTaskValidator) ValidateStatusTransition(from, to common.TaskStatus) error {
	if v.shouldFail {
		return v.failureError
	}
	return nil
}

// ValidateTaskFilter validates task filters (mock implementation)
func (v *MockTaskValidator) ValidateTaskFilter(filter nudge.TaskFilter) error {
	if v.shouldFail {
		return v.failureError
	}
	return nil
}

// MockReminderManager provides mock reminder management for testing
type MockReminderManager struct {
	reminderTime  time.Time
	shouldNudge   bool
	nextNudgeTime time.Time
}

// NewMockReminderManager creates a new mock reminder manager
func NewMockReminderManager() *MockReminderManager {
	return &MockReminderManager{
		reminderTime:  time.Now().Add(time.Hour),
		shouldNudge:   true,
		nextNudgeTime: time.Now().Add(2 * time.Hour),
	}
}

// CalculateReminderTime calculates reminder time (mock implementation)
func (rm *MockReminderManager) CalculateReminderTime(task *nudge.Task, settings *nudge.NudgeSettings) time.Time {
	return rm.reminderTime
}

// ShouldCreateNudge determines if a nudge should be created (mock implementation)
func (rm *MockReminderManager) ShouldCreateNudge(task *nudge.Task, reminderCount int, settings *nudge.NudgeSettings) bool {
	return rm.shouldNudge
}

// GetNextNudgeTime calculates next nudge time (mock implementation)
func (rm *MockReminderManager) GetNextNudgeTime(lastNudge time.Time, settings *nudge.NudgeSettings) time.Time {
	return rm.nextNudgeTime
}

// MockStatusManager provides mock status management for testing
type MockStatusManager struct {
	shouldFail   bool
	failureError error
}

// NewMockStatusManager creates a new mock status manager
func NewMockStatusManager() *MockStatusManager {
	return &MockStatusManager{}
}

// SetShouldFail configures the manager to fail
func (sm *MockStatusManager) SetShouldFail(shouldFail bool, err error) {
	sm.shouldFail = shouldFail
	sm.failureError = err
}

// TransitionStatus handles status transitions (mock implementation)
func (sm *MockStatusManager) TransitionStatus(task *nudge.Task, newStatus common.TaskStatus) error {
	if sm.shouldFail {
		return sm.failureError
	}

	task.Status = newStatus
	task.UpdatedAt = time.Now()

	if newStatus == common.TaskStatusCompleted {
		now := time.Now()
		task.CompletedAt = &now
	}

	return nil
}

// CompleteTask completes a task (mock implementation)
func (sm *MockStatusManager) CompleteTask(task *nudge.Task) error {
	if sm.shouldFail {
		return sm.failureError
	}

	now := time.Now()
	task.Status = common.TaskStatusCompleted
	task.CompletedAt = &now
	task.UpdatedAt = now

	return nil
}

// SnoozeTask snoozes a task (mock implementation)
func (sm *MockStatusManager) SnoozeTask(task *nudge.Task, snoozeUntil time.Time) error {
	if sm.shouldFail {
		return sm.failureError
	}

	task.Status = common.TaskStatusSnoozed
	task.DueDate = &snoozeUntil
	task.UpdatedAt = time.Now()

	return nil
}

// DeleteTask deletes a task (mock implementation)
func (sm *MockStatusManager) DeleteTask(task *nudge.Task) error {
	if sm.shouldFail {
		return sm.failureError
	}

	task.Status = common.TaskStatusDeleted
	task.UpdatedAt = time.Now()

	return nil
}
