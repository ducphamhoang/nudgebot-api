package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nudgebot-api/api/routes"
	"nudgebot-api/internal/chatbot"
	"nudgebot-api/internal/common"
	"nudgebot-api/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// Integration test setup
func setupIntegrationTest(t *testing.T) (*gin.Engine, func()) {
	gin.SetMode(gin.TestMode)

	// Mock dependencies
	mockChatbot := &MockChatbotService{}
	mockDB := &gorm.DB{} // Mock database
	logger := logger.New()

	// Create router
	router := gin.New()
	routes.SetupRoutes(router, mockDB, logger, mockChatbot)

	cleanup := func() {
		// Any cleanup if needed
	}

	return router, cleanup
}

// Mock chatbot service for integration tests
type MockChatbotService struct {
	messages []MockMessage
	webhooks [][]byte
	errors   []error
}

type MockMessage struct {
	ChatID   common.ChatID
	Text     string
	Keyboard chatbot.InlineKeyboard
}

func (m *MockChatbotService) SendMessage(chatID common.ChatID, text string) error {
	if len(m.errors) > 0 {
		err := m.errors[0]
		m.errors = m.errors[1:]
		if err != nil {
			return err
		}
	}

	m.messages = append(m.messages, MockMessage{
		ChatID: chatID,
		Text:   text,
	})
	return nil
}

func (m *MockChatbotService) SendMessageWithKeyboard(chatID common.ChatID, text string, keyboard chatbot.InlineKeyboard) error {
	if len(m.errors) > 0 {
		err := m.errors[0]
		m.errors = m.errors[1:]
		if err != nil {
			return err
		}
	}

	m.messages = append(m.messages, MockMessage{
		ChatID:   chatID,
		Text:     text,
		Keyboard: keyboard,
	})
	return nil
}

func (m *MockChatbotService) HandleWebhook(webhookData []byte) error {
	if len(m.errors) > 0 {
		err := m.errors[0]
		m.errors = m.errors[1:]
		if err != nil {
			return err
		}
	}

	m.webhooks = append(m.webhooks, webhookData)
	return nil
}

func (m *MockChatbotService) ProcessCommand(command chatbot.Command, userID common.UserID, chatID common.ChatID) error {
	if len(m.errors) > 0 {
		err := m.errors[0]
		m.errors = m.errors[1:]
		if err != nil {
			return err
		}
	}

	// Simulate command processing
	return nil
}

func (m *MockChatbotService) SetError(err error) {
	m.errors = append(m.errors, err)
}

func (m *MockChatbotService) GetMessages() []MockMessage {
	return m.messages
}

func (m *MockChatbotService) GetWebhooks() [][]byte {
	return m.webhooks
}

func (m *MockChatbotService) Reset() {
	m.messages = nil
	m.webhooks = nil
	m.errors = nil
}

func TestIntegration_HealthEndpoint(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code) // Mock DB returns unhealthy
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	// Verify response structure
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "status")
}

func TestIntegration_TelegramWebhook(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Test valid webhook payload
	webhookPayload := `{
		"message": {
			"message_id": 123,
			"from": {
				"id": 456,
				"username": "testuser"
			},
			"chat": {
				"id": 789,
				"type": "private"
			},
			"text": "/start"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/telegram/webhook", strings.NewReader(webhookPayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	// Verify response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
}

func TestIntegration_TelegramWebhook_InvalidPayload(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Test invalid JSON
	invalidPayload := `{"invalid": json}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/telegram/webhook", strings.NewReader(invalidPayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should still return 200 for Telegram compatibility
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIntegration_TelegramWebhook_EmptyPayload(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/telegram/webhook", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should still return 200 for Telegram compatibility
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIntegration_SetupWebhook(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	setupPayload := `{"url": "https://example.com/webhook"}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/telegram/setup-webhook", strings.NewReader(setupPayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should fail validation in mock environment
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIntegration_WebhookInfo(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/telegram/webhook-info", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	// Verify response structure
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
}

func TestIntegration_MiddlewareExecution(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Test that middleware is executed
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "test-request-123")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify middleware headers or logs
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestIntegration_ConcurrentRequests(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	const numRequests = 10
	done := make(chan int, numRequests)

	// Send concurrent requests
	for i := 0; i < numRequests; i++ {
		go func(id int) {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			done <- w.Code
		}(i)
	}

	// Collect responses
	statusCodes := make([]int, numRequests)
	for i := 0; i < numRequests; i++ {
		statusCodes[i] = <-done
	}

	// Verify all requests completed
	for _, code := range statusCodes {
		assert.Equal(t, http.StatusServiceUnavailable, code)
	}
}

func TestIntegration_RequestTimeout(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Create request with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	start := time.Now()
	router.ServeHTTP(w, req)
	duration := time.Since(start)

	// Request should complete quickly in test environment
	assert.Less(t, duration, 50*time.Millisecond)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestIntegration_ContentTypeHandling(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	tests := []struct {
		name        string
		contentType string
		body        string
		expectCode  int
	}{
		{
			name:        "Valid JSON",
			contentType: "application/json",
			body:        `{"message": {"text": "test"}}`,
			expectCode:  http.StatusOK,
		},
		{
			name:        "Invalid content type",
			contentType: "text/plain",
			body:        "plain text",
			expectCode:  http.StatusOK, // Telegram webhook should still accept
		},
		{
			name:        "Missing content type",
			contentType: "",
			body:        `{"message": {"text": "test"}}`,
			expectCode:  http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/telegram/webhook", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)
		})
	}
}

func TestIntegration_ErrorHandling(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	tests := []struct {
		name       string
		method     string
		path       string
		expectCode int
	}{
		{
			name:       "Not found",
			method:     http.MethodGet,
			path:       "/nonexistent",
			expectCode: http.StatusNotFound,
		},
		{
			name:       "Method not allowed",
			method:     http.MethodPut,
			path:       "/health",
			expectCode: http.StatusNotFound, // Gin returns 404
		},
		{
			name:       "Valid endpoint",
			method:     http.MethodGet,
			path:       "/health",
			expectCode: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)
		})
	}
}

func TestIntegration_EndToEndWorkflow(t *testing.T) {
	router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Step 1: Check health
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	// Step 2: Send webhook
	webhookPayload := `{
		"message": {
			"message_id": 123,
			"from": {"id": 456, "username": "testuser"},
			"chat": {"id": 789, "type": "private"},
			"text": "/help"
		}
	}`

	req = httptest.NewRequest(http.MethodPost, "/api/v1/telegram/webhook", strings.NewReader(webhookPayload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Step 3: Check webhook info
	req = httptest.NewRequest(http.MethodGet, "/api/v1/telegram/webhook-info", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify all responses have proper content type
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}
