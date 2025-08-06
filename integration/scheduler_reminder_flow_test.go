//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"nudgebot-api/internal/chatbot"
	"nudgebot-api/internal/common"
	"nudgebot-api/internal/config"
	"nudgebot-api/internal/events"
	"nudgebot-api/internal/mocks"
	"nudgebot-api/internal/nudge"
	"nudgebot-api/internal/scheduler"
)

func TestSchedulerReminderFlow_DueTaskReminder(t *testing.T) {
	// Setup test environment
	testDB := SetupTestDatabase(t)
	defer testDB.TeardownTestDatabase(t)

	// Initialize mock clock for deterministic timing
	now := time.Now()
	mockClock := common.NewMockClock(now)

	// Create synchronous event bus for deterministic event ordering
	eventBus := events.NewEventBus()

	// Initialize services with real implementations
	nudgeRepo := nudge.NewGormRepository(testDB.DB)
	nudgeService, err := nudge.NewNudgeService(eventBus, zap.NewNop(), nudgeRepo)
	require.NoError(t, err)

	// Create stub providers
	mockTelegramProvider := mocks.NewMockTelegramProvider()
	mockLLMProvider := &mocks.LLMProviderMock{}

	chatbotService, err := chatbot.NewService(
		mockTelegramProvider,
		mockLLMProvider,
		nudgeService,
		eventBus,
		zap.NewNop(),
	)
	require.NoError(t, err)

	// Create scheduler with short poll interval for testing
	schedulerConfig := config.SchedulerConfig{
		PollInterval:    1, // 1 second for fast testing
		NudgeDelay:      60,
		WorkerCount:     1,
		ShutdownTimeout: 5,
		Enabled:         true,
	}

	schedulerService, err := scheduler.NewScheduler(schedulerConfig, nudgeRepo, eventBus, zap.NewNop())
	require.NoError(t, err)

	// Start services
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		eventBus.Start(ctx)
	}()

	// Subscribe chatbot to events
	err = chatbotService.Start(ctx)
	require.NoError(t, err)

	// Create a task that is due 2 minutes ago (overdue)
	userID := common.UserID("test_user_123")
	dueTime := now.Add(-2 * time.Minute)

	task := &nudge.Task{
		UserID:      userID,
		Description: "Test reminder task",
		DueDate:     &dueTime,
		Status:      common.TaskStatusActive,
		Priority:    common.TaskPriorityMedium,
	}

	err = nudgeService.CreateTask(task)
	require.NoError(t, err)

	// Clear any previous mock calls
	mockTelegramProvider.ClearHistory()

	// Start scheduler and let it run for a short time
	go func() {
		err := schedulerService.Start(ctx)
		if err != nil {
			t.Logf("Scheduler stopped with error: %v", err)
		}
	}()

	// Wait for scheduler to process reminders
	time.Sleep(3 * time.Second)

	// Stop scheduler
	err = schedulerService.Stop()
	assert.NoError(t, err)

	// Verify reminder was sent via Telegram
	sentMessages := mockTelegramProvider.GetSentMessages()
	assert.NotEmpty(t, sentMessages, "Expected at least one message to be sent")

	if len(sentMessages) > 0 {
		lastMessage := sentMessages[len(sentMessages)-1]
		assert.Equal(t, int64(userID), lastMessage.ChatID)
		assert.Contains(t, lastMessage.Text, "Test reminder task")
	}

	// Verify inline keyboard was included with action buttons
	sentKeyboards := mockTelegramProvider.GetSentKeyboards()
	assert.NotEmpty(t, sentKeyboards, "Expected keyboard message to be sent")
}

