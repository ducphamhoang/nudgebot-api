package llm

import (
	"fmt"
	"net/http"
)

// LLMError defines the interface for LLM-specific errors
type LLMError interface {
	error
	Code() string    // Error code for categorization
	Message() string // Human-readable error message
	Temporary() bool // Whether the error is temporary and retryable
}

// APIError represents errors from the Gemma API
type APIError struct {
	HTTPStatus int    `json:"http_status"`
	ErrorCode  string `json:"error_code"`
	ErrorMsg   string `json:"error_message"`
	Details    string `json:"details"`
	Retryable  bool   `json:"retryable"`
}

func (e APIError) Error() string {
	return fmt.Sprintf("API error (HTTP %d): %s - %s", e.HTTPStatus, e.ErrorCode, e.ErrorMsg)
}

func (e APIError) Code() string {
	return e.ErrorCode
}

func (e APIError) Message() string {
	return e.ErrorMsg
}

func (e APIError) Temporary() bool {
	return e.Retryable
}

// NetworkError represents connection and timeout issues
type NetworkError struct {
	Operation string `json:"operation"`
	ErrorMsg  string `json:"error_message"`
	Wrapped   error  `json:"-"`
}

func (e NetworkError) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("network error during %s: %s (wrapped: %v)", e.Operation, e.ErrorMsg, e.Wrapped)
	}
	return fmt.Sprintf("network error during %s: %s", e.Operation, e.ErrorMsg)
}

func (e NetworkError) Code() string {
	return "NETWORK_ERROR"
}

func (e NetworkError) Message() string {
	return e.ErrorMsg
}

func (e NetworkError) Temporary() bool {
	return true // Network errors are generally retryable
}

func (e NetworkError) Unwrap() error {
	return e.Wrapped
}

// ExtendedParseError extends the existing ParseError with LLM-specific functionality
type ExtendedParseError struct {
	ParseError
	HTTPStatus int  `json:"http_status,omitempty"`
	Retryable  bool `json:"retryable"`
}

func (e ExtendedParseError) Temporary() bool {
	return e.Retryable
}

// ConfigurationError represents invalid configuration
type ConfigurationError struct {
	Field    string `json:"field"`
	ErrorMsg string `json:"error_message"`
	Details  string `json:"details"`
}

func (e ConfigurationError) Error() string {
	return fmt.Sprintf("configuration error for field '%s': %s", e.Field, e.ErrorMsg)
}

func (e ConfigurationError) Code() string {
	return "CONFIGURATION_ERROR"
}

func (e ConfigurationError) Message() string {
	return e.ErrorMsg
}

func (e ConfigurationError) Temporary() bool {
	return false // Configuration errors are not retryable
}

// RateLimitError represents API rate limiting
type RateLimitError struct {
	RetryAfter int    `json:"retry_after_seconds"`
	ErrorMsg   string `json:"error_message"`
}

func (e RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded: %s (retry after %d seconds)", e.ErrorMsg, e.RetryAfter)
}

func (e RateLimitError) Code() string {
	return "RATE_LIMIT_EXCEEDED"
}

func (e RateLimitError) Message() string {
	return e.ErrorMsg
}

func (e RateLimitError) Temporary() bool {
	return true // Rate limit errors are retryable after waiting
}

// Error creation helpers

// NewAPIError creates a new API error with appropriate retry logic
func NewAPIError(httpStatus int, errorCode, message, details string) APIError {
	retryable := isRetryableHTTPStatus(httpStatus)
	return APIError{
		HTTPStatus: httpStatus,
		ErrorCode:  errorCode,
		ErrorMsg:   message,
		Details:    details,
		Retryable:  retryable,
	}
}

// NewNetworkError creates a new network error
func NewNetworkError(operation, message string, wrapped error) NetworkError {
	return NetworkError{
		Operation: operation,
		ErrorMsg:  message,
		Wrapped:   wrapped,
	}
}

// NewExtendedParseError creates an extended parse error
func NewExtendedParseError(code, message, details string, retryable bool) ExtendedParseError {
	return ExtendedParseError{
		ParseError: ParseError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Retryable: retryable,
	}
}

// NewConfigurationError creates a new configuration error
func NewConfigurationError(field, message, details string) ConfigurationError {
	return ConfigurationError{
		Field:    field,
		ErrorMsg: message,
		Details:  details,
	}
}

// NewRateLimitError creates a new rate limit error
func NewRateLimitError(retryAfter int, message string) RateLimitError {
	return RateLimitError{
		RetryAfter: retryAfter,
		ErrorMsg:   message,
	}
}

// Error classification helpers

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if llmErr, ok := err.(LLMError); ok {
		return llmErr.Temporary()
	}
	return false
}

// IsTemporary is an alias for IsRetryable for consistency with standard library
func IsTemporary(err error) bool {
	return IsRetryable(err)
}

// isRetryableHTTPStatus determines if an HTTP status code indicates a retryable error
func isRetryableHTTPStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

// Error constants for common scenarios
const (
	ErrorCodeInvalidAPIKey      = "INVALID_API_KEY"
	ErrorCodeModelNotFound      = "MODEL_NOT_FOUND"
	ErrorCodeInsufficientQuota  = "INSUFFICIENT_QUOTA"
	ErrorCodeRequestTooLarge    = "REQUEST_TOO_LARGE"
	ErrorCodeInvalidRequest     = "INVALID_REQUEST"
	ErrorCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrorCodeTimeout            = "TIMEOUT"
	ErrorCodeUnknown            = "UNKNOWN_ERROR"
)
