package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"nudgebot-api/internal/common"
	"nudgebot-api/internal/config"

	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
)

// GemmaProvider implements the LLMProvider interface for Google Gemma API
type GemmaProvider struct {
	config     config.LLMConfig
	logger     *zap.Logger
	httpClient *http.Client
	backoff    backoff.BackOff
}

// GemmaRequest represents the request structure for Gemma API
type GemmaRequest struct {
	Contents         []GemmaContent        `json:"contents"`
	SafetySettings   []GemmaSafetySetting  `json:"safetySettings,omitempty"`
	GenerationConfig GemmaGenerationConfig `json:"generationConfig,omitempty"`
}

// GemmaContent represents content in the Gemma request
type GemmaContent struct {
	Parts []GemmaPart `json:"parts"`
	Role  string      `json:"role,omitempty"`
}

// GemmaPart represents a part of the content
type GemmaPart struct {
	Text string `json:"text"`
}

// GemmaSafetySetting represents safety settings for content generation
type GemmaSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// GemmaGenerationConfig represents generation configuration
type GemmaGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

// GemmaResponse represents the response from Gemma API
type GemmaResponse struct {
	Candidates []GemmaCandidate `json:"candidates"`
	Error      *GemmaError      `json:"error,omitempty"`
}

// GemmaCandidate represents a candidate response
type GemmaCandidate struct {
	Content       GemmaContent        `json:"content"`
	FinishReason  string              `json:"finishReason"`
	SafetyRatings []GemmaSafetyRating `json:"safetyRatings,omitempty"`
}

// GemmaSafetyRating represents safety rating for content
type GemmaSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// GemmaError represents an error from the API
type GemmaError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// NewGemmaProvider creates a new GemmaProvider instance
func NewGemmaProvider(config config.LLMConfig, logger *zap.Logger) *GemmaProvider {
	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}

	// Create exponential backoff strategy
	backoffStrategy := backoff.NewExponentialBackOff()
	backoffStrategy.InitialInterval = 1 * time.Second
	backoffStrategy.MaxInterval = 30 * time.Second
	backoffStrategy.MaxElapsedTime = 2 * time.Minute
	backoffStrategy.Multiplier = 2.0

	// Limit retries based on config
	backoffWithRetry := backoff.WithMaxRetries(backoffStrategy, uint64(config.MaxRetries))

	return &GemmaProvider{
		config:     config,
		logger:     logger,
		httpClient: httpClient,
		backoff:    backoffWithRetry,
	}
}

