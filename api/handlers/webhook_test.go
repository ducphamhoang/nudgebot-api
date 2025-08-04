package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
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

// Mock chatbot service
type mockChatbotService struct {
	shouldFail         bool
	handleWebhookError error
}

func (m *mockChatbotService) SendMessage(chatID common.ChatID, text string) error {
	if m.shouldFail {
		return errors.New("mock send message error")
	}
	return nil
}

func (m *mockChatbotService) SendMessageWithKeyboard(chatID common.ChatID, text string, keyboard chatbot.InlineKeyboard) error {
	if m.shouldFail {
		return errors.New("mock send message with keyboard error")
	}
	return nil
}

func (m *mockChatbotService) HandleWebhook(webhookData []byte) error {
	if m.handleWebhookError != nil {
		return m.handleWebhookError
	}
	return nil
}

func (m *mockChatbotService) ProcessCommand(command chatbot.Command, userID common.UserID, chatID common.ChatID) error {
	if m.shouldFail {
		return errors.New("mock process command error")
	}
	return nil
}

// Mock database
type mockDB struct {
	gorm.DB
	shouldFail bool
}

func newMockDB(shouldFail bool) *mockDB {
	return &mockDB{shouldFail: shouldFail}
}

func setupTest() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func TestWebhookHandler_HandleTelegramWebhook(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func() *mockChatbotService
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful webhook processing",
			requestBody: map[string]interface{}{
				"update_id": 123456,
				"message": map[string]interface{}{
					"message_id": 1,
					"from": map[string]interface{}{
						"id":         12345,
						"first_name": "Test",
						"username":   "testuser",
					},
					"chat": map[string]interface{}{
						"id":   12345,
						"type": "private",
					},
					"text": "Hello bot",
				},
			},
			setupMock: func() *mockChatbotService {
				return &mockChatbotService{shouldFail: false}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "chatbot service error still returns 200",
			requestBody: map[string]interface{}{
				"update_id": 123456,
				"message": map[string]interface{}{
					"message_id": 1,
					"text":       "Hello bot",
				},
			},
			setupMock: func() *mockChatbotService {
				return &mockChatbotService{
					handleWebhookError: errors.New("processing error"),
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "invalid JSON returns 200",
			requestBody: "invalid json{",
			setupMock: func() *mockChatbotService {
				return &mockChatbotService{shouldFail: false}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "empty request body returns 200",
			requestBody: "",
			setupMock: func() *mockChatbotService {
				return &mockChatbotService{shouldFail: false}
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTest()
			mockService := tt.setupMock()
			logger := logger.New()

			handler := NewWebhookHandler(mockService, logger)
			router.POST("/webhook", handler.HandleTelegramWebhook)

			var requestBody []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				requestBody = []byte(str)
			} else {
				requestBody, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify response contains "ok": true
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, true, response["ok"])
		})
	}
}

func TestWebhookHandler_SetupWebhook(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(*testing.T, string)
	}{
		{
			name: "successful webhook setup",
			requestBody: map[string]interface{}{
				"webhook_url": "https://example.com/webhook",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var response map[string]interface{}
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Equal(t, true, response["ok"])
				assert.Equal(t, "Webhook setup requested", response["message"])
				assert.Equal(t, "https://example.com/webhook", response["webhook_url"])
			},
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid json{",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				var response map[string]interface{}
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], "Invalid request body")
			},
		},
		{
			name: "missing webhook URL",
			requestBody: map[string]interface{}{
				"other_field": "value",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				var response map[string]interface{}
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], "Invalid request body")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTest()
			mockService := &mockChatbotService{shouldFail: false}
			logger := logger.New()

			handler := NewWebhookHandler(mockService, logger)
			router.POST("/setup-webhook", handler.SetupWebhook)

			var requestBody []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				requestBody = []byte(str)
			} else {
				requestBody, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/setup-webhook", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w.Body.String())
		})
	}
}

func TestWebhookHandler_GetWebhookInfo(t *testing.T) {
	router := setupTest()
	mockService := &mockChatbotService{shouldFail: false}
	logger := logger.New()

	handler := NewWebhookHandler(mockService, logger)
	router.GET("/webhook-info", handler.GetWebhookInfo)

	req := httptest.NewRequest(http.MethodGet, "/webhook-info", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, true, response["ok"])
	assert.Equal(t, "Webhook info endpoint", response["message"])
}

func TestHealthHandler_Check(t *testing.T) {
	tests := []struct {
		name           string
		dbHealthy      bool
		expectedStatus int
		checkResponse  func(*testing.T, string)
	}{
		{
			name:           "healthy system",
			dbHealthy:      true,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var response map[string]interface{}
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Equal(t, "ok", response["status"])
				assert.Equal(t, "nudgebot-api", response["service"])
				assert.NotNil(t, response["timestamp"])
			},
		},
		{
			name:           "unhealthy database",
			dbHealthy:      false,
			expectedStatus: http.StatusServiceUnavailable,
			checkResponse: func(t *testing.T, body string) {
				var response map[string]interface{}
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Equal(t, "error", response["status"])
				assert.Equal(t, "nudgebot-api", response["service"])
				assert.NotNil(t, response["timestamp"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTest()

			// Create mock database - we'll need to create a proper mock for GORM
			// For now, we'll test the handler logic by injecting a nil DB and checking error handling
			var mockDB *gorm.DB
			if !tt.dbHealthy {
				// This will cause the health check to fail since db is nil
				mockDB = nil
			} else {
				// For healthy case, we'd need a working DB connection
				// In real tests, this would use testcontainers or in-memory DB
				mockDB = &gorm.DB{} // This is a placeholder
			}

			logger := logger.New()
			handler := NewHealthHandler(mockDB, logger)
			router.GET("/health", handler.Check)

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w.Body.String())
		})
	}
}

func TestWebhookHandler_ContentTypeValidation(t *testing.T) {
	router := setupTest()
	mockService := &mockChatbotService{shouldFail: false}
	logger := logger.New()

	handler := NewWebhookHandler(mockService, logger)
	router.POST("/webhook", handler.HandleTelegramWebhook)

	requestBody := map[string]interface{}{
		"update_id": 123456,
		"message": map[string]interface{}{
			"message_id": 1,
			"text":       "Hello bot",
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "text/plain") // Wrong content type

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should still return 200 as per Telegram requirements, but log a warning
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, true, response["ok"])
}

func TestWebhookHandler_LargeRequestHandling(t *testing.T) {
	router := setupTest()
	mockService := &mockChatbotService{shouldFail: false}
	logger := logger.New()

	handler := NewWebhookHandler(mockService, logger)
	router.POST("/webhook", handler.HandleTelegramWebhook)

	// Create a request with a moderately large message
	largeMessage := make([]byte, 1024*10) // 10KB
	for i := range largeMessage {
		largeMessage[i] = 'A'
	}

	requestBody := map[string]interface{}{
		"update_id": 123456,
		"message": map[string]interface{}{
			"message_id": 1,
			"text":       string(largeMessage),
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 200 OK
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, true, response["ok"])
}
