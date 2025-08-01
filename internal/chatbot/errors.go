package chatbot

import (
	"fmt"
	"net/http"
)

// ChatbotError defines the interface for chatbot-specific errors
type ChatbotError interface {
	error
	Code() string
	Message() string
	Temporary() bool
}

// TelegramAPIError represents errors from Telegram Bot API
type TelegramAPIError struct {
	Operation   string
	StatusCode  int
	APIError    string
	Description string
	RetryAfter  int
}

func (e TelegramAPIError) Error() string {
	return fmt.Sprintf("telegram API error during %s: %s (status: %d)", e.Operation, e.Description, e.StatusCode)
}

func (e TelegramAPIError) Code() string {
	return "TELEGRAM_API_ERROR"
}

func (e TelegramAPIError) Message() string {
	return e.Description
}

func (e TelegramAPIError) Temporary() bool {
	// Rate limiting and server errors are temporary
	return e.StatusCode == http.StatusTooManyRequests ||
		e.StatusCode >= http.StatusInternalServerError ||
		e.RetryAfter > 0
}

// WebhookParsingError represents errors when parsing webhook data
type WebhookParsingError struct {
	UpdateType string
	Details    string
	Cause      error
}

func (e WebhookParsingError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("webhook parsing error for %s: %s (caused by: %v)", e.UpdateType, e.Details, e.Cause)
	}
	return fmt.Sprintf("webhook parsing error for %s: %s", e.UpdateType, e.Details)
}

func (e WebhookParsingError) Code() string {
	return "WEBHOOK_PARSING_ERROR"
}

func (e WebhookParsingError) Message() string {
	return e.Details
}

func (e WebhookParsingError) Temporary() bool {
	return false // Parsing errors are typically not temporary
}

func (e WebhookParsingError) Unwrap() error {
	return e.Cause
}

// CommandProcessingError represents errors during command execution
type CommandProcessingError struct {
	Command string
	Reason  string
	UserID  string
	ChatID  string
	Cause   error
}

func (e CommandProcessingError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("command processing error for %s: %s (caused by: %v)", e.Command, e.Reason, e.Cause)
	}
	return fmt.Sprintf("command processing error for %s: %s", e.Command, e.Reason)
}

func (e CommandProcessingError) Code() string {
	return "COMMAND_PROCESSING_ERROR"
}

func (e CommandProcessingError) Message() string {
	return e.Reason
}

func (e CommandProcessingError) Temporary() bool {
	return false // Command errors are typically not temporary
}

func (e CommandProcessingError) Unwrap() error {
	return e.Cause
}

// ConfigurationError represents invalid bot configuration
type ConfigurationError struct {
	Field  string
	Reason string
	Value  string
}

func (e ConfigurationError) Error() string {
	return fmt.Sprintf("configuration error for field %s: %s (value: %s)", e.Field, e.Reason, e.Value)
}

func (e ConfigurationError) Code() string {
	return "CONFIGURATION_ERROR"
}

func (e ConfigurationError) Message() string {
	return e.Reason
}

func (e ConfigurationError) Temporary() bool {
	return false // Configuration errors are not temporary
}

// SessionError represents session management failures
type SessionError struct {
	Operation string
	UserID    string
	Reason    string
	Cause     error
}

func (e SessionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("session error during %s for user %s: %s (caused by: %v)", e.Operation, e.UserID, e.Reason, e.Cause)
	}
	return fmt.Sprintf("session error during %s for user %s: %s", e.Operation, e.UserID, e.Reason)
}

func (e SessionError) Code() string {
	return "SESSION_ERROR"
}

func (e SessionError) Message() string {
	return e.Reason
}

func (e SessionError) Temporary() bool {
	return false // Session errors are typically not temporary
}

func (e SessionError) Unwrap() error {
	return e.Cause
}

// Error wrapping utilities

// WrapTelegramError wraps an error as a TelegramAPIError
func WrapTelegramError(err error, operation string) error {
	if err == nil {
		return nil
	}

	return TelegramAPIError{
		Operation:   operation,
		StatusCode:  http.StatusInternalServerError,
		APIError:    "UNKNOWN_ERROR",
		Description: err.Error(),
	}
}

// WrapParsingError wraps an error as a WebhookParsingError
func WrapParsingError(err error, updateType string) error {
	if err == nil {
		return nil
	}

	return WebhookParsingError{
		UpdateType: updateType,
		Details:    "failed to parse webhook data",
		Cause:      err,
	}
}

// NewCommandError creates a new CommandProcessingError
func NewCommandError(command, reason, userID, chatID string) error {
	return CommandProcessingError{
		Command: command,
		Reason:  reason,
		UserID:  userID,
		ChatID:  chatID,
	}
}

// NewConfigurationError creates a new ConfigurationError
func NewConfigurationError(field, reason, value string) error {
	return ConfigurationError{
		Field:  field,
		Reason: reason,
		Value:  value,
	}
}

// NewSessionError creates a new SessionError
func NewSessionError(operation, userID, reason string) error {
	return SessionError{
		Operation: operation,
		UserID:    userID,
		Reason:    reason,
	}
}

// Error classification helpers

// IsRetryableError determines if an error should be retried
func IsRetryableError(err error) bool {
	if chatbotErr, ok := err.(ChatbotError); ok {
		return chatbotErr.Temporary()
	}
	return false
}

// IsConfigurationError determines if an error is configuration-related
func IsConfigurationError(err error) bool {
	_, ok := err.(ConfigurationError)
	return ok
}

// IsTemporaryError determines if an error is temporary
func IsTemporaryError(err error) bool {
	if chatbotErr, ok := err.(ChatbotError); ok {
		return chatbotErr.Temporary()
	}
	return false
}

// IsTelegramAPIError determines if an error is from Telegram API
func IsTelegramAPIError(err error) bool {
	_, ok := err.(TelegramAPIError)
	return ok
}

// IsWebhookParsingError determines if an error is from webhook parsing
func IsWebhookParsingError(err error) bool {
	_, ok := err.(WebhookParsingError)
	return ok
}

// HTTP status code mapping for Telegram API errors
var telegramErrorCodes = map[int]string{
	400: "BAD_REQUEST",
	401: "UNAUTHORIZED",
	403: "FORBIDDEN",
	404: "NOT_FOUND",
	409: "CONFLICT",
	429: "TOO_MANY_REQUESTS",
	500: "INTERNAL_SERVER_ERROR",
	502: "BAD_GATEWAY",
	503: "SERVICE_UNAVAILABLE",
	504: "GATEWAY_TIMEOUT",
}

// GetTelegramErrorCode returns error code for HTTP status
func GetTelegramErrorCode(statusCode int) string {
	if code, exists := telegramErrorCodes[statusCode]; exists {
		return code
	}
	return "UNKNOWN_ERROR"
}
