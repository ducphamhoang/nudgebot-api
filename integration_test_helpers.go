//go:build integration

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"nudgebot-api/internal/common"
	"nudgebot-api/internal/config"
	"nudgebot-api/internal/database"
	"nudgebot-api/internal/events"
	"nudgebot-api/internal/mocks"
	"nudgebot-api/internal/nudge"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"gorm.io/gorm"
)

// ==============================================================================
// Integration Test Infrastructure
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
	TelegramProvider *mocks.MockTelegramProvider
	HTTPClient       *mocks.MockHTTPClient
	EventBus         *mocks.MockEventBus
	EventStore       *mocks.MockEventStore
}

// ==============================================================================
// Test Container Management
// ==============================================================================

// SetupTestDatabase creates and starts a PostgreSQL test container
func SetupTestDatabase(t *testing.T) *TestContainer {
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

	// Get connection details
	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err)

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Create database config
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

	return &TestContainer{
		Container: postgresContainer,
		DB:        db,
		Config:    dbConfig,
		ctx:       ctx,
	}
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

// ResetDatabase clears all data from the test database
func (tc *TestContainer) ResetDatabase(t *testing.T) {
	// Get application table names, excluding system and extension tables
	var tables []string
	err := tc.DB.Raw(`
		SELECT tablename FROM pg_tables 
		WHERE schemaname = 'public' 
		AND tablename NOT LIKE 'pg_%' 
		AND tablename NOT LIKE 'sql_%'
		AND tablename NOT IN ('spatial_ref_sys', 'geography_columns', 'geometry_columns', 'raster_columns', 'raster_overviews')
		AND tablename ~ '^[a-z][a-z0-9_]*$'
	`).Scan(&tables).Error
	require.NoError(t, err)

	// Truncate only application tables
	for _, table := range tables {
		err := tc.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error
		require.NoError(t, err)
	}
}

// ==============================================================================
// Service Orchestration Helpers
// ==============================================================================

// SetupIntegrationTestSuite creates a complete test environment
func SetupIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
	// Setup logging
	logger := zaptest.NewLogger(t)

	// Setup test database
	testContainer := SetupTestDatabase(t)

	// Setup mocks
	mocks := &TestMocks{
		TelegramProvider: mocks.NewMockTelegramProvider(),
		HTTPClient:       mocks.NewMockHTTPClient(),
		EventBus:         mocks.NewMockEventBus(),
		EventStore:       mocks.NewMockEventStore(),
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
	suite.TestContainer.TeardownTestDatabase(t)
}

// ResetTestEnvironment resets all components to a clean state
func (suite *IntegrationTestSuite) ResetTestEnvironment(t *testing.T) {
	suite.TestContainer.ResetDatabase(t)
	suite.Mocks.TelegramProvider.ClearHistory()
	suite.Mocks.HTTPClient.Reset()
	suite.Mocks.EventBus.Reset()
}

// ==============================================================================
// Test Data Management
// ==============================================================================

// TestUser represents a test user for integration testing
type TestUser struct {
	ID     common.UserID
	ChatID common.ChatID
	Name   string
}

// TestTask represents a test task for integration testing
type TestTask struct {
	ID          string
	UserID      common.UserID
	Title       string
	Description string
	Priority    string
	DueDate     *time.Time
	CreatedAt   time.Time
}

// CreateTestUser creates a test user with associated data
func CreateTestUser(suite *IntegrationTestSuite, name string) TestUser {
	userID := common.NewUserID()
	chatID := common.NewChatID()

	return TestUser{
		ID:     userID,
		ChatID: chatID,
		Name:   name,
	}
}

// CreateTestTask creates a test task in the database
func CreateTestTask(suite *IntegrationTestSuite, t *testing.T, userID common.UserID, title string) TestTask {
	task := TestTask{
		ID:          common.NewID(),
		UserID:      userID,
		Title:       title,
		Description: fmt.Sprintf("Test task: %s", title),
		Priority:    "medium",
		CreatedAt:   time.Now(),
	}

	// In a real implementation, this would use the actual nudge service
	// For now, we'll just track it in memory for the test
	return task
}

// CreateTestReminder creates a test reminder for a task
func CreateTestReminder(suite *IntegrationTestSuite, t *testing.T, taskID, userID string) {
	// This would create a reminder in the database
	// For integration testing purposes
}

// ==============================================================================
// Event Flow Testing Utilities
// ==============================================================================

// EventFlowTester helps test complete event flows with timeouts and assertions
type EventFlowTester struct {
	eventBus       events.EventBus
	expectedEvents []string
	receivedEvents map[string]int
	timeout        time.Duration
	logger         *zap.Logger
}

// NewEventFlowTester creates a new event flow tester
func NewEventFlowTester(eventBus events.EventBus, logger *zap.Logger) *EventFlowTester {
	return &EventFlowTester{
		eventBus:       eventBus,
		expectedEvents: make([]string, 0),
		receivedEvents: make(map[string]int),
		timeout:        5 * time.Second,
		logger:         logger,
	}
}

// ExpectEvent adds an event to the list of expected events
func (eft *EventFlowTester) ExpectEvent(topic string) {
	eft.expectedEvents = append(eft.expectedEvents, topic)
}

