package mocks

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// HTTPClientInterface defines the interface for HTTP clients used by the application
type HTTPClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

// MockHTTPClient is a mock implementation of the HTTPClientInterface
type MockHTTPClient struct {
	mu sync.RWMutex

	// Request tracking
	requests    []CapturedRequest
	callCounts  map[string]int

	// Response configuration
	responses       map[string]*http.Response
	errors          map[string]error
	defaultResponse *http.Response
	defaultError    error

	// Behavior simulation
	delays      map[string]time.Duration
	callbackFn  func(*http.Request) (*http.Response, error)
}

// CapturedRequest represents a captured HTTP request for verification
type CapturedRequest struct {
	Method    string
	URL       *url.URL
	Headers   http.Header
	Body      []byte
	Timestamp time.Time
}

// NewMockHTTPClient creates a new mock HTTP client
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		requests:   make([]CapturedRequest, 0),
		callCounts: make(map[string]int),
		responses:  make(map[string]*http.Response),
		errors:     make(map[string]error),
		delays:     make(map[string]time.Duration),
	}
}

// Do implements the HTTPClientInterface.Do method
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Capture the request
	body := []byte{}
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		body = bodyBytes
		// Restore the body for potential reuse
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	capturedReq := CapturedRequest{
		Method:    req.Method,
		URL:       req.URL,
		Headers:   req.Header.Clone(),
		Body:      body,
		Timestamp: time.Now(),
	}
	m.requests = append(m.requests, capturedReq)

	// Track call counts
	key := fmt.Sprintf("%s:%s", req.Method, req.URL.Path)
	m.callCounts[key]++
	m.callCounts["total"]++

	// Simulate delay if configured
	if delay, exists := m.delays[key]; exists {
		time.Sleep(delay)
	} else if delay, exists := m.delays["default"]; exists {
		time.Sleep(delay)
	}

	// Use callback function if provided
	if m.callbackFn != nil {
		return m.callbackFn(req)
	}

	// Check for specific error for this request
	if err, exists := m.errors[key]; exists {
		return nil, err
	}

	// Check for default error
	if m.defaultError != nil {
		return nil, m.defaultError
	}

	// Check for specific response for this request
	if resp, exists := m.responses[key]; exists {
		return resp, nil
	}

	// Return default response or create one
	if m.defaultResponse != nil {
		return m.defaultResponse, nil
	}

	// Create a default 200 OK response
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("{}")),
		Request:    req,
	}, nil
}

// ==============================================================================
// Configuration Methods for Setting Up Mock Responses
// ==============================================================================

// SetResponse configures a specific response for a URL pattern
func (m *MockHTTPClient) SetResponse(method, urlPattern string, statusCode int, body string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", method, urlPattern)
	m.responses[key] = &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// SetError configures an error for a specific URL pattern
func (m *MockHTTPClient) SetError(method, urlPattern string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", method, urlPattern)
	m.errors[key] = err
}

// SetDefaultResponse configures a default response for all requests
func (m *MockHTTPClient) SetDefaultResponse(statusCode int, body string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.defaultResponse = &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// SetDefaultError configures a default error for all requests
func (m *MockHTTPClient) SetDefaultError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultError = err
}

// SetCallback configures a callback function to handle requests dynamically
func (m *MockHTTPClient) SetCallback(fn func(*http.Request) (*http.Response, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbackFn = fn
}

// ==============================================================================
// Request Tracking and Verification Methods
// ==============================================================================

// GetRequests returns all captured requests
func (m *MockHTTPClient) GetRequests() []CapturedRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions
	requests := make([]CapturedRequest, len(m.requests))
	copy(requests, m.requests)
	return requests
}

// GetRequestCount returns the total number of requests made
func (m *MockHTTPClient) GetRequestCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCounts["total"]
}

// GetRequestCountForURL returns the number of requests made to a specific URL pattern
func (m *MockHTTPClient) GetRequestCountForURL(method, urlPattern string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := fmt.Sprintf("%s:%s", method, urlPattern)
	return m.callCounts[key]
}

// GetLastRequest returns the last captured request
func (m *MockHTTPClient) GetLastRequest() *CapturedRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.requests) == 0 {
		return nil
	}
	return &m.requests[len(m.requests)-1]
}

// GetRequestsForURL returns all requests made to a specific URL pattern
func (m *MockHTTPClient) GetRequestsForURL(method, urlPattern string) []CapturedRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matchingRequests []CapturedRequest
	for _, req := range m.requests {
		if req.Method == method && strings.Contains(req.URL.Path, urlPattern) {
			matchingRequests = append(matchingRequests, req)
		}
	}
	return matchingRequests
}

