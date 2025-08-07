//go:build integration

package helpers

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"gorm.io/gorm"

	"nudgebot-api/internal/common"
	"nudgebot-api/internal/config"
	"nudgebot-api/internal/database"
	"nudgebot-api/internal/events"
	internalmocks "nudgebot-api/internal/mocks"
	"nudgebot-api/internal/nudge"
)

// ==============================================================================
// Core Test Infrastructure Types
// ==============================================================================

// TestContainer manages the lifecycle of a test database container
type TestContainer struct {
	Container testcontainers.Container
	DB        *gorm.DB
	Config    config.DatabaseConfig
	ctx       context.Context
}

// IntegrationTestSuite provides a complete test environment
type IntegrationTestSuite struct {
	TestContainer *TestContainer
	EventBus      events.EventBus
	Logger        *zap.Logger
	Mocks         *TestMocks
}

// TestMocks contains all mock services for integration testing
type TestMocks struct {
	TelegramProvider *internalmocks.MockTelegramProvider
	HTTPClient       *internalmocks.MockHTTPClient
	EventBus         *events.MockEventBus
}

// TestAPIServer represents a mock API server for testing
type TestAPIServer struct {
	responses map[string]TestAPIResponse
	port      int
	server    *httptest.Server
}

// TestAPIResponse represents a mock API response
type TestAPIResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

// EventFlowTester provides event-driven test assertions with timeouts
type EventFlowTester struct {
	timeout time.Duration
	mu      sync.RWMutex
}

// TestTask represents a task for testing purposes
type TestTask struct {
	ID          string
	UserID      common.UserID
	Title       string
	Description string
	Priority    string
	CreatedAt   time.Time
}

// TestReport contains test execution results
type TestReport struct {
	TestName        string
	Success         bool
	Duration        time.Duration
	EventsProcessed int
	ErrorsOccurred  int
	Details         map[string]interface{}
}

// LoadTestConfig configures load testing parameters
type LoadTestConfig struct {
	ConcurrentUsers int
	RequestsPerUser int
	Duration        time.Duration
	RampUpTime      time.Duration
}

// ==============================================================================
// Test Container Management
// ==============================================================================

// SetupTestDatabase creates and starts a PostgreSQL test container
func SetupTestDatabase(t *testing.T) (*TestContainer, func()) {
	ctx := context.Background()

	// Create PostgreSQL container
	postgresContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("test_nudgebot"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")

	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err, "Failed to get container host")

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get container port")

	// Configure database connection
	dbConfig := config.DatabaseConfig{
		Host:            host,
		Port:            port.Int(),
		User:            "test_user",
		Password:        "test_password",
		DBName:          "test_nudgebot",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 300,
	}

	// Connect to database
	db, err := database.NewPostgresConnection(dbConfig)
	require.NoError(t, err, "Failed to connect to test database")

	// Run database migrations
	err = nudge.MigrateWithValidation(db)
	require.NoError(t, err, "Failed to run database migrations")

	testContainer := &TestContainer{
		Container: postgresContainer,
		DB:        db,
		Config:    dbConfig,
		ctx:       ctx,
	}

	// Cleanup function
	cleanup := func() {
		testContainer.TeardownTestDatabase(t)
	}

	return testContainer, cleanup
}

// TeardownTestDatabase stops and removes the test container
func (tc *TestContainer) TeardownTestDatabase(t *testing.T) {
	if tc.DB != nil {
		sqlDB, err := tc.DB.DB()
		if err == nil {
			sqlDB.Close()
		}
	}

	if tc.Container != nil {
		err := tc.Container.Terminate(tc.ctx)
		require.NoError(t, err, "Failed to terminate test container")
	}
}

// ResetDatabase cleans all data from the test database
func (tc *TestContainer) ResetDatabase() error {
	if tc.DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Get all table names
	var tables []string
	err := tc.DB.Raw("SELECT tablename FROM pg_tables WHERE schemaname = 'public'").Scan(&tables).Error
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}

	// Disable foreign key checks and truncate all tables
	for _, table := range tables {
		if err := tc.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error; err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	return nil
}

// ==============================================================================
// Service Orchestration Helpers
// ==============================================================================

// SetupIntegrationTestSuite creates a complete test environment
func SetupIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
	// Setup logging
	logger := zaptest.NewLogger(t)

	// Setup test database
	testContainer, _ := SetupTestDatabase(t)

	// Setup mocks
	mocks := &TestMocks{
		TelegramProvider: &internalmocks.MockTelegramProvider{}, // This would need proper initialization
		HTTPClient:       &internalmocks.MockHTTPClient{},       // This would need proper initialization
		EventBus:         events.NewMockEventBus(),
	}

	// Configure successful responses by default
	mocks.TelegramProvider.SetSendMessageError(nil)
	mocks.TelegramProvider.SetSendKeyboardError(nil)
	mocks.HTTPClient.SetDefaultResponse(200, `{"status": "ok"}`)

	return &IntegrationTestSuite{
		TestContainer: testContainer,
		EventBus:      mocks.EventBus,
		Logger:        logger,
		Mocks:         mocks,
	}
}

