//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nudgebot-api/api/handlers"
	"nudgebot-api/internal/chatbot"
	"nudgebot-api/internal/common"
	"nudgebot-api/internal/config"
	"nudgebot-api/internal/database"
	"nudgebot-api/internal/events"
	"nudgebot-api/internal/llm"
	"nudgebot-api/internal/nudge"
	"nudgebot-api/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// setupTestDatabase sets up a PostgreSQL test database using testcontainers
func setupTestDatabase(t *testing.T) (*gorm.DB, func()) {
	ctx := context.Background()

	// Create PostgreSQL container
	dbContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15"),
		postgres.WithDatabase("test_nudgebot"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")

	// Get the mapped port
	port, err := dbContainer.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get mapped port")

	// Parse connection string to create database config
	dbConfig := config.DatabaseConfig{
		Host:            "localhost",
		Port:            port.Int(),
		User:            "test_user",
		Password:        "test_password",
		DBName:          "test_nudgebot",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 300,
	}

	// Connect to database with retries
	var db *gorm.DB
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		db, err = database.NewPostgresConnection(dbConfig)
		if err == nil {
			break
		}
		if i == maxRetries-1 {
			require.NoError(t, err, "Failed to connect to test database after retries")
		}
		time.Sleep(1 * time.Second)
	}

	// Return database and cleanup function
	cleanup := func() {
		if db != nil {
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}
		}
		dbContainer.Terminate(ctx)
	}

	return db, cleanup
}

// telegramIDToUUID converts a Telegram numeric ID to a deterministic UUID for testing
func telegramIDToUUID(telegramID int64) string {
	// Create a deterministic UUID based on the Telegram ID
	hash := md5.Sum([]byte(fmt.Sprintf("telegram_id_%d", telegramID)))
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		hash[0:4], hash[4:6], hash[6:8], hash[8:10], hash[10:16])
}

// createTestTelegramWebhook creates a test Telegram webhook payload
func createTestTelegramWebhook(userID, chatID, messageText string) []byte {
	webhookJSON := fmt.Sprintf(`{
		"update_id": 123456789,
		"message": {
			"message_id": 1,
			"from": {
				"id": %s,
				"is_bot": false,
				"first_name": "Test",
				"last_name": "User",
				"username": "testuser"
			},
			"chat": {
				"id": %s,
				"first_name": "Test",
				"last_name": "User",
				"username": "testuser",
				"type": "private"
			},
			"date": %d,
			"text": "%s"
		}
	}`, userID, chatID, time.Now().Unix(), messageText)

	return []byte(webhookJSON)
}