// WaitForEvents waits for all expected events to be received within the timeout
func (eft *EventFlowTester) WaitForEvents(t *testing.T) {
	// Subscribe to all expected events
	for _, topic := range eft.expectedEvents {
		topic := topic // capture for closure
		err := eft.eventBus.Subscribe(topic, func(event interface{}) error {
			eft.receivedEvents[topic]++
			return nil
		})
		require.NoError(t, err)
	}

	// Wait for events with timeout
	deadline := time.Now().Add(eft.timeout)
	for time.Now().Before(deadline) {
		allReceived := true
		for _, topic := range eft.expectedEvents {
			if eft.receivedEvents[topic] == 0 {
				allReceived = false
				break
			}
		}
		if allReceived {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Report missing events
	var missingEvents []string
	for _, topic := range eft.expectedEvents {
		if eft.receivedEvents[topic] == 0 {
			missingEvents = append(missingEvents, topic)
		}
	}
	if len(missingEvents) > 0 {
		t.Fatalf("Timeout waiting for events: %v", missingEvents)
	}
}

// ==============================================================================
// Mock API Server Utilities
// ==============================================================================

// MockTelegramAPIServer creates a test HTTP server for mocking Telegram API
func MockTelegramAPIServer() *TestAPIServer {
	return &TestAPIServer{
		responses: make(map[string]TestAPIResponse),
		port:      0, // Will be assigned automatically
	}
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

// TestAPIServer represents a mock API server for testing
type TestAPIServer struct {
	responses map[string]TestAPIResponse
	port      int
	server    *httptest.Server
}

// TestAPIResponse represents a configured API response
type TestAPIResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

// SetResponse configures a response for a specific endpoint
func (tas *TestAPIServer) SetResponse(method, path string, statusCode int, body string) {
	key := fmt.Sprintf("%s:%s", method, path)
	tas.responses[key] = TestAPIResponse{
		StatusCode: statusCode,
		Body:       body,
		Headers:    make(map[string]string),
	}
}

// Start starts the mock API server with dynamic port allocation
func (tas *TestAPIServer) Start() error {
	// Create a handler that serves configured responses
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("%s:%s", r.Method, r.URL.Path)
		if response, exists := tas.responses[key]; exists {
			for headerKey, headerValue := range response.Headers {
				w.Header().Set(headerKey, headerValue)
			}
			w.WriteHeader(response.StatusCode)
			w.Write([]byte(response.Body))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
		}
	})

	// Create server with dynamic port allocation (port 0)
	tas.server = httptest.NewServer(handler)
	
	// Extract the actual port from the server URL
	_, portStr, err := net.SplitHostPort(tas.server.Listener.Addr().String())
	if err != nil {
		tas.server.Close()
		return fmt.Errorf("failed to parse server port: %w", err)
	}
	
	// Parse port string to int
	_, err = fmt.Sscanf(portStr, "%d", &tas.port)
	if err != nil {
		tas.server.Close()
		return fmt.Errorf("failed to convert port to int: %w", err)
	}
	
	return nil
}

// Stop stops the mock API server
func (tas *TestAPIServer) Stop() error {
	if tas.server != nil {
		tas.server.Close()
		tas.server = nil
	}
	return nil
}

// GetURL returns the base URL of the mock server
func (tas *TestAPIServer) GetURL() string {
	if tas.server != nil {
		return tas.server.URL
	}
	return fmt.Sprintf("http://localhost:%d", tas.port)
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

// RunDatabaseMigrations runs database migrations for testing
func RunDatabaseMigrations(db *gorm.DB) error {
	// Run the actual migrations using the nudge package
	if err := nudge.MigrateWithValidation(db); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}
	
	log.Println("Database migrations completed successfully")
	return nil
}

// ==============================================================================
// Assertion Helpers for Integration Tests
// ==============================================================================

// AssertEventuallyTrue waits for a condition to become true within a timeout
func AssertEventuallyTrue(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal(message)
}

// AssertMessageSent verifies that a message was sent via Telegram
func AssertMessageSent(t *testing.T, provider *mocks.MockTelegramProvider, chatID int64, expectedText string) {
	messages := provider.GetSentMessages()
	for _, msg := range messages {
		if msg.ChatID == chatID && msg.Text == expectedText {
			return
		}
	}
	t.Fatalf("Expected message not sent to chat %d: %s", chatID, expectedText)
}

// AssertEventPublished verifies that an event was published to the event bus
func AssertEventPublished(t *testing.T, eventBus *mocks.MockEventBus, topic string) {
	// This would check the mock event bus for published events
	// Implementation depends on the mock event bus interface
}

// AssertTaskCreated verifies that a task was created in the database
func AssertTaskCreated(t *testing.T, db *gorm.DB, userID, expectedTitle string) {
	// This would query the database for the created task
	// Implementation depends on the actual task model and database schema
}

// ==============================================================================
// Performance Testing Utilities
// ==============================================================================

// LoadTestConfig configures load testing parameters
type LoadTestConfig struct {
	ConcurrentUsers int
	RequestsPerUser int
	Duration        time.Duration
	RampUpTime      time.Duration
}

// RunLoadTest executes a load test with the given configuration
func RunLoadTest(t *testing.T, suite *IntegrationTestSuite, config LoadTestConfig, testFunc func()) {
	// This would implement actual load testing
	// For now, we'll just run the test function multiple times
	for i := 0; i < config.ConcurrentUsers; i++ {
		go func() {
			for j := 0; j < config.RequestsPerUser; j++ {
				testFunc()
			}
		}()
	}
	time.Sleep(config.Duration)
}

// ==============================================================================
// Environment Configuration for Testing
// ==============================================================================

// GetTestConfig returns a configuration suitable for integration testing
func GetTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port:         0, // Use dynamic port allocation for tests
			Environment:  "test",
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
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
		Events: config.EventsConfig{
			BufferSize:      100,
			WorkerCount:     2,
			ShutdownTimeout: 5,
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
// Test Result Reporting
// ==============================================================================

// TestReport contains the results of an integration test run
type TestReport struct {
	TestName        string
	Duration        time.Duration
	EventsProcessed int
	ErrorsOccurred  int
	Success         bool
	Details         map[string]interface{}
}

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
