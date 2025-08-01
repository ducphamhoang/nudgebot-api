package llm

import (
	"context"
)

// LLMProvider defines the interface for LLM implementations
type LLMProvider interface {
	// ParseTask parses natural language text into a structured task
	// ctx: context for timeout and cancellation control
	// req: the parse request containing text and user context
	// Returns the parsed task with confidence score or an error
	ParseTask(ctx context.Context, req ParseRequest) (*LLMResponse, error)

	// ValidateConnection checks if the LLM provider is reachable and healthy
	// ctx: context for timeout and cancellation control
	// Returns error if connection validation fails
	ValidateConnection(ctx context.Context) error

	// GetModelInfo returns metadata about the LLM model being used
	// Returns model information including name, version, and capabilities
	GetModelInfo() ModelInfo
}

// ModelInfo contains metadata about the LLM model
type ModelInfo struct {
	Name         string   `json:"name"`         // Model name (e.g., "gemma-2-27b-it")
	Version      string   `json:"version"`      // Model version
	Provider     string   `json:"provider"`     // Provider name (e.g., "Google")
	Capabilities []string `json:"capabilities"` // List of supported capabilities
	MaxTokens    int      `json:"max_tokens"`   // Maximum token limit
}
