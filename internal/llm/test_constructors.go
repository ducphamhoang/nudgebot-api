//go:build test || integration

package llm

import (
	"context"
	"time"

	"nudgebot-api/internal/common"
	"nudgebot-api/internal/events"

	"go.uber.org/zap"
)

// NewLLMServiceWithProvider creates an LLMService with a custom provider for testing
func NewLLMServiceWithProvider(eventBus events.EventBus, logger *zap.Logger, provider LLMProvider) LLMService {
	service := &llmService{
		eventBus: eventBus,
		logger:   logger,
		provider: provider,
	}

	// Subscribe to relevant events
	service.setupEventSubscriptions()

	return service
}

// StubLLMProvider is a stub implementation of LLMProvider for testing
type StubLLMProvider struct {
	logger *zap.Logger
}

// NewStubLLMProvider creates a new stub LLM provider for testing
func NewStubLLMProvider(logger *zap.Logger) LLMProvider {
	return &StubLLMProvider{
		logger: logger,
	}
}

// ParseTask implements LLMProvider interface with deterministic stub behavior
func (s *StubLLMProvider) ParseTask(ctx context.Context, req ParseRequest) (*LLMResponse, error) {
	s.logger.Info("Stub LLM provider parsing task", 
		zap.String("text", req.Text),
		zap.String("user_id", string(req.UserID)))

	// Deterministic parsing for the test message "call mom tomorrow at 5pm"
	var parsedTask ParsedTask
	var confidence float64

	switch req.Text {
	case "call mom tomorrow at 5pm":
		// Parse the time to tomorrow at 5pm
		now := time.Now()
		tomorrow := now.AddDate(0, 0, 1)
		dueDate := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 17, 0, 0, 0, tomorrow.Location())
		
		parsedTask = ParsedTask{
			Title:       "Call mom",
			Description: "Call mom tomorrow at 5pm",
			DueDate:     &dueDate,
			Priority:    common.PriorityMedium,
			Tags:        []string{"personal", "call"},
		}
		confidence = ConfidenceHigh
	default:
		// Generic parsing for other messages
		parsedTask = ParsedTask{
			Title:       "Generic Task",
			Description: req.Text,
			DueDate:     nil,
			Priority:    common.PriorityMedium,
			Tags:        []string{"general"},
		}
		confidence = ConfidenceMedium
	}

	response := &LLMResponse{
		ParsedTask: parsedTask,
		Confidence: confidence,
		Reasoning:  "Stub LLM provider response for testing",
	}

	s.logger.Info("Stub LLM provider parsed task successfully",
		zap.String("title", parsedTask.Title),
		zap.String("priority", string(parsedTask.Priority)),
		zap.Float64("confidence", confidence))

	return response, nil
}

// ValidateConnection implements LLMProvider interface (always succeeds for stub)
func (s *StubLLMProvider) ValidateConnection(ctx context.Context) error {
	s.logger.Info("Stub LLM provider connection validation (always succeeds)")
	return nil
}

// GetModelInfo implements LLMProvider interface with mock model information
func (s *StubLLMProvider) GetModelInfo() ModelInfo {
	return ModelInfo{
		Name:         "stub-llm-model",
		Version:      "1.0.0-test",
		Provider:     "Stub Provider",
		Capabilities: []string{"task_parsing", "text_analysis"},
		MaxTokens:    4096,
	}
}
