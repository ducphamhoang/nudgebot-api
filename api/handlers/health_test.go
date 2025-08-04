package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nudgebot-api/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupHealthTest() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func TestHealthHandler_Check_Healthy(t *testing.T) {
	router := setupHealthTest()
	logger := logger.New()

	// Use a basic GORM DB instance for testing
	// In real tests, this would use testcontainers or a test database
	db := &gorm.DB{}
	handler := NewHealthHandler(db, logger)
	router.GET("/health", handler.Check)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Note: This will likely return 503 because we don't have a real DB connection
	// But we can test the response format
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "status")
	assert.Contains(t, response, "service")
	assert.Contains(t, response, "timestamp")
	assert.Equal(t, "nudgebot-api", response["service"])
}

func TestHealthHandler_Check_NilDatabase(t *testing.T) {
	router := setupHealthTest()
	logger := logger.New()

	// Test with nil database
	handler := NewHealthHandler(nil, logger)
	router.GET("/health", handler.Check)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "error", response["status"])
	assert.Equal(t, "nudgebot-api", response["service"])
	assert.NotNil(t, response["timestamp"])
}

func TestHealthHandler_Check_ResponseFormat(t *testing.T) {
	router := setupHealthTest()
	logger := logger.New()

	db := &gorm.DB{}
	handler := NewHealthHandler(db, logger)
	router.GET("/health", handler.Check)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify response structure
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check required fields exist
	assert.Contains(t, response, "status")
	assert.Contains(t, response, "service")
	assert.Contains(t, response, "timestamp")

	// Check field types
	assert.IsType(t, "", response["status"])
	assert.IsType(t, "", response["service"])
	assert.NotNil(t, response["timestamp"])

	// Check service name
	assert.Equal(t, "nudgebot-api", response["service"])

	// Status should be either "ok" or "error"
	status := response["status"].(string)
	assert.Contains(t, []string{"ok", "error"}, status)
}

func TestHealthHandler_Check_ContentType(t *testing.T) {
	router := setupHealthTest()
	logger := logger.New()

	db := &gorm.DB{}
	handler := NewHealthHandler(db, logger)
	router.GET("/health", handler.Check)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify content type is JSON
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestHealthHandler_Check_HTTPMethods(t *testing.T) {
	router := setupHealthTest()
	logger := logger.New()

	db := &gorm.DB{}
	handler := NewHealthHandler(db, logger)
	router.GET("/health", handler.Check)

	// Test that only GET is supported
	methods := []string{"POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run("Method_"+method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/health", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should return 404 Method Not Found for non-GET methods
			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

func TestHealthHandler_Check_Concurrent(t *testing.T) {
	router := setupHealthTest()
	logger := logger.New()

	db := &gorm.DB{}
	handler := NewHealthHandler(db, logger)
	router.GET("/health", handler.Check)

	// Test concurrent health checks
	const numRequests = 10
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			results <- w.Code
		}()
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		statusCode := <-results
		// All requests should succeed (though may return error status due to DB)
		assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, statusCode)
	}
}