// ParseTask implements the LLMProvider interface
func (p *GemmaProvider) ParseTask(ctx context.Context, req ParseRequest) (*LLMResponse, error) {
	p.logger.Info("Parsing task with Gemma API",
		zap.String("text", req.Text),
		zap.String("userID", string(req.UserID)))

	// Validate configuration
	if p.config.APIKey == "" {
		return nil, NewConfigurationError("api_key", "API key is required", "Gemma API key must be configured")
	}

	// Build the prompt
	prompt := p.buildPrompt(req)

	// Create the request
	gemmaReq := GemmaRequest{
		Contents: []GemmaContent{
			{
				Parts: []GemmaPart{
					{Text: prompt},
				},
				Role: "user",
			},
		},
		GenerationConfig: GemmaGenerationConfig{
			Temperature:     0.1, // Low temperature for consistent structured output
			TopK:            40,
			TopP:            0.95,
			MaxOutputTokens: 1024,
		},
		SafetySettings: []GemmaSafetySetting{
			{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		},
	}

	// Execute with retry logic
	var response *LLMResponse
	var err error

	operation := func() error {
		response, err = p.callAPI(ctx, gemmaReq)
		if err != nil {
			// Check if error is retryable
			if IsRetryable(err) {
				p.logger.Warn("Retryable error occurred, will retry",
					zap.Error(err))
				return err
			}
			// Non-retryable error, stop retrying
			return backoff.Permanent(err)
		}
		return nil
	}

	err = backoff.Retry(operation, backoff.WithContext(p.backoff, ctx))
	if err != nil {
		p.logger.Error("Failed to parse task after retries",
			zap.Error(err),
			zap.String("text", req.Text))
		return nil, err
	}

	return response, nil
}

// ValidateConnection implements the LLMProvider interface
func (p *GemmaProvider) ValidateConnection(ctx context.Context) error {
	// Simple validation request
	testReq := ParseRequest{
		Text:   "test connection",
		UserID: "test",
	}

	_, err := p.ParseTask(ctx, testReq)
	if err != nil {
		return fmt.Errorf("connection validation failed: %w", err)
	}

	return nil
}

// GetModelInfo implements the LLMProvider interface
func (p *GemmaProvider) GetModelInfo() ModelInfo {
	return ModelInfo{
		Name:     p.config.Model,
		Version:  "2.0",
		Provider: "Google",
		Capabilities: []string{
			"text_generation",
			"task_parsing",
			"json_output",
			"natural_language_understanding",
		},
		MaxTokens: 8192,
	}
}

// buildPrompt creates a structured prompt for the Gemma API
func (p *GemmaProvider) buildPrompt(req ParseRequest) string {
	prompt := `You are a task parsing assistant. Parse the following natural language text into a structured task.

IMPORTANT: You must respond with valid JSON only, no other text or explanations.

The JSON must have this exact structure:
{
  "title": "clear, concise task title",
  "description": "detailed description if available, empty string if not",
  "due_date": "ISO 8601 date string if a date is mentioned, null if not",
  "priority": "low|medium|high|urgent",
  "tags": ["array", "of", "relevant", "tags"],
  "confidence": 0.85,
  "reasoning": "brief explanation of parsing decisions"
}

Priority guidelines:
- "urgent": explicitly urgent/critical/ASAP
- "high": important, has deadline within days
- "medium": normal task, may have loose deadline
- "low": minor task, no urgency indicators

Extract tags from context, topics, or task categories mentioned.

Text to parse: "` + req.Text + `"

Respond with JSON only:`

	return prompt
}

// callAPI makes the actual HTTP request to the Gemma API
func (p *GemmaProvider) callAPI(ctx context.Context, req GemmaRequest) (*LLMResponse, error) {
	// Marshal request
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, NewExtendedParseError(
			ParseErrorCodeInvalidInput,
			"Failed to marshal request",
			err.Error(),
			false,
		)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.APIEndpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, NewNetworkError("create_request", "Failed to create HTTP request", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", p.config.APIKey)

	// Make the request
	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, NewNetworkError("http_request", "Failed to make HTTP request", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, NewNetworkError("read_response", "Failed to read response body", err)
	}

	// Handle HTTP errors
	if httpResp.StatusCode != http.StatusOK {
		return nil, p.handleHTTPError(httpResp.StatusCode, responseBody)
	}

	// Parse response
	return p.parseGemmaResponse(responseBody)
}

// parseGemmaResponse parses the Gemma API response and extracts the task data
func (p *GemmaProvider) parseGemmaResponse(responseBody []byte) (*LLMResponse, error) {
	var gemmaResp GemmaResponse
	if err := json.Unmarshal(responseBody, &gemmaResp); err != nil {
		return nil, NewExtendedParseError(
			ParseErrorCodeInvalidInput,
			"Failed to parse API response",
			err.Error(),
			false,
		)
	}

	// Check for API errors
	if gemmaResp.Error != nil {
		return nil, NewAPIError(
			gemmaResp.Error.Code,
			gemmaResp.Error.Status,
			gemmaResp.Error.Message,
			"API returned error response",
		)
	}

	// Check for candidates
	if len(gemmaResp.Candidates) == 0 {
		return nil, NewExtendedParseError(
			ParseErrorCodeServiceUnavailable,
			"No candidates in API response",
			"Gemma API returned empty candidates array",
			true,
		)
	}

	candidate := gemmaResp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, NewExtendedParseError(
			ParseErrorCodeServiceUnavailable,
			"No content parts in API response",
			"Gemma API returned empty content parts",
			true,
		)
	}

	// Extract JSON from the response text
	responseText := candidate.Content.Parts[0].Text
	jsonStr := p.extractJSON(responseText)

	// Parse the extracted JSON
	var taskData struct {
		Title       string     `json:"title"`
		Description string     `json:"description"`
		DueDate     *time.Time `json:"due_date"`
		Priority    string     `json:"priority"`
		Tags        []string   `json:"tags"`
		Confidence  float64    `json:"confidence"`
		Reasoning   string     `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &taskData); err != nil {
		return nil, NewExtendedParseError(
			ParseErrorCodeInvalidInput,
			"Failed to parse task JSON from response",
			fmt.Sprintf("Response text: %s, Error: %v", responseText, err),
			false,
		)
	}

	// Validate and convert priority
	var priority common.Priority
	switch strings.ToLower(taskData.Priority) {
	case "low":
		priority = common.PriorityLow
	case "medium":
		priority = common.PriorityMedium
	case "high":
		priority = common.PriorityHigh
	case "urgent":
		priority = common.PriorityUrgent
	default:
		priority = common.PriorityMedium // Default fallback
	}

	// Create the response
	response := &LLMResponse{
		ParsedTask: ParsedTask{
			Title:       taskData.Title,
			Description: taskData.Description,
			DueDate:     taskData.DueDate,
			Priority:    priority,
			Tags:        taskData.Tags,
		},
		Confidence: taskData.Confidence,
		Reasoning:  taskData.Reasoning,
	}

	return response, nil
}

// extractJSON extracts JSON from response text that might contain other content
func (p *GemmaProvider) extractJSON(text string) string {
	// Look for JSON object boundaries
	start := strings.Index(text, "{")
	if start == -1 {
		return text // Return as-is if no opening brace found
	}

	// Find the matching closing brace
	braceCount := 0
	for i := start; i < len(text); i++ {
		if text[i] == '{' {
			braceCount++
		} else if text[i] == '}' {
			braceCount--
			if braceCount == 0 {
				return text[start : i+1]
			}
		}
	}

	// If no proper JSON found, return from first brace to end
	return text[start:]
}

// handleHTTPError creates appropriate error based on HTTP status code
func (p *GemmaProvider) handleHTTPError(statusCode int, responseBody []byte) error {
	var errorMsg string = "Unknown error"
	var errorCode string = ErrorCodeUnknown

	// Try to parse error response
	var gemmaResp GemmaResponse
	if err := json.Unmarshal(responseBody, &gemmaResp); err == nil && gemmaResp.Error != nil {
		errorMsg = gemmaResp.Error.Message
		errorCode = gemmaResp.Error.Status
	} else {
		errorMsg = string(responseBody)
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return NewAPIError(statusCode, ErrorCodeInvalidAPIKey, "Invalid API key", errorMsg)
	case http.StatusForbidden:
		return NewAPIError(statusCode, ErrorCodeInsufficientQuota, "Insufficient quota or permissions", errorMsg)
	case http.StatusNotFound:
		return NewAPIError(statusCode, ErrorCodeModelNotFound, "Model not found", errorMsg)
	case http.StatusRequestEntityTooLarge:
		return NewAPIError(statusCode, ErrorCodeRequestTooLarge, "Request too large", errorMsg)
	case http.StatusTooManyRequests:
		retryAfter := 60 // Default to 60 seconds
		// Try to parse Retry-After header would go here in a real implementation
		return NewRateLimitError(retryAfter, errorMsg)
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return NewAPIError(statusCode, ErrorCodeServiceUnavailable, "Service unavailable", errorMsg)
	default:
		return NewAPIError(statusCode, errorCode, errorMsg, string(responseBody))
	}
}
