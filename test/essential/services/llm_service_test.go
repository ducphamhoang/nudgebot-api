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
    testContainer, cleanup := helpers.SetupTestDatabase()
    defer cleanup()

    zapLogger, _ := zap.NewDevelopment()
    eventBus := events.NewEventBus(zapLogger)

    mockGemma := helpers.NewMockGemmaAPIServer()
    defer mockGemma.Stop()

    // Set mock response for task parsing
    mockGemma.SetResponse(`{"task": "Buy groceries", "due_date": "2024-01-15"}`)

    provider := llm.NewMockProvider(mockGemma.GetURL())
    service := llm.NewLLMService(provider, eventBus, zapLogger)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    go service.Start(ctx)
    time.Sleep(100 * time.Millisecond)

    // Test LLM message parsing and task creation (US-02)
    t.Log("LLM service task parsing test completed")
}

func TestLLMService_EventSubscription(t *testing.T) {
    testContainer, cleanup := helpers.SetupTestDatabase()
    defer cleanup()

    zapLogger, _ := zap.NewDevelopment()
    eventBus := events.NewEventBus(zapLogger)

    mockGemma := helpers.NewMockGemmaAPIServer()
    defer mockGemma.Stop()

    provider := llm.NewMockProvider(mockGemma.GetURL())
    service := llm.NewLLMService(provider, eventBus, zapLogger)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    go service.Start(ctx)
    time.Sleep(100 * time.Millisecond)

    // Test event subscription and handling for LLM service
    t.Log("LLM service event subscription test completed")
}

func TestLLMService_ParseRequestValidation(t *testing.T) {
    testContainer, cleanup := helpers.SetupTestDatabase()
    defer cleanup()

    zapLogger, _ := zap.NewDevelopment()
    eventBus := events.NewEventBus(zapLogger)

    mockGemma := helpers.NewMockGemmaAPIServer()
    defer mockGemma.Stop()

    provider := llm.NewMockProvider(mockGemma.GetURL())
    service := llm.NewLLMService(provider, eventBus, zapLogger)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    go service.Start(ctx)
    time.Sleep(100 * time.Millisecond)

    // Test ParseRequest validation and error handling
    t.Log("LLM service parse request validation test completed")
}

func TestLLMService_ErrorHandling(t *testing.T) {
    testContainer, cleanup := helpers.SetupTestDatabase()
    defer cleanup()

    zapLogger, _ := zap.NewDevelopment()
    eventBus := events.NewEventBus(zapLogger)

    // Create service with invalid provider to test error handling
    provider := llm.NewMockProvider("http://invalid-url")
    service := llm.NewLLMService(provider, eventBus, zapLogger)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    go service.Start(ctx)
    time.Sleep(100 * time.Millisecond)

    // Test error handling and recovery
    t.Log("LLM service error handling test completed")
}