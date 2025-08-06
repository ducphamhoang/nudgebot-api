//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"nudgebot-api/api/handlers"
	"nudgebot-api/internal/chatbot"
	"nudgebot-api/internal/common"
	"nudgebot-api/internal/config"
	"nudgebot-api/internal/events"
	"nudgebot-api/internal/mocks"
	"nudgebot-api/internal/nudge"
	"nudgebot-api/pkg/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestCallbackFlow_TaskDoneButton(t *testing.T) {
	// Setup test environment
	testContainer, cleanup := SetupTestDatabase(t)
	defer cleanup()

	zapLogger := zap.NewNop()
	eventBus := events.NewEventBus(zapLogger)
	nudgeRepo := nudge.NewGormNudgeRepository(testContainer.DB, zapLogger)
	nudgeService, err := nudge.NewNudgeService(eventBus, zapLogger, nudgeRepo)
	require.NoError(t, err)

	mockTelegramProvider := mocks.NewMockTelegramProvider()

	chatbotService, err := chatbot.NewChatbotServiceWithProvider(
		eventBus,
		zapLogger,
		mockTelegramProvider,
		config.ChatbotConfig{Token: "test-token", WebhookURL: "test-url", Timeout: 30},
	)
	require.NoError(t, err)

	// Setup HTTP handler
	gin.SetMode(gin.TestMode)
	router := gin.New()
	loggerInstance := logger.New()
	webhookHandler := handlers.NewWebhookHandler(chatbotService, loggerInstance)
	router.POST("/webhook", webhookHandler.HandleTelegramWebhook)

	// Create test task with reminder
	userID := common.UserID("123456")
	dueTime := time.Now().Add(-30 * time.Minute) // Overdue task

	task := &nudge.Task{
		UserID:      userID,
		Description: "Task with reminder",
		DueDate:     &dueTime,
		Status:      common.TaskStatusActive,
		Priority:    common.PriorityMedium,
	}

	err = nudgeService.CreateTask(task)
	require.NoError(t, err)

	// Get created task to get its ID
	tasks, err := nudgeService.GetTasks(userID, nudge.TaskFilter{})
	require.NoError(t, err)
	require.NotEmpty(t, tasks)
	createdTask := tasks[0]

	// Create callback query for 'Done' button
	callbackData := createTaskActionCallbackData("done", string(createdTask.ID))
	callbackQuery := createTestCallbackQuery(123456, callbackData)

	// Make HTTP request
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/webhook", bytes.NewBuffer(callbackQuery))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	// Verify HTTP response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify task was completed in database
	updatedTask, err := nudgeRepo.GetTaskByID(createdTask.ID)
	require.NoError(t, err)
	assert.Equal(t, common.TaskStatusCompleted, updatedTask.Status)
	assert.NotNil(t, updatedTask.CompletedAt)

	// Verify callback answer was sent
	assert.True(t, mockTelegramProvider.GetCallCount("AnswerCallbackQuery") > 0)
}