func TestSchedulerReminderFlow_NudgeAfterOverdue(t *testing.T) {
	// Setup test environment
	testDB := SetupTestDatabase(t)
	defer testDB.TeardownTestDatabase(t)

	// Initialize event bus and services
	eventBus := events.NewEventBus()
	nudgeRepo := nudge.NewGormRepository(testDB.DB)
	nudgeService, err := nudge.NewNudgeService(eventBus, zap.NewNop(), nudgeRepo)
	require.NoError(t, err)

	mockTelegramProvider := mocks.NewMockTelegramProvider()
	mockLLMProvider := &mocks.LLMProviderMock{}

	chatbotService, err := chatbot.NewService(
		mockTelegramProvider,
		mockLLMProvider,
		nudgeService,
		eventBus,
		zap.NewNop(),
	)
	require.NoError(t, err)

	schedulerConfig := config.SchedulerConfig{
		PollInterval:    1,
		NudgeDelay:      60, // 1 minute for testing
		WorkerCount:     1,
		ShutdownTimeout: 5,
		Enabled:         true,
	}

	schedulerService, err := scheduler.NewScheduler(schedulerConfig, nudgeRepo, eventBus, zap.NewNop())
	require.NoError(t, err)

	// Start services
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		eventBus.Start(ctx)
	}()

	err = chatbotService.Start(ctx)
	require.NoError(t, err)

	// Create a task that is overdue by more than the nudge delay (2 minutes ago)
	userID := common.UserID("test_user_123")
	dueTime := time.Now().Add(-2 * time.Minute)

	task := &nudge.Task{
		UserID:      userID,
		Description: "Overdue task for nudging",
		DueDate:     &dueTime,
		Status:      common.TaskStatusActive,
		Priority:    common.TaskPriorityHigh,
	}

	err = nudgeService.CreateTask(task)
	require.NoError(t, err)

	// Clear mock calls
	mockTelegramProvider.ClearHistory()

	// Start scheduler
	go func() {
		err := schedulerService.Start(ctx)
		if err != nil {
			t.Logf("Scheduler stopped with error: %v", err)
		}
	}()

	// Wait for scheduler to process
	time.Sleep(3 * time.Second)

	// Stop scheduler
	err = schedulerService.Stop()
	assert.NoError(t, err)

	// Verify nudge was sent
	sentMessages := mockTelegramProvider.GetSentMessages()
	assert.NotEmpty(t, sentMessages, "Expected nudge message to be sent")

	if len(sentMessages) > 0 {
		lastMessage := sentMessages[len(sentMessages)-1]
		assert.Equal(t, int64(userID), lastMessage.ChatID)
		assert.Contains(t, lastMessage.Text, "Overdue task for nudging")
	}
}

func TestSchedulerReminderFlow_UserActionAfterReminder(t *testing.T) {
	// Setup test environment
	testDB := SetupTestDatabase(t)
	defer testDB.TeardownTestDatabase(t)

	eventBus := events.NewEventBus()
	nudgeRepo := nudge.NewGormRepository(testDB.DB)
	nudgeService, err := nudge.NewNudgeService(eventBus, zap.NewNop(), nudgeRepo)
	require.NoError(t, err)

	mockTelegramProvider := mocks.NewMockTelegramProvider()
	mockLLMProvider := &mocks.LLMProviderMock{}

	chatbotService, err := chatbot.NewService(
		mockTelegramProvider,
		mockLLMProvider,
		nudgeService,
		eventBus,
		zap.NewNop(),
	)
	require.NoError(t, err)

	schedulerConfig := config.SchedulerConfig{
		PollInterval:    1,
		NudgeDelay:      60,
		WorkerCount:     1,
		ShutdownTimeout: 5,
		Enabled:         true,
	}

	schedulerService, err := scheduler.NewScheduler(schedulerConfig, nudgeRepo, eventBus, zap.NewNop())
	require.NoError(t, err)

	// Start services
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	go func() {
		eventBus.Start(ctx)
	}()

	err = chatbotService.Start(ctx)
	require.NoError(t, err)

	// Create overdue task
	userID := common.UserID("test_user_123")
	dueTime := time.Now().Add(-30 * time.Minute)

	task := &nudge.Task{
		UserID:      userID,
		Description: "Task to be completed",
		DueDate:     &dueTime,
		Status:      common.TaskStatusActive,
		Priority:    common.TaskPriorityMedium,
	}

	err = nudgeService.CreateTask(task)
	require.NoError(t, err)

	// Start scheduler to send reminder
	go func() {
		err := schedulerService.Start(ctx)
		if err != nil {
			t.Logf("Scheduler stopped with error: %v", err)
		}
	}()

	// Wait for initial reminder
	time.Sleep(3 * time.Second)

	// Verify reminder was sent
	sentMessages := mockTelegramProvider.GetSentMessages()
	assert.NotEmpty(t, sentMessages, "Expected initial reminder to be sent")

	// Simulate user completing the task
	tasks, err := nudgeService.GetTasks(userID, nudge.TaskFilter{})
	require.NoError(t, err)
	require.NotEmpty(t, tasks)

	err = nudgeService.UpdateTaskStatus(tasks[0].ID, common.TaskStatusCompleted)
	require.NoError(t, err)

	// Clear message history and wait to see if more reminders are sent
	mockTelegramProvider.ClearHistory()
	time.Sleep(3 * time.Second)

	// Stop scheduler
	err = schedulerService.Stop()
	assert.NoError(t, err)

	// Verify no additional reminders were sent for completed task
	newMessages := mockTelegramProvider.GetSentMessages()
	assert.Len(t, newMessages, 0, "Should not send reminders for completed tasks")
}

