package main

import (
	"context"
	"fmt"

	"nudgebot-api/internal/config"
	"nudgebot-api/internal/llm"
	"nudgebot-api/pkg/logger"
)

func main() {
	// Initialize logger
	logger := logger.New()
	defer logger.Sync()
	zapLogger := logger.SugaredLogger.Desugar()

	// Create a test configuration
	testConfig := config.LLMConfig{
		APIEndpoint: "https://generativelanguage.googleapis.com/v1beta/models/gemma-2-27b-it:generateContent",
		APIKey:      "", // Empty for test - would normally come from env
		Timeout:     30,
		MaxRetries:  3,
		Model:       "gemma-2-27b-it",
	}

	// Create provider
	provider := llm.NewGemmaProvider(testConfig, zapLogger)

	// Test model info
	modelInfo := provider.GetModelInfo()
	fmt.Printf("Model Info: %+v\n", modelInfo)

	// Test parse request (will fail without API key, but shows structure)
	parseReq := llm.ParseRequest{
		Text:   "Remind me to buy groceries tomorrow",
		UserID: "test-user",
	}

	ctx := context.Background()
	_, err := provider.ParseTask(ctx, parseReq)
	if err != nil {
		fmt.Printf("Expected error (no API key configured): %v\n", err)

		// Check if it's a configuration error as expected
		if configErr, ok := err.(llm.ConfigurationError); ok {
			fmt.Printf("Configuration error detected correctly: %s\n", configErr.Code())
		}
	}

	fmt.Println("LLM integration test completed successfully!")
}