// TeardownIntegrationTestSuite cleans up the test environment
func (suite *IntegrationTestSuite) TeardownIntegrationTestSuite(t *testing.T) {
	if suite.TestContainer != nil {
		suite.TestContainer.TeardownTestDatabase(t)
	}
}

// ResetTestEnvironment resets state between tests
func (suite *IntegrationTestSuite) ResetTestEnvironment(t *testing.T) error {
	if err := suite.TestContainer.ResetDatabase(); err != nil {
		return err
	}
	return nil
}

// ResetSharedState wraps ResetTestEnvironment for master suite usage
func ResetSharedState(t *testing.T) {
	// This would be implemented to reset any global state
	// For now, it's a placeholder that can be extended as needed
	t.Helper()
}

// ==============================================================================
// Mock Infrastructure
// ==============================================================================

// MockTelegramAPIServer creates a test HTTP server for mocking Telegram API
func MockTelegramAPIServer() *TestAPIServer {
	server := &TestAPIServer{
		responses: make(map[string]TestAPIResponse),
		port:      0,
	}

	// Set default Telegram response
	server.SetResponse("POST", "/bot123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11/sendMessage", 200, `{
        "ok": true,
        "result": {
            "message_id": 123,
            "from": {
                "id": 123456,
                "is_bot": true,
                "first_name": "TestBot"
            },
            "chat": {
                "id": 123456,
                "type": "private"
            },
            "date": 1234567890,
            "text": "Test message"
        }
    }`)

	return server
}

// MockGemmaAPIServer creates a test HTTP server for mocking Gemma API
func MockGemmaAPIServer() *TestAPIServer {
	server := &TestAPIServer{
		responses: make(map[string]TestAPIResponse),
		port:      0,
	}

	// Set default Gemma response
	server.SetResponse("POST", "/v1beta/models/gemma-2-27b-it:generateContent", 200, `{
        "candidates": [
            {
                "content": {
                    "parts": [
                        {
                            "text": "TASK_EXTRACTED:{\"title\":\"Test Task\",\"description\":\"Test description\",\"priority\":\"medium\"}"
                        }
                    ]
                },
                "finishReason": "STOP"
            }
        ]
    }`)

	return server
}

// SetResponse sets a mock API response for a specific method and path
func (tas *TestAPIServer) SetResponse(method, path string, statusCode int, body string) {
	key := fmt.Sprintf("%s %s", method, path)
	tas.responses[key] = TestAPIResponse{
		StatusCode: statusCode,
		Body:       body,
		Headers:    make(map[string]string),
	}
}

