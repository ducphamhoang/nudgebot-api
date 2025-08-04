package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"nudgebot-api/internal/chatbot"
	"nudgebot-api/internal/common"
	"nudgebot-api/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// Mock chatbot service for route testing
type mockChatbotService struct{}

func (m *mockChatbotService) SendMessage(chatID common.ChatID, text string) error {
	return nil
}

func (m *mockChatbotService) SendMessageWithKeyboard(chatID common.ChatID, text string, keyboard chatbot.InlineKeyboard) error {
	return nil
}

func (m *mockChatbotService) HandleWebhook(webhookData []byte) error {
	return nil
}

func (m *mockChatbotService) ProcessCommand(command chatbot.Command, userID common.UserID, chatID common.ChatID) error {
	return nil
}

func createTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	mockChatbot := &mockChatbotService{}
	mockDB := &gorm.DB{}
	logger := logger.New()

	router := gin.New()
	SetupRoutes(router, mockDB, logger, mockChatbot)
	return router
}

func TestSetupRoutes_HealthEndpoint(t *testing.T) {
	router := createTestRouter()

	// Test health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code) // Will fail due to mock DB
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestSetupRoutes_WebhookEndpoint(t *testing.T) {
	router := createTestRouter()

	// Test webhook endpoint with empty body
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telegram/webhook", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code) // Should always return 200 for Telegram
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestSetupRoutes_SetupWebhookEndpoint(t *testing.T) {
	router := createTestRouter()

	// Test setup webhook endpoint with empty body (should fail validation)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telegram/setup-webhook", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code) // Should fail validation
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestSetupRoutes_WebhookInfoEndpoint(t *testing.T) {
	router := createTestRouter()

	// Test webhook info endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/v1/telegram/webhook-info", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestSetupRoutes_APIHealthEndpoint(t *testing.T) {
	router := createTestRouter()

	// Test API health endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code) // Will fail due to mock DB
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestSetupRoutes_NotFoundEndpoint(t *testing.T) {
	router := createTestRouter()

	// Test non-existent endpoint
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSetupRoutes_MethodNotAllowed(t *testing.T) {
	router := createTestRouter()

	// Test wrong method on health endpoint
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code) // Gin returns 404 for method not found
}

func TestSetupRoutes_DependencyInjection(t *testing.T) {
	// Test that routes are properly created with dependencies
	gin.SetMode(gin.TestMode)

	mockChatbot := &mockChatbotService{}
	mockDB := &gorm.DB{}
	logger := logger.New()

	// This should not panic if dependencies are properly injected
	assert.NotPanics(t, func() {
		router := gin.New()
		SetupRoutes(router, mockDB, logger, mockChatbot)
		assert.NotNil(t, router)
	})
}

func TestSetupRoutes_HandlerInitialization(t *testing.T) {
	// Test that handlers are properly initialized with their dependencies
	router := createTestRouter()

	// Test that routes return proper response types
	tests := []struct {
		method              string
		path                string
		expectedContentType string
	}{
		{http.MethodGet, "/health", "application/json; charset=utf-8"},
		{http.MethodGet, "/api/v1/health", "application/json; charset=utf-8"},
		{http.MethodPost, "/api/v1/telegram/webhook", "application/json; charset=utf-8"},
		{http.MethodGet, "/api/v1/telegram/webhook-info", "application/json; charset=utf-8"},
	}

	for _, tt := range tests {
		t.Run(tt.method+"_"+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Verify content type is set correctly
			assert.Equal(t, tt.expectedContentType, w.Header().Get("Content-Type"))
		})
	}
}

func TestSetupRoutes_MiddlewareExecution(t *testing.T) {
	// Test that middleware is properly executed
	router := createTestRouter()

	// Make a request and verify middleware headers/behavior
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify that response is properly formed (middleware working)
	assert.NotEmpty(t, w.Body.String())
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestSetupRoutes_AllEndpointsAccessible(t *testing.T) {
	// Test that all expected endpoints are accessible
	router := createTestRouter()

	// List of expected endpoints
	endpoints := []struct {
		method           string
		path             string
		expectStatusCode int
	}{
		{http.MethodGet, "/health", http.StatusServiceUnavailable},        // Mock DB will fail
		{http.MethodGet, "/api/v1/health", http.StatusServiceUnavailable}, // Mock DB will fail
		{http.MethodPost, "/api/v1/telegram/webhook", http.StatusOK},      // Always returns 200
		{http.MethodGet, "/api/v1/telegram/webhook-info", http.StatusOK},
		{http.MethodPost, "/api/v1/telegram/setup-webhook", http.StatusBadRequest}, // No body = validation error
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.method+"_"+endpoint.path, func(t *testing.T) {
			req := httptest.NewRequest(endpoint.method, endpoint.path, nil)
			if endpoint.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, endpoint.expectStatusCode, w.Code,
				"Endpoint %s %s should return status %d",
				endpoint.method, endpoint.path, endpoint.expectStatusCode)
		})
	}
}
