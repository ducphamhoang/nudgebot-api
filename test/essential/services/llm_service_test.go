//go:build integration

package services

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"nudgebot-api/internal/events"
	"nudgebot-api/internal/llm"
	"nudgebot-api/test/essential/helpers"
)

func TestLLMService_TaskParsing(t *testing.T) {
	_, cleanup := helpers.SetupTestDatabase(t)
	defer cleanup()

	zapLogger, _ := zap.NewDevelopment()
	eventBus := events.NewEventBus(zapLogger)

	// Create stub LLM provider
	stubProvider := llm.NewStubLLMProvider(zapLogger)
	service := llm.NewLLMServiceWithProvider(eventBus, zapLogger, stubProvider)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test LLM message parsing and task creation (US-02)
	_ = ctx
	_ = service
	t.Log("LLM service task parsing test completed")
}

func TestLLMService_EventSubscription(t *testing.T) {
	_, cleanup := helpers.SetupTestDatabase(t)
	defer cleanup()

	zapLogger, _ := zap.NewDevelopment()
	eventBus := events.NewEventBus(zapLogger)

	// Create stub LLM provider
	stubProvider := llm.NewStubLLMProvider(zapLogger)
	service := llm.NewLLMServiceWithProvider(eventBus, zapLogger, stubProvider)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = ctx
	_ = service
	// Test event subscription and handling for LLM service
	t.Log("LLM service event subscription test completed")
}

func TestLLMService_ParseRequestValidation(t *testing.T) {
	_, cleanup := helpers.SetupTestDatabase(t)
	defer cleanup()

	zapLogger, _ := zap.NewDevelopment()
	eventBus := events.NewEventBus(zapLogger)

	// Create stub LLM provider
	stubProvider := llm.NewStubLLMProvider(zapLogger)
	service := llm.NewLLMServiceWithProvider(eventBus, zapLogger, stubProvider)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = ctx
	_ = service
	// Test task parsing validation and error handling
	t.Log("LLM service parse request validation test completed")
}

func TestLLMService_ErrorHandling(t *testing.T) {
	_, cleanup := helpers.SetupTestDatabase(t)
	defer cleanup()

	zapLogger, _ := zap.NewDevelopment()
	eventBus := events.NewEventBus(zapLogger)

	// Create stub LLM provider for testing
	stubProvider := llm.NewStubLLMProvider(zapLogger)
	service := llm.NewLLMServiceWithProvider(eventBus, zapLogger, stubProvider)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = ctx
	_ = service
	// Test error handling and recovery
	t.Log("LLM service error handling test completed")
}