// ==============================================================================
// Simulation Methods for Testing Different Scenarios
// ==============================================================================

// SimulateTimeout configures the client to simulate a timeout error
func (m *MockHTTPClient) SimulateTimeout(method, urlPattern string) {
	m.SetError(method, urlPattern, fmt.Errorf("timeout: context deadline exceeded"))
}

// SimulateNetworkError configures the client to simulate a network error
func (m *MockHTTPClient) SimulateNetworkError(method, urlPattern string) {
	m.SetError(method, urlPattern, fmt.Errorf("network error: connection refused"))
}

// SimulateRateLimit configures the client to simulate a rate limit response
func (m *MockHTTPClient) SimulateRateLimit(method, urlPattern string) {
	rateLimitBody := `{"error":{"code":429,"message":"Rate limit exceeded","status":"RESOURCE_EXHAUSTED"}}`
	m.SetResponse(method, urlPattern, http.StatusTooManyRequests, rateLimitBody)
}

// SimulateDelay configures the client to add delays to specific requests
func (m *MockHTTPClient) SimulateDelay(method, urlPattern string, delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", method, urlPattern)
	m.delays[key] = delay
}

// SimulateDefaultDelay configures the client to add delays to all requests
func (m *MockHTTPClient) SimulateDefaultDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delays["default"] = delay
}

// ==============================================================================
// Factory Methods for Common Test Scenarios
// ==============================================================================

// CreateGemmaSuccessClient creates a mock client for successful Gemma API responses
func CreateGemmaSuccessClient() *MockHTTPClient {
	client := NewMockHTTPClient()
	
	successResponse := `{
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
	}`
	
	client.SetResponse("POST", "/v1beta/models/gemma-2-27b-it:generateContent", http.StatusOK, successResponse)
	return client
}

// CreateGemmaErrorClient creates a mock client for Gemma API error responses
func CreateGemmaErrorClient() *MockHTTPClient {
	client := NewMockHTTPClient()
	
	errorResponse := `{
		"error": {
			"code": 400,
			"message": "Invalid request format",
			"status": "INVALID_ARGUMENT"
		}
	}`
	
	client.SetResponse("POST", "/v1beta/models/gemma-2-27b-it:generateContent", http.StatusBadRequest, errorResponse)
	return client
}

// CreateNetworkErrorClient creates a mock client that simulates network errors
func CreateNetworkErrorClient() *MockHTTPClient {
	client := NewMockHTTPClient()
	client.SetDefaultError(fmt.Errorf("network error: connection timeout"))
	return client
}

// CreateTimeoutClient creates a mock client that simulates timeout errors
func CreateTimeoutClient() *MockHTTPClient {
	client := NewMockHTTPClient()
	client.SetDefaultError(fmt.Errorf("timeout: context deadline exceeded"))
	return client
}

// ==============================================================================
// Assertion Helpers for Testing
// ==============================================================================

// AssertRequestMade verifies that a request was made to a specific URL
func (m *MockHTTPClient) AssertRequestMade(method, urlPattern string) bool {
	return m.GetRequestCountForURL(method, urlPattern) > 0
}

// AssertRequestBody verifies that a request was made with specific body content
func (m *MockHTTPClient) AssertRequestBody(method, urlPattern string, expectedBodyContent string) bool {
	requests := m.GetRequestsForURL(method, urlPattern)
	for _, req := range requests {
		if strings.Contains(string(req.Body), expectedBodyContent) {
			return true
		}
	}
	return false
}

// AssertRequestHeaders verifies that a request was made with specific headers
func (m *MockHTTPClient) AssertRequestHeaders(method, urlPattern string, expectedHeaders map[string]string) bool {
	requests := m.GetRequestsForURL(method, urlPattern)
	for _, req := range requests {
		allMatch := true
		for key, expectedValue := range expectedHeaders {
			if req.Headers.Get(key) != expectedValue {
				allMatch = false
				break
			}
		}
		if allMatch {
			return true
		}
	}
	return false
}

// Reset clears all request history and configurations
func (m *MockHTTPClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requests = make([]CapturedRequest, 0)
	m.callCounts = make(map[string]int)
	m.responses = make(map[string]*http.Response)
	m.errors = make(map[string]error)
	m.delays = make(map[string]time.Duration)
	m.defaultResponse = nil
	m.defaultError = nil
	m.callbackFn = nil
}