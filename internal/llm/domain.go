package llm

import (
	"time"

	"nudgebot-api/internal/common"
)

// ParseRequest represents a request to parse natural language text into a task
type ParseRequest struct {
	Text    string        `json:"text" validate:"required"`
	UserID  common.UserID `json:"user_id" validate:"required"`
	Context *ContextData  `json:"context,omitempty"`
}

// ParsedTask represents a task that has been parsed from natural language
type ParsedTask struct {
	Title       string          `json:"title" validate:"required"`
	Description string          `json:"description"`
	DueDate     *time.Time      `json:"due_date,omitempty"`
	Priority    common.Priority `json:"priority" validate:"required"`
	Tags        []string        `json:"tags"`
}

// LLMResponse represents the response from the LLM service
type LLMResponse struct {
	ParsedTask ParsedTask `json:"parsed_task" validate:"required"`
	Confidence float64    `json:"confidence" validate:"min=0,max=1"`
	Reasoning  string     `json:"reasoning"`
}

// ParseError represents an error that occurred during parsing
type ParseError struct {
	Code    string `json:"code" validate:"required"`
	Message string `json:"message" validate:"required"`
	Details string `json:"details"`
}

// Error implements the error interface
func (pe ParseError) Error() string {
	return pe.Message
}

// ContextData represents conversation context for better parsing
type ContextData struct {
	PreviousTasks    []ParsedTask `json:"previous_tasks"`
	ConversationFlow []string     `json:"conversation_flow"`
	UserPreferences  UserPrefs    `json:"user_preferences"`
}

// UserPrefs represents user preferences for task parsing
type UserPrefs struct {
	DefaultPriority common.Priority `json:"default_priority"`
	TimeZone        string          `json:"timezone"`
	DateFormat      string          `json:"date_format"`
	CommonTags      []string        `json:"common_tags"`
}

// Confidence levels
const (
	ConfidenceHigh   = 0.8
	ConfidenceMedium = 0.6
	ConfidenceLow    = 0.4
)

// Parse error codes
const (
	ParseErrorCodeInvalidInput       = "INVALID_INPUT"
	ParseErrorCodeAmbiguousTask      = "AMBIGUOUS_TASK"
	ParseErrorCodeMissingTitle       = "MISSING_TITLE"
	ParseErrorCodeInvalidDate        = "INVALID_DATE"
	ParseErrorCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// IsHighConfidence checks if the confidence level is high
func (r LLMResponse) IsHighConfidence() bool {
	return r.Confidence >= ConfidenceHigh
}

// IsMediumConfidence checks if the confidence level is medium
func (r LLMResponse) IsMediumConfidence() bool {
	return r.Confidence >= ConfidenceMedium && r.Confidence < ConfidenceHigh
}

// IsLowConfidence checks if the confidence level is low
func (r LLMResponse) IsLowConfidence() bool {
	return r.Confidence < ConfidenceMedium
}