func TestSchedulerReminderFlow_MultipleTasksAndUsers(t *testing.T) {
	// Setup test environment
	testDB := SetupTestDatabase(t)
	defer testDB.TeardownTestDatabase(t)

	eventBus := events.NewEventBus()
	nudgeRepo := nudge.NewGormRepository(testDB.DB)
	nudgeService, err := nudge.NewNudgeService(eventBus, zap.NewNop(), nudgeRepo)
	require.NoError(t, err)

	mockTelegramProvider := mocks.NewMockTelegramProvider()
	mockLLMProvider := &mocks.LLMProviderMock{}

	chatbotService, err := chatbot.NewService(
		mockTelegramProvider,
		mockLLMProvider,
		nudgeService,
		eventBus,
		zap.NewNop(),
	)
	require.NoError(t, err)

	schedulerConfig := config.SchedulerConfig{
		PollInterval:    1,
		NudgeDelay:      60,
		WorkerCount:     2, // Multiple workers for concurrent processing
		ShutdownTimeout: 5,
		Enabled:         true,
	}

	schedulerService, err := scheduler.NewScheduler(schedulerConfig, nudgeRepo, eventBus, zap.NewNop())
	require.NoError(t, err)

	// Start services
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		eventBus.Start(ctx)
	}()

	err = chatbotService.Start(ctx)
	require.NoError(t, err)

	// Create tasks for multiple users
	user1 := common.UserID("user_1")
	user2 := common.UserID("user_2")
	dueTime := time.Now().Add(-10 * time.Minute)

	// User 1 tasks
	task1 := &nudge.Task{
		UserID:      user1,
		Description: "User 1 Task 1",
		DueDate:     &dueTime,
		Status:      common.TaskStatusActive,
		Priority:    common.TaskPriorityHigh,
	}

	task2 := &nudge.Task{
		UserID:      user1,
		Description: "User 1 Task 2",
		DueDate:     &dueTime,
		Status:      common.TaskStatusActive,
		Priority:    common.TaskPriorityMedium,
	}

	// User 2 task
	task3 := &nudge.Task{
		UserID:      user2,
		Description: "User 2 Task 1",
		DueDate:     &dueTime,
		Status:      common.TaskStatusActive,
		Priority:    common.TaskPriorityLow,
	}

	err = nudgeService.CreateTask(task1)
	require.NoError(t, err)

	err = nudgeService.CreateTask(task2)
	require.NoError(t, err)

	err = nudgeService.CreateTask(task3)
	require.NoError(t, err)

	// Clear mock calls and start scheduler
	mockTelegramProvider.ClearHistory()

	go func() {
		err := schedulerService.Start(ctx)
		if err != nil {
			t.Logf("Scheduler stopped with error: %v", err)
		}
	}()

	// Wait for processing
	time.Sleep(5 * time.Second)

	// Stop scheduler
	err = schedulerService.Stop()
	assert.NoError(t, err)

	// Verify messages were sent
	sentMessages := mockTelegramProvider.GetSentMessages()
	assert.NotEmpty(t, sentMessages, "Expected messages to be sent")

	// Verify both users received messages
	user1Messages := mockTelegramProvider.GetMessagesForChat(int64(user1))
	user2Messages := mockTelegramProvider.GetMessagesForChat(int64(user2))

	assert.NotEmpty(t, user1Messages, "User 1 should receive reminder messages")
	assert.NotEmpty(t, user2Messages, "User 2 should receive reminder messages")
}
