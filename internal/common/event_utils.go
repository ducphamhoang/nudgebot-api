package common

import (
	"fmt"
	"strings"
	"time"

	"nudgebot-api/internal/events"

	"go.uber.org/zap"
)

// EventPublisher provides utilities for reliable event publishing
type EventPublisher struct {
	eventBus events.EventBus
	logger   *zap.Logger
}

// NewEventPublisher creates a new EventPublisher instance
func NewEventPublisher(eventBus events.EventBus, logger *zap.Logger) *EventPublisher {
	return &EventPublisher{
		eventBus: eventBus,
		logger:   logger,
	}
}

// PublishWithRetry publishes an event with retry logic and exponential backoff
func (p *EventPublisher) PublishWithRetry(topic string, event interface{}, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := p.eventBus.Publish(topic, event)
		if err == nil {
			if attempt > 0 {
				p.logger.Info("Event published successfully after retry",
					zap.String("topic", topic),
					zap.Int("attempt", attempt+1))
			}
			return nil
		}

		lastErr = err
		p.logger.Warn("Failed to publish event, retrying",
			zap.String("topic", topic),
			zap.Int("attempt", attempt+1),
			zap.Int("max_retries", maxRetries),
			zap.Error(err))

		if attempt < maxRetries {
			// Exponential backoff: 100ms, 200ms, 400ms, 800ms, etc.
			backoffDuration := time.Duration(100*(1<<attempt)) * time.Millisecond
			time.Sleep(backoffDuration)
		}
	}

	return fmt.Errorf("failed to publish event after %d retries: %w", maxRetries, lastErr)
}

