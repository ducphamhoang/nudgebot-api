package mocks

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"nudgebot-api/internal/nudge"
	"nudgebot-api/internal/scheduler"
)

//go:generate mockgen -source=../scheduler/scheduler.go -destination=scheduler_mocks.go -package=mocks

// MockScheduler implements the Scheduler interface for testing
type MockScheduler struct {
	started    atomic.Bool
	startError error
	stopError  error
	callCounts map[string]int
	metrics    *scheduler.SchedulerMetrics
	mu         sync.RWMutex
}

// NewMockScheduler creates a new mock scheduler
func NewMockScheduler() *MockScheduler {
	return &MockScheduler{
		callCounts: make(map[string]int),
		metrics:    scheduler.NewSchedulerMetrics(),
	}
}

// Start implements the Scheduler interface
func (m *MockScheduler) Start(ctx context.Context) error {
	m.incrementCallCount("Start")
	if m.startError != nil {
		return m.startError
	}
	m.started.Store(true)
	return nil
}

// Stop implements the Scheduler interface
func (m *MockScheduler) Stop() error {
	m.incrementCallCount("Stop")
	if m.stopError != nil {
		return m.stopError
	}
	m.started.Store(false)
	return nil
}

// IsRunning implements the Scheduler interface
func (m *MockScheduler) IsRunning() bool {
	m.incrementCallCount("IsRunning")
	return m.started.Load()
}

// GetMetrics implements the Scheduler interface
func (m *MockScheduler) GetMetrics() *scheduler.SchedulerMetrics {
	m.incrementCallCount("GetMetrics")
	return m.metrics
}

// Test configuration methods
func (m *MockScheduler) SetStartError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startError = err
}

func (m *MockScheduler) SetStopError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopError = err
}

func (m *MockScheduler) GetCallCount(method string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCounts[method]
}

func (m *MockScheduler) incrementCallCount(method string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCounts[method]++
}

// MockReminderProcessor simulates reminder processing for testing
type MockReminderProcessor struct {
	reminders       []*nudge.Reminder
	processingDelay time.Duration
	processedCount  int
	errorScenario   bool
	mu              sync.RWMutex
}

// NewMockReminderProcessor creates a new mock reminder processor
func NewMockReminderProcessor() *MockReminderProcessor {
	return &MockReminderProcessor{
		reminders: make([]*nudge.Reminder, 0),
	}
}

// AddDueReminder adds a reminder for testing
func (m *MockReminderProcessor) AddDueReminder(reminder *nudge.Reminder) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reminders = append(m.reminders, reminder)
}

// SimulateProcessingDelay sets a delay for processing simulation
func (m *MockReminderProcessor) SimulateProcessingDelay(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processingDelay = duration
}

// SetErrorScenario enables error simulation
func (m *MockReminderProcessor) SetErrorScenario(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorScenario = enabled
}

// ProcessReminders simulates reminder processing
func (m *MockReminderProcessor) ProcessReminders() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.errorScenario {
		return scheduler.NewSchedulerError("test_error", "simulated error")
	}

	if m.processingDelay > 0 {
		time.Sleep(m.processingDelay)
	}

	m.processedCount += len(m.reminders)
	m.reminders = m.reminders[:0] // Clear processed reminders

	return nil
}

// GetProcessedCount returns the number of processed reminders
func (m *MockReminderProcessor) GetProcessedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.processedCount
}

// Factory methods for test scenarios

// CreateTestScheduler creates a basic scheduler for testing
func CreateTestScheduler() scheduler.Scheduler {
	return NewMockScheduler()
}

// CreateFailingScheduler creates a scheduler that fails on start
func CreateFailingScheduler() scheduler.Scheduler {
	mock := NewMockScheduler()
	mock.SetStartError(scheduler.NewSchedulerError("start_failed", "mock start failure"))
	return mock
}

// CreateSlowScheduler creates a scheduler that simulates slow operations
func CreateSlowScheduler() *MockScheduler {
	mock := NewMockScheduler()
	// Simulate slow operations by adding delays in test
	return mock
}

// Test assertion helpers

// AssertStarted verifies the scheduler is started
func (m *MockScheduler) AssertStarted(t TestingT) {
	if !m.IsRunning() {
		t.Errorf("Expected scheduler to be started, but it was not")
	}
}

// AssertStopped verifies the scheduler is stopped
func (m *MockScheduler) AssertStopped(t TestingT) {
	if m.IsRunning() {
		t.Errorf("Expected scheduler to be stopped, but it was running")
	}
}

// AssertProcessedCount verifies the expected number of reminders were processed
func (m *MockReminderProcessor) AssertProcessedCount(t TestingT, expected int) {
	actual := m.GetProcessedCount()
	if actual != expected {
		t.Errorf("Expected %d processed reminders, got %d", expected, actual)
	}
}

// TestingT is a minimal interface for testing frameworks
type TestingT interface {
	Errorf(format string, args ...interface{})
}
