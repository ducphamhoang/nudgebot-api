package llm

import (
	"context"
	"time"

	"nudgebot-api/internal/common"
	"nudgebot-api/internal/config"
	"nudgebot-api/internal/events"

	"go.uber.org/zap"
)

// LLMService defines the interface for LLM operations
type LLMService interface {
	ParseTask(text string, userID common.UserID) (*LLMResponse, error)
	ValidateTask(parsedTask ParsedTask) error
	GetSuggestions(partialText string, userID common.UserID) ([]string, error)
}

// llmService implements the LLMService interface
type llmService struct {
	eventBus events.EventBus
	logger   *zap.Logger
	provider LLMProvider
}

// NewLLMService creates a new instance of LLMService
func NewLLMService(eventBus events.EventBus, logger *zap.Logger, config config.LLMConfig) LLMService {
	// Create Gemma provider
	provider := NewGemmaProvider(config, logger)

	service := &llmService{
		eventBus: eventBus,
		logger:   logger,
		provider: provider,
	}

	// Subscribe to relevant events
	service.setupEventSubscriptions()

	return service
}

// setupEventSubscriptions sets up event subscriptions for the LLM service
func (s *llmService) setupEventSubscriptions() {
	// Subscribe to MessageReceived events from the chatbot
	err := s.eventBus.Subscribe(events.TopicMessageReceived, s.handleMessageReceived)
	if err != nil {
		s.logger.Error("Failed to subscribe to MessageReceived events", zap.Error(err))
	}
}

// ParseTask parses natural language text into a structured task
func (s *llmService) ParseTask(text string, userID common.UserID) (*LLMResponse, error) {
	s.logger.Info("Parsing task",
		zap.String("text", text),
		zap.String("userID", string(userID)))

	// Create parse request
	parseRequest := ParseRequest{
		Text:    text,
		UserID:  userID,
		Context: nil, // Context can be added later for conversation flow
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Delegate to provider
	response, err := s.provider.ParseTask(ctx, parseRequest)
	if err != nil {
		s.logger.Error("Failed to parse task", zap.Error(err))
		return nil, err
	}

	return response, nil
}

// ValidateTask validates a parsed task for completeness and correctness
func (s *llmService) ValidateTask(parsedTask ParsedTask) error {
	s.logger.Info("Validating task", zap.String("title", parsedTask.Title))

	if parsedTask.Title == "" {
		return ParseError{
			Code:    ParseErrorCodeMissingTitle,
			Message: "Task title is required",
			Details: "Every task must have a non-empty title",
		}
	}

	if !parsedTask.Priority.IsValid() {
		return ParseError{
			Code:    ParseErrorCodeInvalidInput,
			Message: "Invalid priority level",
			Details: "Priority must be one of: low, medium, high, urgent",
		}
	}

	// TODO: Add more validation rules
	return nil
}

// GetSuggestions provides auto-completion suggestions for partial text
func (s *llmService) GetSuggestions(partialText string, userID common.UserID) ([]string, error) {
	s.logger.Info("Getting suggestions",
		zap.String("partialText", partialText),
		zap.String("userID", string(userID)))

	// TODO: Implement actual suggestion logic
	// For now, return mock suggestions
	suggestions := []string{
		partialText + " by tomorrow",
		partialText + " high priority",
		partialText + " weekly reminder",
	}

	return suggestions, nil
}

// handleMessageReceived handles MessageReceived events from the chatbot
func (s *llmService) handleMessageReceived(event events.MessageReceived) {
	s.logger.Info("Handling MessageReceived event",
		zap.String("correlationID", event.CorrelationID),
		zap.String("userID", event.UserID),
		zap.String("messageText", event.MessageText))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create parse request
	parseRequest := ParseRequest{
		Text:    event.MessageText,
		UserID:  common.UserID(event.UserID),
		Context: nil, // Context can be added later for conversation flow
	}

	// Parse the message text into a task using the provider
	response, err := s.provider.ParseTask(ctx, parseRequest)
	if err != nil {
		s.logger.Error("Failed to parse task", zap.Error(err))
		return
	}

	// Validate the parsed task
	if err := s.ValidateTask(response.ParsedTask); err != nil {
		s.logger.Error("Task validation failed", zap.Error(err))
		return
	}

	// Convert to events.ParsedTask format
	eventsParsedTask := events.ParsedTask{
		Title:       response.ParsedTask.Title,
		Description: response.ParsedTask.Description,
		DueDate:     response.ParsedTask.DueDate,
		Priority:    string(response.ParsedTask.Priority),
		Tags:        response.ParsedTask.Tags,
	}

	// Publish TaskParsed event
	taskParsedEvent := events.TaskParsed{
		Event:      events.NewEvent(),
		UserID:     event.UserID,
		ChatID:     event.ChatID, // Include ChatID from the original message
		ParsedTask: eventsParsedTask,
	}

	err = s.eventBus.Publish(events.TopicTaskParsed, taskParsedEvent)
	if err != nil {
		s.logger.Error("Failed to publish TaskParsed event", zap.Error(err))
	}
}
