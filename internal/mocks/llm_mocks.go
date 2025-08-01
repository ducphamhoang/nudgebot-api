package mocks

//go:generate mockgen -source=../llm/provider.go -destination=llm_mocks.go -package=mocks

import (
	"context"
	"errors"

	"nudgebot-api/internal/common"
	"nudgebot-api/internal/llm"
)

// MockLLMProvider provides a mock implementation of the LLMProvider interface
type MockLLMProvider struct {
	parseTaskResponse *llm.LLMResponse
	parseTaskError    error
	validateConnError error
	modelInfo         llm.ModelInfo
	parseTaskCalls    []llm.ParseRequest
	validateConnCalls int
	getModelInfoCalls int
}

// NewMockLLMProvider creates a new mock LLM provider
func NewMockLLMProvider() *MockLLMProvider {
	return &MockLLMProvider{
		parseTaskCalls: make([]llm.ParseRequest, 0),
		modelInfo: llm.ModelInfo{
			Name:         "mock-model",
			Version:      "1.0",
			Provider:     "Mock",
			Capabilities: []string{"task_parsing", "text_generation"},
			MaxTokens:    4096,
		},
	}
}

// ParseTask implements the LLMProvider interface
func (m *MockLLMProvider) ParseTask(ctx context.Context, req llm.ParseRequest) (*llm.LLMResponse, error) {
	m.parseTaskCalls = append(m.parseTaskCalls, req)

	if m.parseTaskError != nil {
		return nil, m.parseTaskError
	}

	if m.parseTaskResponse != nil {
		return m.parseTaskResponse, nil
	}

	// Default mock response
	return &llm.LLMResponse{
		ParsedTask: llm.ParsedTask{
			Title:       "Mock task: " + req.Text,
			Description: "Mock description for task parsing",
			Priority:    common.PriorityMedium,
			Tags:        []string{"mock", "test"},
		},
		Confidence: 0.85,
		Reasoning:  "Mock reasoning for test purposes",
	}, nil
}

// ValidateConnection implements the LLMProvider interface
func (m *MockLLMProvider) ValidateConnection(ctx context.Context) error {
	m.validateConnCalls++
	return m.validateConnError
}

// GetModelInfo implements the LLMProvider interface
func (m *MockLLMProvider) GetModelInfo() llm.ModelInfo {
	m.getModelInfoCalls++
	return m.modelInfo
}

// Test helper methods

// SetParseTaskResponse sets the response for ParseTask calls
func (m *MockLLMProvider) SetParseTaskResponse(response *llm.LLMResponse) {
	m.parseTaskResponse = response
}

// SetParseTaskError sets the error for ParseTask calls
func (m *MockLLMProvider) SetParseTaskError(err error) {
	m.parseTaskError = err
}

// SetValidateConnectionError sets the error for ValidateConnection calls
func (m *MockLLMProvider) SetValidateConnectionError(err error) {
	m.validateConnError = err
}

// SetModelInfo sets the model info to return
func (m *MockLLMProvider) SetModelInfo(info llm.ModelInfo) {
	m.modelInfo = info
}

// ExpectParseTaskCall expects a specific ParseTask call
func (m *MockLLMProvider) ExpectParseTaskCall(expectedReq llm.ParseRequest, response *llm.LLMResponse, err error) {
	m.parseTaskResponse = response
	m.parseTaskError = err
}

// GetParseTaskCalls returns all ParseTask calls made
func (m *MockLLMProvider) GetParseTaskCalls() []llm.ParseRequest {
	return m.parseTaskCalls
}

// GetValidateConnectionCallCount returns the number of ValidateConnection calls
func (m *MockLLMProvider) GetValidateConnectionCallCount() int {
	return m.validateConnCalls
}

// GetModelInfoCallCount returns the number of GetModelInfo calls
func (m *MockLLMProvider) GetModelInfoCallCount() int {
	return m.getModelInfoCalls
}

// Reset clears all call history and resets the mock
func (m *MockLLMProvider) Reset() {
	m.parseTaskCalls = make([]llm.ParseRequest, 0)
	m.validateConnCalls = 0
	m.getModelInfoCalls = 0
	m.parseTaskResponse = nil
	m.parseTaskError = nil
	m.validateConnError = nil
}

// MockGemmaProvider provides a mock implementation for integration testing
type MockGemmaProvider struct {
	*MockLLMProvider
	simulateNetworkError bool
	simulateRateLimit    bool
	simulateAPIError     bool
}

// NewMockGemmaProvider creates a new mock Gemma provider for integration testing
func NewMockGemmaProvider() *MockGemmaProvider {
	return &MockGemmaProvider{
		MockLLMProvider: NewMockLLMProvider(),
	}
}

// ParseTask implements specialized behavior for Gemma provider testing
func (m *MockGemmaProvider) ParseTask(ctx context.Context, req llm.ParseRequest) (*llm.LLMResponse, error) {
	// Simulate various error conditions for testing
	if m.simulateNetworkError {
		return nil, llm.NewNetworkError("test_operation", "Simulated network error", errors.New("connection failed"))
	}

	if m.simulateRateLimit {
		return nil, llm.NewRateLimitError(60, "Simulated rate limit")
	}

	if m.simulateAPIError {
		return nil, llm.NewAPIError(500, "INTERNAL_ERROR", "Simulated API error", "Test error details")
	}

	// Delegate to base mock
	return m.MockLLMProvider.ParseTask(ctx, req)
}

// SimulateNetworkError enables network error simulation
func (m *MockGemmaProvider) SimulateNetworkError(enable bool) {
	m.simulateNetworkError = enable
}

// SimulateRateLimit enables rate limit simulation
func (m *MockGemmaProvider) SimulateRateLimit(enable bool) {
	m.simulateRateLimit = enable
}

// SimulateAPIError enables API error simulation
func (m *MockGemmaProvider) SimulateAPIError(enable bool) {
	m.simulateAPIError = enable
}

// Factory methods for common test scenarios

// CreateSuccessScenario creates a mock provider that returns successful responses
func CreateSuccessScenario() *MockLLMProvider {
	mock := NewMockLLMProvider()
	mock.SetParseTaskResponse(&llm.LLMResponse{
		ParsedTask: llm.ParsedTask{
			Title:       "Test Task",
			Description: "Test task description",
			Priority:    common.PriorityHigh,
			Tags:        []string{"test", "success"},
		},
		Confidence: 0.95,
		Reasoning:  "High confidence parse",
	})
	return mock
}

// CreateAPIErrorScenario creates a mock provider that simulates API errors
func CreateAPIErrorScenario() *MockLLMProvider {
	mock := NewMockLLMProvider()
	mock.SetParseTaskError(llm.NewAPIError(500, "INTERNAL_ERROR", "API error", "Server error"))
	return mock
}

// CreateNetworkErrorScenario creates a mock provider that simulates network errors
func CreateNetworkErrorScenario() *MockLLMProvider {
	mock := NewMockLLMProvider()
	mock.SetParseTaskError(llm.NewNetworkError("http_request", "Network error", errors.New("connection timeout")))
	return mock
}

// CreateInvalidJSONScenario creates a mock provider that simulates JSON parsing errors
func CreateInvalidJSONScenario() *MockLLMProvider {
	mock := NewMockLLMProvider()
	mock.SetParseTaskError(llm.NewExtendedParseError(
		llm.ParseErrorCodeInvalidInput,
		"Invalid JSON response",
		"Failed to parse response JSON",
		false,
	))
	return mock
}