// Start starts the mock HTTP server
func (tas *TestAPIServer) Start() {
	if tas.server != nil {
		return // Already started
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("%s %s", r.Method, r.URL.Path)

		if response, exists := tas.responses[key]; exists {
			for k, v := range response.Headers {
				w.Header().Set(k, v)
			}
			w.WriteHeader(response.StatusCode)
			w.Write([]byte(response.Body))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "not found"}`))
		}
	})

	tas.server = httptest.NewServer(mux)
}

// Stop stops the mock HTTP server
func (tas *TestAPIServer) Stop() {
	if tas.server != nil {
		tas.server.Close()
		tas.server = nil
	}
}

// GetURL returns the base URL of the mock server
func (tas *TestAPIServer) GetURL() string {
	if tas.server != nil {
		return tas.server.URL
	}
	return fmt.Sprintf("http://localhost:%d", tas.port)
}

// AssertEventuallyTrue waits for condition to become true
func (e *EventFlowTester) AssertEventuallyTrue(t *testing.T, condition func() bool, message string) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(e.timeout)

	for {
		select {
		case <-timeout:
			t.Fatalf("Timeout waiting for condition: %s", message)
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

// AssertMessageSent validates message was sent
func AssertMessageSent(t *testing.T, mockServer *TestAPIServer, expectedText string) {
	// Implementation would check mock server received expected message
	t.Helper()
	// Placeholder implementation
}

// AssertEventPublished validates event was published
func AssertEventPublished(t *testing.T, eventType string) {
	t.Helper()
	// Placeholder implementation
}

// AssertTaskCreated validates task was created in database
func AssertTaskCreated(t *testing.T, db *gorm.DB, expectedTitle string) {
	t.Helper()
	// Placeholder implementation
}

// ==============================================================================
// Test Utilities
// ==============================================================================

// CreateTestTelegramWebhook generates a properly formatted Telegram webhook JSON payload for testing
func CreateTestTelegramWebhook(userID, chatID, messageText string) []byte {
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

// TelegramIDToUUID converts a Telegram numeric ID to a deterministic UUID for testing
func TelegramIDToUUID(telegramID int64) string {
	// Create a deterministic UUID based on the Telegram ID
	hash := md5.Sum([]byte(fmt.Sprintf("telegram_id_%d", telegramID)))
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		hash[0:4], hash[4:6], hash[6:8], hash[8:10], hash[10:16])
}

// CreateTaskActionCallbackData creates callback data for task actions
func CreateTaskActionCallbackData(action, taskID string) string {
	data := map[string]interface{}{
		"action":  action,
		"task_id": taskID,
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

// CreateSnoozeCallbackData creates callback data for snooze actions
func CreateSnoozeCallbackData(taskID, duration string) string {
	data := map[string]interface{}{
		"action":   "snooze",
		"task_id":  taskID,
		"duration": duration,
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

// CreatePaginationCallbackData creates callback data for pagination
func CreatePaginationCallbackData(direction string, page int) string {
	data := map[string]interface{}{
		"action": direction,
		"page":   page,
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

// CreateTestCallbackQuery creates test callback query payload
func CreateTestCallbackQuery(callbackData string, chatID int64) []byte {
	query := map[string]interface{}{
		"update_id": 123456,
		"callback_query": map[string]interface{}{
			"id": "test_callback_id",
			"from": map[string]interface{}{
				"id":         chatID,
				"first_name": "Test",
				"username":   "testuser",
			},
			"message": map[string]interface{}{
				"message_id": 1,
				"chat": map[string]interface{}{
					"id":   chatID,
					"type": "private",
				},
			},
			"data": callbackData,
		},
	}

	data, _ := json.Marshal(query)
	return data
}

// CreateTestUser creates test user in database
func CreateTestUser(db *gorm.DB, telegramID int64) (common.UserID, error) {
	userID := common.UserID(common.NewID())
	user := map[string]interface{}{
		"id":          userID,
		"telegram_id": telegramID,
		"username":    "testuser",
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
	}

	if err := db.Table("users").Create(user).Error; err != nil {
		return common.UserID(""), err
	}

	return userID, nil
}

// CreateTestTask creates test task in database
func CreateTestTask(db *gorm.DB, userID common.UserID, title string) (common.TaskID, error) {
	taskID := common.TaskID(common.NewID())
	task := map[string]interface{}{
		"id":         taskID,
		"user_id":    userID,
		"title":      title,
		"status":     "pending",
		"created_at": time.Now(),
		"updated_at": time.Now(),
	}

	if err := db.Table("tasks").Create(task).Error; err != nil {
		return common.TaskID(""), err
	}

	return taskID, nil
}

// CreateTestReminder creates test reminder in database
func CreateTestReminder(db *gorm.DB, taskID common.TaskID, reminderTime time.Time) error {
	reminderID := common.ID(common.NewID()) // Using base ID type for reminder
	reminder := map[string]interface{}{
		"id":            reminderID,
		"task_id":       taskID,
		"user_id":       "test-user-id",
		"chat_id":       "test-chat-id",
		"scheduled_at":  reminderTime,
		"sent_at":       nil,
		"reminder_type": "initial",
	}

	return db.Table("reminders").Create(reminder).Error
}

// ==============================================================================
// Environment Setup/Cleanup
// ==============================================================================

// SetupTestEnvironment sets up environment variables for testing
func SetupTestEnvironment() {
	os.Setenv("GO_ENV", "test")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("DATABASE_HOST", "localhost")
	os.Setenv("DATABASE_PORT", "5432")
}

// CleanupTestEnvironment removes test environment variables
func CleanupTestEnvironment() {
	testEnvVars := []string{
		"GO_ENV",
		"LOG_LEVEL",
		"DATABASE_HOST",
		"DATABASE_PORT",
	}

	for _, envVar := range testEnvVars {
		os.Unsetenv(envVar)
	}
}

// ==============================================================================
// Database Test Utilities
// ==============================================================================

// WaitForDatabaseReady waits for the database to be ready for queries
func WaitForDatabaseReady(db *gorm.DB, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := database.HealthCheck(db); err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("database not ready within timeout")
}

// ==============================================================================
// Test Configuration
// ==============================================================================

// GetTestConfig returns a test configuration
func GetTestConfig() config.Config {
	return config.Config{
		Database: config.DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "test_user",
			Password:        "test_password",
			DBName:          "test_nudgebot",
			SSLMode:         "disable",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 300,
		},
		Scheduler: config.SchedulerConfig{
			PollInterval:    5,
			NudgeDelay:      60,
			WorkerCount:     1,
			ShutdownTimeout: 5,
			Enabled:         true,
		},
	}
}

// ==============================================================================
// Test Reporting
// ==============================================================================

// ReportTestResults logs the results of an integration test
func ReportTestResults(t *testing.T, report TestReport) {
	status := "PASS"
	if !report.Success {
		status = "FAIL"
	}

	log.Printf("Integration Test Report: %s", report.TestName)
	log.Printf("Status: %s", status)
	log.Printf("Duration: %v", report.Duration)
	log.Printf("Events Processed: %d", report.EventsProcessed)
	log.Printf("Errors: %d", report.ErrorsOccurred)

	if len(report.Details) > 0 {
		log.Printf("Details: %+v", report.Details)
	}
}