func TestCallbackFlow_TaskDeleteButton(t *testing.T) {
	// Setup test environment
	testContainer, cleanup := SetupTestDatabase(t)
	defer cleanup()

	zapLogger := zap.NewNop()
	eventBus := events.NewEventBus(zapLogger)
	nudgeRepo := nudge.NewGormNudgeRepository(testContainer.DB, zapLogger)
	nudgeService, err := nudge.NewNudgeService(eventBus, zapLogger, nudgeRepo)
	require.NoError(t, err)

	mockTelegramProvider := mocks.NewMockTelegramProvider()

	chatbotService, err := chatbot.NewChatbotServiceWithProvider(
		eventBus,
		zapLogger,
		mockTelegramProvider,
		config.ChatbotConfig{Token: "test-token", WebhookURL: "test-url", Timeout: 30},
	)
	require.NoError(t, err)

	// Setup HTTP handler
	gin.SetMode(gin.TestMode)
	router := gin.New()
	loggerInstance := logger.New()
	webhookHandler := handlers.NewWebhookHandler(chatbotService, loggerInstance)
	router.POST("/webhook", webhookHandler.HandleTelegramWebhook)

	// Create test task
	userID := common.UserID("123456")
	dueTime := time.Now().Add(24 * time.Hour)

	task := &nudge.Task{
		UserID:      userID,
		Description: "Task to be deleted",
		DueDate:     &dueTime,
		Status:      common.TaskStatusActive,
		Priority:    common.PriorityLow,
	}

	err = nudgeService.CreateTask(task)
	require.NoError(t, err)

	// Get created task to get its ID
	tasks, err := nudgeService.GetTasks(userID, nudge.TaskFilter{})
	require.NoError(t, err)
	require.NotEmpty(t, tasks)
	createdTask := tasks[0]

	// Create callback query for 'Delete' button
	callbackData := createTaskActionCallbackData("delete", string(createdTask.ID))
	callbackQuery := createTestCallbackQuery(123456, callbackData)

	// Make HTTP request
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/webhook", bytes.NewBuffer(callbackQuery))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	// Verify HTTP response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify task was deleted from database
	remainingTasks, err := nudgeService.GetTasks(userID, nudge.TaskFilter{})
	require.NoError(t, err)
	assert.Empty(t, remainingTasks, "Task should be deleted")

	// Verify callback answer was sent
	assert.True(t, mockTelegramProvider.GetCallCount("AnswerCallbackQuery") > 0)
}

func TestCallbackFlow_SnoozeButton(t *testing.T) {
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

	// Setup HTTP handler
	gin.SetMode(gin.TestMode)
	router := gin.New()
	webhookHandler := handlers.NewWebhookHandler(chatbotService, zap.NewNop())
	router.POST("/webhook", webhookHandler.HandleWebhook)

	// Start services
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		eventBus.Start(ctx)
	}()

	err = chatbotService.Start(ctx)
	require.NoError(t, err)

	// Create overdue test task
	userID := common.UserID("123456")
	dueTime := time.Now().Add(-10 * time.Minute)

	task := &nudge.Task{
		UserID:      userID,
		Description: "Task to be snoozed",
		DueDate:     &dueTime,
		Status:      common.TaskStatusActive,
		Priority:    common.TaskPriorityMedium,
	}

	err = nudgeService.CreateTask(task)
	require.NoError(t, err)

	// Get created task to get its ID
	tasks, err := nudgeService.GetTasks(userID, nudge.TaskFilter{})
	require.NoError(t, err)
	require.NotEmpty(t, tasks)
	createdTask := tasks[0]

	// Create callback query for 'Snooze 1 hour' button
	callbackData := createSnoozeCallbackData(string(createdTask.ID), "1h")
	callbackQuery := createTestCallbackQuery(123456, callbackData)

	// Make HTTP request
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/webhook", bytes.NewBuffer(callbackQuery))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	// Verify HTTP response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify task was snoozed in database
	updatedTask, err := nudgeService.GetTaskByID(createdTask.ID)
	require.NoError(t, err)
	assert.Equal(t, common.TaskStatusSnoozed, updatedTask.Status)

	// Verify due date was updated (approximately 1 hour from now)
	expectedSnoozeTime := time.Now().Add(1 * time.Hour)
	timeDiff := updatedTask.DueDate.Sub(expectedSnoozeTime)
	assert.Less(t, timeDiff, 2*time.Minute, "Snooze time should be approximately 1 hour from now")

	// Verify callback answer was sent
	assert.True(t, mockTelegramProvider.GetCallCount("AnswerCallbackQuery") > 0)
}

