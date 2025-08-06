//go:build integration

package integration

import (
    "bytes"
    "context"
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
    "nudgebot-api/test/essential/helpers"
)

func TestCallbackFlow_TaskDoneButton(t *testing.T) {
    // Setup test environment
    testContainer, cleanup := helpers.SetupTestDatabase(t)
    defer cleanup()

    eventBus := events.NewEventBus(zap.NewNop())
    nudgeRepo := nudge.NewGormNudgeRepository(testContainer.DB, zap.NewNop())
    nudgeService, err := nudge.NewNudgeService(eventBus, zap.NewNop(), nudgeRepo)
    require.NoError(t, err)

    mockTelegramProvider := mocks.NewMockTelegramProvider()

    chatbotService, err := chatbot.NewChatbotServiceWithProvider(eventBus, zap.NewNop(), mockTelegramProvider, config.ChatbotConfig{
        Token:      "test-token",
        WebhookURL: "https://test.example.com/webhook",
        Timeout:    30,
    })
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
    callbackData := helpers.CreateTaskActionCallbackData("done", string(createdTask.ID))
    callbackQuery := helpers.CreateTestCallbackQuery(callbackData, 123456)

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
    testContainer, cleanup := helpers.SetupTestDatabase(t)
    defer cleanup()

    eventBus := events.NewEventBus(zap.NewNop())
    nudgeRepo := nudge.NewGormNudgeRepository(testContainer.DB, zap.NewNop())
    nudgeService, err := nudge.NewNudgeService(eventBus, zap.NewNop(), nudgeRepo)
    require.NoError(t, err)

    mockTelegramProvider := mocks.NewMockTelegramProvider()

    chatbotService, err := chatbot.NewChatbotServiceWithProvider(eventBus, zap.NewNop(), mockTelegramProvider, config.ChatbotConfig{
        Token:      "test-token",
        WebhookURL: "https://test.example.com/webhook",
        Timeout:    30,
    })
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
    callbackData := helpers.CreateTaskActionCallbackData("delete", string(createdTask.ID))
    callbackQuery := helpers.CreateTestCallbackQuery(callbackData, 123456)

    // Make HTTP request
    w := httptest.NewRecorder()
    req, err := http.NewRequest("POST", "/webhook", bytes.NewBuffer(callbackQuery))
    require.NoError(t, err)
    req.Header.Set("Content-Type", "application/json")

    router.ServeHTTP(w, req)

    // Verify HTTP response
    assert.Equal(t, http.StatusOK, w.Code)

    // Verify task was deleted from database
    _, err = nudgeRepo.GetTaskByID(createdTask.ID)
    assert.Error(t, err, "Task should be deleted from database")

    // Verify callback answer was sent
    assert.True(t, mockTelegramProvider.GetCallCount("AnswerCallbackQuery") > 0)
}

func TestCallbackFlow_SnoozeButton(t *testing.T) {
    // Setup test environment
    testContainer, cleanup := helpers.SetupTestDatabase(t)
    defer cleanup()

    eventBus := events.NewEventBus(zap.NewNop())
    nudgeRepo := nudge.NewGormNudgeRepository(testContainer.DB, zap.NewNop())
    nudgeService, err := nudge.NewNudgeService(eventBus, zap.NewNop(), nudgeRepo)
    require.NoError(t, err)

    mockTelegramProvider := mocks.NewMockTelegramProvider()

    chatbotService, err := chatbot.NewChatbotServiceWithProvider(eventBus, zap.NewNop(), mockTelegramProvider, config.ChatbotConfig{
        Token:      "test-token",
        WebhookURL: "https://test.example.com/webhook",
        Timeout:    30,
    })
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
    callbackData := helpers.CreateSnoozeCallbackData(string(createdTask.ID), "1h")
    callbackQuery := helpers.CreateTestCallbackQuery(callbackData, 123456)

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
    testContainer, cleanup := helpers.SetupTestDatabase(t)
    defer cleanup()

    eventBus := events.NewEventBus(zap.NewNop())
    nudgeRepo := nudge.NewGormNudgeRepository(testContainer.DB, zap.NewNop())
    nudgeService, err := nudge.NewNudgeService(eventBus, zap.NewNop(), nudgeRepo)
    require.NoError(t, err)

    mockTelegramProvider := mocks.NewMockTelegramProvider()

    chatbotService, err := chatbot.NewChatbotServiceWithProvider(eventBus, zap.NewNop(), mockTelegramProvider, config.ChatbotConfig{
        Token:      "test-token",
        WebhookURL: "https://test.example.com/webhook",
        Timeout:    30,
    })
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
    callbackData := helpers.CreatePaginationCallbackData("next", 1)
    callbackQuery := helpers.CreateTestCallbackQuery(callbackData, 123456)

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
    testContainer, cleanup := helpers.SetupTestDatabase(t)
    defer cleanup()

    eventBus := events.NewEventBus(zap.NewNop())
    nudgeRepo := nudge.NewGormNudgeRepository(testContainer.DB, zap.NewNop())
    nudgeService, err := nudge.NewNudgeService(eventBus, zap.NewNop(), nudgeRepo)
    require.NoError(t, err)

    mockTelegramProvider := mocks.NewMockTelegramProvider()

    chatbotService, err := chatbot.NewChatbotServiceWithProvider(eventBus, zap.NewNop(), mockTelegramProvider, config.ChatbotConfig{
        Token:      "test-token",
        WebhookURL: "https://test.example.com/webhook",
        Timeout:    30,
    })
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
    callbackQuery := helpers.CreateTestCallbackQuery("invalid_json_data", 123456)

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
    testContainer, cleanup := helpers.SetupTestDatabase(t)
    defer cleanup()

    eventBus := events.NewEventBus(zap.NewNop())
    nudgeRepo := nudge.NewGormNudgeRepository(testContainer.DB, zap.NewNop())
    nudgeService, err := nudge.NewNudgeService(eventBus, zap.NewNop(), nudgeRepo)
    require.NoError(t, err)

    mockTelegramProvider := mocks.NewMockTelegramProvider()

    chatbotService, err := chatbot.NewChatbotServiceWithProvider(eventBus, zap.NewNop(), mockTelegramProvider, config.ChatbotConfig{
        Token:      "test-token",
        WebhookURL: "https://test.example.com/webhook",
        Timeout:    30,
    })
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
    callbackData := helpers.CreateTaskActionCallbackData("done", "non_existent_task_id")
    callbackQuery := helpers.CreateTestCallbackQuery(callbackData, 123456)

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