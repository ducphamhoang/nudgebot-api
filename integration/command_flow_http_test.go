//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
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
	"nudgebot-api/internal/events"
	"nudgebot-api/internal/mocks"
	"nudgebot-api/internal/nudge"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestCommandFlow_StartCommand(t *testing.T) {
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

	// Create /start command webhook payload
	startCommand := createTestTelegramWebhook(123456, "/start", "")

	// Make HTTP request
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/webhook", bytes.NewBuffer(startCommand))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	// Verify HTTP response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify welcome message was sent
	sentMessages := mockTelegramProvider.GetSentMessages()
	assert.NotEmpty(t, sentMessages, "Expected welcome message to be sent")

	if len(sentMessages) > 0 {
		welcomeMessage := sentMessages[0]
		assert.Equal(t, int64(123456), welcomeMessage.ChatID)
		assert.Contains(t, welcomeMessage.Text, "welcome")
	}
}

func TestCommandFlow_ListCommand(t *testing.T) {
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

	// Create test tasks in database
	userID := common.UserID("123456")
	dueTime := time.Now().Add(24 * time.Hour)

	task1 := &nudge.Task{
		UserID:      userID,
		Description: "Test task 1",
		DueDate:     &dueTime,
		Status:      common.TaskStatusActive,
		Priority:    common.TaskPriorityHigh,
	}

	task2 := &nudge.Task{
		UserID:      userID,
		Description: "Test task 2",
		DueDate:     &dueTime,
		Status:      common.TaskStatusActive,
		Priority:    common.TaskPriorityMedium,
	}

	err = nudgeService.CreateTask(task1)
	require.NoError(t, err)

	err = nudgeService.CreateTask(task2)
	require.NoError(t, err)

	// Create /list command webhook payload
	listCommand := createTestTelegramWebhook(123456, "/list", "")

	// Make HTTP request
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/webhook", bytes.NewBuffer(listCommand))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	// Verify HTTP response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify task list message was sent
	sentMessages := mockTelegramProvider.GetSentMessages()
	assert.NotEmpty(t, sentMessages, "Expected task list message to be sent")

	if len(sentMessages) > 0 {
		listMessage := sentMessages[len(sentMessages)-1]
		assert.Equal(t, int64(123456), listMessage.ChatID)
		assert.Contains(t, listMessage.Text, "Test task 1")
		assert.Contains(t, listMessage.Text, "Test task 2")
	}
}

func TestCommandFlow_DoneCommand(t *testing.T) {
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

	// Create test task in database
	userID := common.UserID("123456")
	dueTime := time.Now().Add(24 * time.Hour)

	task := &nudge.Task{
		UserID:      userID,
		Description: "Task to be completed",
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

	// Create /done command webhook payload with task ID
	doneCommand := createTestTelegramWebhook(123456, "/done", string(createdTask.ID))

	// Make HTTP request
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/webhook", bytes.NewBuffer(doneCommand))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	// Verify HTTP response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify task was completed in database
	updatedTask, err := nudgeService.GetTaskByID(createdTask.ID)
	require.NoError(t, err)
	assert.Equal(t, common.TaskStatusCompleted, updatedTask.Status)
	assert.NotNil(t, updatedTask.CompletedAt)

	// Verify confirmation message was sent
	sentMessages := mockTelegramProvider.GetSentMessages()
	assert.NotEmpty(t, sentMessages, "Expected confirmation message to be sent")

	if len(sentMessages) > 0 {
		confirmMessage := sentMessages[len(sentMessages)-1]
		assert.Equal(t, int64(123456), confirmMessage.ChatID)
		assert.Contains(t, confirmMessage.Text, "completed")
	}
}

func TestCommandFlow_DeleteCommand(t *testing.T) {
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

	// Create test task in database
	userID := common.UserID("123456")
	dueTime := time.Now().Add(24 * time.Hour)

	task := &nudge.Task{
		UserID:      userID,
		Description: "Task to be deleted",
		DueDate:     &dueTime,
		Status:      common.TaskStatusActive,
		Priority:    common.TaskPriorityLow,
	}

	err = nudgeService.CreateTask(task)
	require.NoError(t, err)

	// Get created task to get its ID
	tasks, err := nudgeService.GetTasks(userID, nudge.TaskFilter{})
	require.NoError(t, err)
	require.NotEmpty(t, tasks)
	createdTask := tasks[0]

	// Create /delete command webhook payload with task ID
	deleteCommand := createTestTelegramWebhook(123456, "/delete", string(createdTask.ID))

	// Make HTTP request
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/webhook", bytes.NewBuffer(deleteCommand))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	// Verify HTTP response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify task was deleted from database
	remainingTasks, err := nudgeService.GetTasks(userID, nudge.TaskFilter{})
	require.NoError(t, err)
	assert.Empty(t, remainingTasks, "Task should be deleted")

	// Verify confirmation message was sent
	sentMessages := mockTelegramProvider.GetSentMessages()
	assert.NotEmpty(t, sentMessages, "Expected confirmation message to be sent")

	if len(sentMessages) > 0 {
		confirmMessage := sentMessages[len(sentMessages)-1]
		assert.Equal(t, int64(123456), confirmMessage.ChatID)
		assert.Contains(t, confirmMessage.Text, "deleted")
	}
}

func TestCommandFlow_InvalidCommand(t *testing.T) {
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

	// Create invalid command webhook payload
	invalidCommand := createTestTelegramWebhook(123456, "/invalid_command", "")

	// Make HTTP request
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/webhook", bytes.NewBuffer(invalidCommand))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	// Verify HTTP response (should still be 200 for Telegram)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify error message was sent
	sentMessages := mockTelegramProvider.GetSentMessages()
	assert.NotEmpty(t, sentMessages, "Expected error message to be sent")

	if len(sentMessages) > 0 {
		errorMessage := sentMessages[len(sentMessages)-1]
		assert.Equal(t, int64(123456), errorMessage.ChatID)
		assert.Contains(t, errorMessage.Text, "unknown")
	}
}

func TestCommandFlow_CommandWithInvalidArgs(t *testing.T) {
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

	// Create /done command with invalid task ID
	invalidDoneCommand := createTestTelegramWebhook(123456, "/done", "invalid_task_id")

	// Make HTTP request
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/webhook", bytes.NewBuffer(invalidDoneCommand))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	// Verify HTTP response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify error message was sent
	sentMessages := mockTelegramProvider.GetSentMessages()
	assert.NotEmpty(t, sentMessages, "Expected error message to be sent")

	if len(sentMessages) > 0 {
		errorMessage := sentMessages[len(sentMessages)-1]
		assert.Equal(t, int64(123456), errorMessage.ChatID)
		assert.Contains(t, errorMessage.Text, "not found")
	}
}

// Helper function to create test Telegram webhook payloads
func createTestTelegramWebhook(chatID int64, command, args string) []byte {
	message := command
	if args != "" {
		message = command + " " + args
	}

	update := tgbotapi.Update{
		UpdateID: 123,
		Message: &tgbotapi.Message{
			MessageID: 456,
			From: &tgbotapi.User{
				ID:        chatID,
				UserName:  "testuser",
				FirstName: "Test",
				LastName:  "User",
			},
			Chat: &tgbotapi.Chat{
				ID:   chatID,
				Type: "private",
			},
			Date: int(time.Now().Unix()),
			Text: message,
		},
	}

	data, _ := json.Marshal(update)
	return data
}