func TestCallbackFlow_TaskListNavigation(t *testing.T) {
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

	// Setup HTTP handler
	gin.SetMode(gin.TestMode)
	router := gin.New()
	webhookHandler := handlers.NewWebhookHandler(chatbotService, zap.NewNop())
	router.POST("/webhook", webhookHandler.HandleWebhook)

	// Start services
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		eventBus.Start(ctx)
	}()

	err = chatbotService.Start(ctx)
	require.NoError(t, err)

	// Create multiple test tasks
	userID := common.UserID("123456")
	dueTime := time.Now().Add(24 * time.Hour)

	for i := 1; i <= 10; i++ {
		task := &nudge.Task{
			UserID:      userID,
			Description: fmt.Sprintf("Task %d", i),
			DueDate:     &dueTime,
			Status:      common.TaskStatusActive,
			Priority:    common.TaskPriorityMedium,
		}

		err = nudgeService.CreateTask(task)
		require.NoError(t, err)
	}

	// Create callback query for 'next page' button
	callbackData := createPaginationCallbackData("next", 1)
	callbackQuery := createTestCallbackQuery(123456, callbackData)

	// Make HTTP request
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/webhook", bytes.NewBuffer(callbackQuery))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	// Verify HTTP response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify callback answer was sent
	assert.True(t, mockTelegramProvider.GetCallCount("AnswerCallbackQuery") > 0)

	// Verify message was updated with new page content
	sentMessages := mockTelegramProvider.GetSentMessages()
	assert.NotEmpty(t, sentMessages, "Expected updated task list message")
}

func TestCallbackFlow_InvalidCallbackData(t *testing.T) {
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

	// Setup HTTP handler
	gin.SetMode(gin.TestMode)
	router := gin.New()
	webhookHandler := handlers.NewWebhookHandler(chatbotService, zap.NewNop())
	router.POST("/webhook", webhookHandler.HandleWebhook)

	// Start services
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		eventBus.Start(ctx)
	}()

	err = chatbotService.Start(ctx)
	require.NoError(t, err)

	// Create callback query with malformed data
	callbackQuery := createTestCallbackQuery(123456, "invalid_json_data")

	// Make HTTP request
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/webhook", bytes.NewBuffer(callbackQuery))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	// Verify HTTP response (should still be 200 for Telegram)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify error callback answer was sent
	assert.True(t, mockTelegramProvider.GetCallCount("AnswerCallbackQuery") > 0)
}

func TestCallbackFlow_CallbackForNonExistentTask(t *testing.T) {
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

	// Setup HTTP handler
	gin.SetMode(gin.TestMode)
	router := gin.New()
	webhookHandler := handlers.NewWebhookHandler(chatbotService, zap.NewNop())
	router.POST("/webhook", webhookHandler.HandleWebhook)

	// Start services
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		eventBus.Start(ctx)
	}()

	err = chatbotService.Start(ctx)
	require.NoError(t, err)

	// Create callback query for non-existent task
	callbackData := createTaskActionCallbackData("done", "non_existent_task_id")
	callbackQuery := createTestCallbackQuery(123456, callbackData)

	// Make HTTP request
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/webhook", bytes.NewBuffer(callbackQuery))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	// Verify HTTP response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify error callback answer was sent
	assert.True(t, mockTelegramProvider.GetCallCount("AnswerCallbackQuery") > 0)
}

// Helper functions for creating callback data and queries

func createTaskActionCallbackData(action, taskID string) string {
	data := map[string]interface{}{
		"action":  action,
		"task_id": taskID,
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func createSnoozeCallbackData(taskID, duration string) string {
	data := map[string]interface{}{
		"action":   "snooze",
		"task_id":  taskID,
		"duration": duration,
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func createPaginationCallbackData(direction string, page int) string {
	data := map[string]interface{}{
		"action": direction,
		"page":   page,
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func createTestCallbackQuery(chatID int64, callbackData string) []byte {
	update := tgbotapi.Update{
		UpdateID: 123,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID: "callback_123",
			From: &tgbotapi.User{
				ID:        chatID,
				UserName:  "testuser",
				FirstName: "Test",
				LastName:  "User",
			},
			Message: &tgbotapi.Message{
				MessageID: 456,
				Chat: &tgbotapi.Chat{
					ID:   chatID,
					Type: "private",
				},
				Date: int(time.Now().Unix()),
			},
			Data: callbackData,
		},
	}

	data, _ := json.Marshal(update)
	return data
}