// PublishBatch publishes multiple events efficiently
func (p *EventPublisher) PublishBatch(events map[string]interface{}) error {
	var errors []string

	for topic, event := range events {
		if err := p.eventBus.Publish(topic, event); err != nil {
			errorMsg := fmt.Sprintf("topic %s: %v", topic, err)
			errors = append(errors, errorMsg)
			p.logger.Error("Failed to publish event in batch",
				zap.String("topic", topic),
				zap.Error(err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch publish failed for %d events: %s", len(errors), strings.Join(errors, "; "))
	}

	p.logger.Debug("Batch publish completed successfully",
		zap.Int("event_count", len(events)))

	return nil
}

// PublishWithCorrelation publishes an event with correlation ID tracking
func (p *EventPublisher) PublishWithCorrelation(topic string, event interface{}, correlationID string) error {
	// Add correlation ID to event if it supports it
	if correlatedEvent, ok := event.(interface{ SetCorrelationID(string) }); ok {
		correlatedEvent.SetCorrelationID(correlationID)
	}

	err := p.eventBus.Publish(topic, event)
	if err != nil {
		p.logger.Error("Failed to publish correlated event",
			zap.String("topic", topic),
			zap.String("correlation_id", correlationID),
			zap.Error(err))
		return err
	}

	p.logger.Debug("Correlated event published",
		zap.String("topic", topic),
		zap.String("correlation_id", correlationID))

	return nil
}

// EventValidator provides validation utilities for events
type EventValidator struct {
	logger *zap.Logger
}

// NewEventValidator creates a new EventValidator instance
func NewEventValidator(logger *zap.Logger) *EventValidator {
	return &EventValidator{
		logger: logger,
	}
}

// ValidateEventStructure ensures events have required fields
func (v *EventValidator) ValidateEventStructure(event interface{}) error {
	switch e := event.(type) {
	case events.MessageReceived:
		if e.UserID == "" {
			return fmt.Errorf("MessageReceived: UserID is required")
		}
		if e.ChatID == "" {
			return fmt.Errorf("MessageReceived: ChatID is required")
		}
		if e.MessageText == "" {
			return fmt.Errorf("MessageReceived: MessageText is required")
		}

	case events.TaskParsed:
		if e.UserID == "" {
			return fmt.Errorf("TaskParsed: UserID is required")
		}
		if e.ParsedTask.Title == "" {
			return fmt.Errorf("TaskParsed: ParsedTask.Title is required")
		}
		if e.ParsedTask.Priority == "" {
			return fmt.Errorf("TaskParsed: ParsedTask.Priority is required")
		}

	case events.TaskListRequested:
		if e.UserID == "" {
			return fmt.Errorf("TaskListRequested: UserID is required")
		}
		if e.ChatID == "" {
			return fmt.Errorf("TaskListRequested: ChatID is required")
		}

	case events.TaskActionRequested:
		if e.UserID == "" {
			return fmt.Errorf("TaskActionRequested: UserID is required")
		}
		if e.ChatID == "" {
			return fmt.Errorf("TaskActionRequested: ChatID is required")
		}
		if e.TaskID == "" {
			return fmt.Errorf("TaskActionRequested: TaskID is required")
		}
		if e.Action == "" {
			return fmt.Errorf("TaskActionRequested: Action is required")
		}

	case events.ReminderDue:
		if e.TaskID == "" {
			return fmt.Errorf("ReminderDue: TaskID is required")
		}
		if e.UserID == "" {
			return fmt.Errorf("ReminderDue: UserID is required")
		}
		if e.ChatID == "" {
			return fmt.Errorf("ReminderDue: ChatID is required")
		}

	case events.TaskCreated:
		if e.TaskID == "" {
			return fmt.Errorf("TaskCreated: TaskID is required")
		}
		if e.UserID == "" {
			return fmt.Errorf("TaskCreated: UserID is required")
		}
		if e.Title == "" {
			return fmt.Errorf("TaskCreated: Title is required")
		}
		if e.Priority == "" {
			return fmt.Errorf("TaskCreated: Priority is required")
		}

	case events.TaskListResponse:
		if e.UserID == "" {
			return fmt.Errorf("TaskListResponse: UserID is required")
		}
		if e.ChatID == "" {
			return fmt.Errorf("TaskListResponse: ChatID is required")
		}

	case events.TaskActionResponse:
		if e.UserID == "" {
			return fmt.Errorf("TaskActionResponse: UserID is required")
		}
		if e.ChatID == "" {
			return fmt.Errorf("TaskActionResponse: ChatID is required")
		}
		if e.TaskID == "" {
			return fmt.Errorf("TaskActionResponse: TaskID is required")
		}
		if e.Action == "" {
			return fmt.Errorf("TaskActionResponse: Action is required")
		}

	default:
		return fmt.Errorf("unknown event type: %T", event)
	}

	return nil
}

// ValidateCorrelationID validates correlation ID format
func (v *EventValidator) ValidateCorrelationID(correlationID string) error {
	if correlationID == "" {
		return fmt.Errorf("correlation ID cannot be empty")
	}

	if len(correlationID) < 8 {
		return fmt.Errorf("correlation ID must be at least 8 characters long")
	}

	if len(correlationID) > 64 {
		return fmt.Errorf("correlation ID must be no more than 64 characters long")
	}

	return nil
}

// ValidateUserPermissions performs basic authorization checks
func (v *EventValidator) ValidateUserPermissions(userID, action string) error {
	if userID == "" {
		return fmt.Errorf("user ID is required for authorization")
	}

	// Basic validation - in a real system this would check against a permissions system
	allowedActions := map[string]bool{
		"create_task":   true,
		"list_tasks":    true,
		"complete_task": true,
		"delete_task":   true,
		"snooze_task":   true,
		"view_task":     true,
	}

	if !allowedActions[action] {
		return fmt.Errorf("action '%s' is not permitted", action)
	}

	return nil
}

// EventMetrics provides monitoring utilities for events
type EventMetrics struct {
	logger *zap.Logger
}

// NewEventMetrics creates a new EventMetrics instance
func NewEventMetrics(logger *zap.Logger) *EventMetrics {
	return &EventMetrics{
		logger: logger,
	}
}

// RecordEventPublished records metrics for published events
func (m *EventMetrics) RecordEventPublished(topic string, success bool) {
	if success {
		m.logger.Debug("Event published successfully",
			zap.String("topic", topic),
			zap.String("metric", "event_published"),
			zap.Bool("success", true))
	} else {
		m.logger.Warn("Event publication failed",
			zap.String("topic", topic),
			zap.String("metric", "event_published"),
			zap.Bool("success", false))
	}
}

// RecordEventProcessed records performance metrics for event processing
func (m *EventMetrics) RecordEventProcessed(topic string, duration time.Duration) {
	m.logger.Debug("Event processed",
		zap.String("topic", topic),
		zap.String("metric", "event_processed"),
		zap.Duration("processing_duration", duration))

	// Log warning for slow processing
	if duration > 5*time.Second {
		m.logger.Warn("Slow event processing detected",
			zap.String("topic", topic),
			zap.Duration("processing_duration", duration))
	}
}

// RecordEventError records error metrics
func (m *EventMetrics) RecordEventError(topic string, error error) {
	m.logger.Error("Event processing error",
		zap.String("topic", topic),
		zap.String("metric", "event_error"),
		zap.Error(error))
}

// Helper functions for event operations

// ExtractUserContext extracts user information from events
func ExtractUserContext(event interface{}) (userID, chatID string, err error) {
	switch e := event.(type) {
	case events.MessageReceived:
		return e.UserID, e.ChatID, nil
	case events.TaskParsed:
		return e.UserID, "", nil
	case events.TaskListRequested:
		return e.UserID, e.ChatID, nil
	case events.TaskActionRequested:
		return e.UserID, e.ChatID, nil
	case events.ReminderDue:
		return e.UserID, e.ChatID, nil
	case events.TaskCreated:
		return e.UserID, "", nil
	case events.TaskListResponse:
		return e.UserID, e.ChatID, nil
	case events.TaskActionResponse:
		return e.UserID, e.ChatID, nil
	default:
		return "", "", fmt.Errorf("unsupported event type for user context extraction: %T", event)
	}
}

// CreateCorrelationChain creates a correlation ID for linking related events
func CreateCorrelationChain(parentEvent interface{}) string {
	// Extract parent correlation ID if available
	var parentCorrelationID string

	switch e := parentEvent.(type) {
	case events.MessageReceived:
		parentCorrelationID = e.CorrelationID
	case events.TaskParsed:
		parentCorrelationID = e.CorrelationID
	case events.TaskListRequested:
		parentCorrelationID = e.CorrelationID
	case events.TaskActionRequested:
		parentCorrelationID = e.CorrelationID
	case events.ReminderDue:
		parentCorrelationID = e.CorrelationID
	default:
		// Generate new correlation ID if parent doesn't have one
		return fmt.Sprintf("chain_%d", time.Now().UnixNano())
	}

	// Create child correlation ID based on parent
	return fmt.Sprintf("%s_child_%d", parentCorrelationID, time.Now().UnixNano())
}

// FormatEventForLogging formats events for structured logging
func FormatEventForLogging(event interface{}) map[string]interface{} {
	logData := map[string]interface{}{
		"event_type": fmt.Sprintf("%T", event),
		"timestamp":  time.Now(),
	}

	switch e := event.(type) {
	case events.MessageReceived:
		logData["user_id"] = e.UserID
		logData["chat_id"] = e.ChatID
		logData["message_length"] = len(e.MessageText)
		logData["correlation_id"] = e.CorrelationID

	case events.TaskParsed:
		logData["user_id"] = e.UserID
		logData["task_title"] = e.ParsedTask.Title
		logData["task_priority"] = e.ParsedTask.Priority
		logData["has_due_date"] = e.ParsedTask.DueDate != nil
		logData["tag_count"] = len(e.ParsedTask.Tags)
		logData["correlation_id"] = e.CorrelationID

	case events.TaskListRequested:
		logData["user_id"] = e.UserID
		logData["chat_id"] = e.ChatID
		logData["correlation_id"] = e.CorrelationID

	case events.TaskActionRequested:
		logData["user_id"] = e.UserID
		logData["chat_id"] = e.ChatID
		logData["task_id"] = e.TaskID
		logData["action"] = e.Action
		logData["correlation_id"] = e.CorrelationID

	case events.TaskCreated:
		logData["user_id"] = e.UserID
		logData["task_id"] = e.TaskID
		logData["task_title"] = e.Title
		logData["task_priority"] = e.Priority
		logData["has_due_date"] = e.DueDate != nil
		logData["correlation_id"] = e.CorrelationID

	case events.TaskListResponse:
		logData["user_id"] = e.UserID
		logData["chat_id"] = e.ChatID
		logData["task_count"] = len(e.Tasks)
		logData["total_count"] = e.TotalCount
		logData["has_more"] = e.HasMore
		logData["correlation_id"] = e.CorrelationID

	case events.TaskActionResponse:
		logData["user_id"] = e.UserID
		logData["chat_id"] = e.ChatID
		logData["task_id"] = e.TaskID
		logData["action"] = e.Action
		logData["success"] = e.Success
		logData["correlation_id"] = e.CorrelationID
	}

	return logData
}

// IsRetryableError determines if event processing should be retried
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for temporary errors that can be retried
	errorString := strings.ToLower(err.Error())

	retryableErrors := []string{
		"timeout",
		"connection",
		"network",
		"temporary",
		"unavailable",
		"busy",
		"overloaded",
	}

	for _, retryableError := range retryableErrors {
		if strings.Contains(errorString, retryableError) {
			return true
		}
	}

	return false
}

// Event transformation utilities

// CreateTaskSummary creates a task summary from basic task information
func CreateTaskSummary(id, title, description, priority, status string, dueDate *time.Time, isOverdue bool) events.TaskSummary {
	return events.TaskSummary{
		ID:          id,
		Title:       title,
		Description: description,
		DueDate:     dueDate,
		Priority:    priority,
		Status:      status,
		IsOverdue:   isOverdue,
	}
}

// CreateTaskFromParsedData creates basic task data from parsed task information
func CreateTaskFromParsedData(parsedTask events.ParsedTask, userID string) map[string]interface{} {
	taskData := map[string]interface{}{
		"id":          string(NewID()),
		"user_id":     userID,
		"title":       parsedTask.Title,
		"description": parsedTask.Description,
		"due_date":    parsedTask.DueDate,
		"priority":    parsedTask.Priority,
		"status":      string(TaskStatusActive),
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
	}

	return taskData
}

// ExtractTaskMetadata extracts task metadata from events
func ExtractTaskMetadata(event interface{}) (taskID, userID string, err error) {
	switch e := event.(type) {
	case events.TaskCreated:
		return e.TaskID, e.UserID, nil
	case events.TaskActionRequested:
		return e.TaskID, e.UserID, nil
	case events.TaskActionResponse:
		return e.TaskID, e.UserID, nil
	case events.ReminderDue:
		return e.TaskID, e.UserID, nil
	default:
		return "", "", fmt.Errorf("event type %T does not contain task metadata", event)
	}
}