// TestMessageFlowIntegration tests the complete webhook-to-database flow
func TestMessageFlowIntegration(t *testing.T) {
	// Skip test if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Log("Starting message flow integration test")

	// Set up PostgreSQL container and database
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Run database migrations
	err := nudge.MigrateWithValidation(db)
	require.NoError(t, err, "Failed to run database migrations")

	// Create MockEventBus with synchronous mode
	eventBus := events.NewMockEventBus()
	eventBus.SetSynchronousMode(true)

	// Create logger
	zapLogger, err := zap.NewDevelopment()
	require.NoError(t, err, "Failed to create logger")

	loggerInstance := logger.New()

	// Create stub providers
	stubLLMProvider := llm.NewStubLLMProvider(zapLogger)
	stubTelegramProvider := chatbot.NewStubTelegramProvider(zapLogger)

	// Create services with stub providers
	llmService := llm.NewLLMServiceWithProvider(eventBus, zapLogger, stubLLMProvider)

	chatbotConfig := config.ChatbotConfig{
		Token:      "test-token",
		WebhookURL: "https://test.example.com/webhook",
		Timeout:    30,
	}

	chatbotService, err := chatbot.NewChatbotServiceWithProvider(eventBus, zapLogger, stubTelegramProvider, chatbotConfig)
	require.NoError(t, err, "Failed to create chatbot service")

	// Create nudge service with real repository
	nudgeRepo := nudge.NewGormNudgeRepository(db, zapLogger)

	nudgeService, err := nudge.NewNudgeService(eventBus, zapLogger, nudgeRepo)
	require.NoError(t, err, "Failed to create nudge service")

	// Ensure services are initialized (avoid "declared but not used" errors)
	_ = llmService
	_ = nudgeService

	// Create webhook handler
	webhookHandler := handlers.NewWebhookHandler(chatbotService, loggerInstance)

	// Set up gin router for testing
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/webhook", webhookHandler.HandleTelegramWebhook)

	// Create test webhook payload
	testTelegramUserID := 550840000 // Numeric Telegram user ID
	testTelegramChatID := 67890     // Numeric Telegram chat ID
	testMessage := "call mom tomorrow at 5pm"

	// Convert Telegram IDs to expected UUID format for validation
	expectedUserID := telegramIDToUUID(int64(testTelegramUserID))
	expectedChatID := telegramIDToUUID(int64(testTelegramChatID))

	webhookPayload := createTestTelegramWebhook(fmt.Sprintf("%d", testTelegramUserID), fmt.Sprintf("%d", testTelegramChatID), testMessage)

	// Create HTTP request
	req, err := http.NewRequest("POST", "/webhook", bytes.NewBuffer(webhookPayload))
	require.NoError(t, err, "Failed to create HTTP request")
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	recorder := httptest.NewRecorder()

	t.Log("Sending webhook request with test message:", testMessage)

	// Process the webhook
	router.ServeHTTP(recorder, req)

	// Verify webhook response
	assert.Equal(t, http.StatusOK, recorder.Code, "Webhook should return 200 OK")

	// Wait for TaskCreated event with timeout
	t.Log("Waiting for TaskCreated event...")

	taskCreatedEvent, err := eventBus.WaitForEvent(events.TopicTaskCreated, 2*time.Second)
	require.NoError(t, err, "Should receive TaskCreated event within timeout")
	require.NotNil(t, taskCreatedEvent, "TaskCreated event should not be nil")

	t.Log("TaskCreated event received successfully")

	// Extract task ID from event data
	eventData, ok := taskCreatedEvent.(events.TaskCreated)
	require.True(t, ok, "Event data should be TaskCreated")

	// Query database to verify task creation
	var createdTask nudge.Task
	err = db.Where("id = ?", eventData.TaskID).First(&createdTask).Error
	require.NoError(t, err, "Should find created task in database")

	// Verify task fields
	assert.Equal(t, "Call mom", createdTask.Title, "Task title should match")
	assert.Equal(t, "Call mom tomorrow at 5pm", createdTask.Description, "Task description should match")
	assert.Equal(t, common.PriorityMedium, createdTask.Priority, "Task priority should be medium")
	assert.Equal(t, common.TaskStatusActive, createdTask.Status, "Task status should be active")
	assert.NotNil(t, createdTask.DueDate, "Task should have a due date")

	// Verify due date is tomorrow at 5pm
	if createdTask.DueDate != nil {
		expectedTime := time.Now().AddDate(0, 0, 1)
		expectedDueDate := time.Date(expectedTime.Year(), expectedTime.Month(), expectedTime.Day(), 17, 0, 0, 0, expectedTime.Location())

		// Allow for some tolerance in time comparison (within 1 minute)
		timeDiff := createdTask.DueDate.Sub(expectedDueDate)
		assert.True(t, timeDiff >= -time.Minute && timeDiff <= time.Minute,
			"Due date should be tomorrow at 5pm (within 1 minute tolerance). Expected: %v, Got: %v",
			expectedDueDate, *createdTask.DueDate)
	}

	// Verify user and chat IDs
	assert.Equal(t, common.UserID(expectedUserID), createdTask.UserID, "User ID should match")
	assert.Equal(t, common.ChatID(expectedChatID), createdTask.ChatID, "Chat ID should match")

	// Verify event bus processed all expected events
	t.Log("Verifying event flow...")

	// Check that MessageReceived event was published
	messageEvents := eventBus.GetPublishedEvents(events.TopicMessageReceived)
	assert.Len(t, messageEvents, 1, "Should have published one MessageReceived event")

	// Check that TaskParsed event was published
	taskParsedEvents := eventBus.GetPublishedEvents(events.TopicTaskParsed)
	assert.Len(t, taskParsedEvents, 1, "Should have published one TaskParsed event")

	// Check that TaskCreated event was published
	taskCreatedEvents := eventBus.GetPublishedEvents(events.TopicTaskCreated)
	assert.Len(t, taskCreatedEvents, 1, "Should have published one TaskCreated event")

	t.Log("Integration test completed successfully")
	t.Logf("Created task: ID=%s, Title=%s, Priority=%s, DueDate=%v",
		createdTask.ID, createdTask.Title, createdTask.Priority, createdTask.DueDate)
}

// TestMessageFlowIntegrationWithInvalidMessage tests error handling
func TestMessageFlowIntegrationWithInvalidMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Log("Starting invalid message integration test")

	// Set up test infrastructure (similar to main test)
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	err := nudge.MigrateWithValidation(db)
	require.NoError(t, err)

	eventBus := events.NewMockEventBus()
	eventBus.SetSynchronousMode(true)

	zapLogger, err := zap.NewDevelopment()
	require.NoError(t, err)

	loggerInstance := logger.New()

	// Create services
	stubLLMProvider := llm.NewStubLLMProvider(zapLogger)
	stubTelegramProvider := chatbot.NewStubTelegramProvider(zapLogger)

	llmService := llm.NewLLMServiceWithProvider(eventBus, zapLogger, stubLLMProvider)

	chatbotConfig := config.ChatbotConfig{
		Token:      "test-token",
		WebhookURL: "https://test.example.com/webhook",
		Timeout:    30,
	}

	chatbotService, err := chatbot.NewChatbotServiceWithProvider(eventBus, zapLogger, stubTelegramProvider, chatbotConfig)
	require.NoError(t, err)

	nudgeRepo := nudge.NewGormNudgeRepository(db, zapLogger)

	nudgeService, err := nudge.NewNudgeService(eventBus, zapLogger, nudgeRepo)
	require.NoError(t, err)

	// Ensure services are initialized
	_ = llmService
	_ = nudgeService

	webhookHandler := handlers.NewWebhookHandler(chatbotService, loggerInstance)

	// Test with invalid JSON
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/webhook", webhookHandler.HandleTelegramWebhook)

	invalidPayload := []byte(`{"invalid": "json"`)
	req, err := http.NewRequest("POST", "/webhook", bytes.NewBuffer(invalidPayload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	// Should still return 200 OK per Telegram webhook requirements
	assert.Equal(t, http.StatusOK, recorder.Code, "Webhook should return 200 OK even for invalid payload")

	t.Log("Invalid message test completed successfully")
}
