I have created the following plan after thorough exploration and analysis of the codebase. Follow the below plan verbatim. Trust the files and references. Do not re-verify what's written in the plan. Explore only when absolutely necessary. First implement all the proposed file changes and then I'll review all the changes together at the end.

### Observations

The current LLM module has a basic structure with domain models, service interface, and event subscriptions already in place. The service currently uses mock implementations for task parsing. The configuration system supports LLM settings but lacks API key configuration. The main server initializes services but doesn't pass configuration to the LLM service. The codebase follows clean architecture patterns with proper separation of concerns and event-driven communication between modules.

### Approach

I'll implement the Gemma API integration by creating a provider abstraction layer that separates the LLM service logic from the specific API implementation. This approach allows for future extensibility to other LLM providers while maintaining clean interfaces. The implementation will include proper error handling, retry logic with exponential backoff, structured JSON output from Gemma, and comprehensive configuration management. I'll update the existing service to use the new provider while maintaining backward compatibility with the event-driven architecture.

### Reasoning

I explored the current codebase structure including the LLM module implementation, configuration system, event bus, common types, logger, and main server initialization. I also researched Gemma API integration patterns and response formats. The analysis revealed that the foundation is solid but needs actual API integration, proper configuration for API keys, and a provider abstraction layer for clean separation of concerns.

## Mermaid Diagram

sequenceDiagram
    participant EB as EventBus
    participant LS as LLMService
    participant GP as GemmaProvider
    participant GA as Gemma API
    participant Config as Configuration
    
    Note over LS: Service Initialization
    Config->>LS: Load LLMConfig with API key
    LS->>GP: NewGemmaProvider(config, logger)
    GP->>GP: Setup HTTP client & backoff strategy
    LS->>EB: Subscribe to MessageReceived events
    
    Note over EB,GA: Task Parsing Flow
    EB->>LS: MessageReceived event
    LS->>LS: Create ParseRequest with context
    LS->>GP: ParseTask(ctx, parseRequest)
    
    GP->>GP: Build structured JSON prompt
    GP->>GA: POST /generateContent with auth
    GA-->>GP: JSON response with task data
    GP->>GP: Parse & validate JSON response
    GP->>GP: Map to LLMResponse with confidence
    GP-->>LS: Return LLMResponse
    
    LS->>LS: Validate parsed task
    LS->>LS: Convert to events.TaskParsed
    LS->>EB: Publish TaskParsed event
    
    Note over GP,GA: Error Handling & Retry
    GP->>GA: API call fails
    GA-->>GP: Error response
    GP->>GP: Check if retryable
    GP->>GP: Apply exponential backoff
    GP->>GA: Retry API call
    
    Note over GP: Alternative: Max retries exceeded
    GP-->>LS: Return ParseError{SERVICE_UNAVAILABLE}
    LS->>LS: Log error & skip event publishing

## Proposed File Changes

### go.mod(MODIFY)

Add the `github.com/cenkalti/backoff/v4 v4.2.1` dependency for implementing exponential backoff retry logic in the Gemma API provider. This library provides robust retry mechanisms with configurable backoff strategies that are essential for handling API failures gracefully.

### internal/llm/provider.go(NEW)

References: 

- internal/llm/domain.go

Create the `LLMProvider` interface that defines the contract for LLM implementations. The interface should include:

- `ParseTask(ctx context.Context, req ParseRequest) (*LLMResponse, error)` method for parsing natural language into structured tasks
- `ValidateConnection(ctx context.Context) error` method for health checks
- `GetModelInfo() ModelInfo` method for retrieving model metadata

Define a `ModelInfo` struct with fields for model name, version, and capabilities. This abstraction allows the service to work with different LLM providers without coupling to specific implementations. Include comprehensive documentation for each interface method explaining expected behavior, error conditions, and context usage.

### internal/llm/gemma_provider.go(NEW)

References: 

- internal/llm/provider.go(NEW)
- internal/llm/domain.go
- internal/config/config.go(MODIFY)
- pkg/logger/logger.go

Implement the `GemmaProvider` struct that implements the `LLMProvider` interface for Google Gemma API integration. Include:

- `GemmaProvider` struct with fields for HTTP client, configuration, logger, and backoff strategy
- `NewGemmaProvider(config LLMConfig, logger *zap.Logger) *GemmaProvider` constructor
- `GemmaRequest` and `GemmaResponse` structs matching the Gemini API JSON format
- `ParseTask` method implementation that:
  - Constructs the API request with proper JSON structure and authentication
  - Includes a carefully crafted prompt that forces JSON output matching `ParsedTask` structure
  - Implements retry logic with exponential backoff using the backoff library
  - Parses the API response and extracts the generated JSON
  - Handles various error scenarios (network errors, API errors, invalid JSON)
  - Maps confidence scores and reasoning from the API response
- `buildPrompt` helper method that creates a structured prompt ensuring JSON output
- `parseGemmaResponse` helper method for extracting and validating the JSON response
- Comprehensive error handling with specific error codes for different failure modes

The implementation should use the Gemini API endpoint `https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent` with API key authentication. Reference the configuration from `internal/config/config.go` and domain models from `internal/llm/domain.go`.

### internal/config/config.go(MODIFY)

Extend the existing `LLMConfig` struct to include the `APIKey` field for Gemma API authentication:

- Add `APIKey string \`mapstructure:"api_key"\`` field to the `LLMConfig` struct
- Update the `setDefaults()` function to include `viper.SetDefault("llm.api_key", "")` with an empty default
- Ensure the new field follows the same naming convention as existing configuration fields

The configuration structure should maintain backward compatibility while adding the necessary authentication field for Gemma API integration. Reference the existing `LLMConfig` struct and `setDefaults()` function patterns.

### internal/llm/service.go(MODIFY)

References: 

- internal/llm/provider.go(NEW)
- internal/llm/gemma_provider.go(NEW)
- internal/config/config.go(MODIFY)

Update the `llmService` implementation to use the new provider abstraction:

- Add `provider LLMProvider` field to the `llmService` struct
- Remove the hardcoded `config *LLMConfig` field since configuration will be handled by the provider
- Update `NewLLMService(eventBus events.EventBus, logger *zap.Logger, config config.LLMConfig)` constructor signature to accept configuration
- In the constructor, create a `GemmaProvider` instance using `NewGemmaProvider(config, logger)` and assign it to the provider field
- Remove the hardcoded HTTP client and configuration initialization
- Update the `ParseTask` method to delegate to `s.provider.ParseTask(ctx, parseRequest)` instead of using mock logic
- Add proper context creation with timeout from configuration in the `handleMessageReceived` method
- Maintain all existing event subscription and publishing logic
- Update error handling to properly propagate provider errors
- Remove the mock task creation logic and replace with actual provider calls

The service should remain focused on orchestration and event handling while delegating the actual LLM operations to the provider. Reference the provider interface from `internal/llm/provider.go` and the configuration from `internal/config/config.go`.

### cmd/server/main.go(MODIFY)

References: 

- internal/llm/service.go(MODIFY)
- internal/config/config.go(MODIFY)

Update the main server initialization to pass configuration to the LLM service:

- Modify the `llmService := llm.NewLLMService(eventBus, zapLogger)` call to include the configuration: `llmService := llm.NewLLMService(eventBus, zapLogger, cfg.LLM)`
- Ensure the configuration is properly loaded and available before service initialization
- Maintain the existing service initialization order and error handling patterns
- Keep all other service initializations unchanged

The change should be minimal and focused only on passing the LLM configuration to the service constructor. Reference the configuration loading from `internal/config/config.go` and the updated service constructor from `internal/llm/service.go`.

### configs/config.yaml(MODIFY)

Add the LLM configuration section to the YAML configuration file:

```yaml
llm:
  api_endpoint: "https://generativelanguage.googleapis.com/v1beta/models/gemma-2-27b-it:generateContent"
  api_key: "" # Set via environment variable LLM_API_KEY
  timeout: 30
  max_retries: 3
  model: "gemma-2-27b-it"
```

Add this section after the existing database configuration. Include a comment indicating that the API key should be set via environment variable for security. Use the Gemma model endpoint as the default API endpoint.

### .env.example(MODIFY)

Add LLM configuration environment variables to the example file:

```
# LLM Configuration
LLM_API_ENDPOINT=https://generativelanguage.googleapis.com/v1beta/models/gemma-2-27b-it:generateContent
LLM_API_KEY=your_gemini_api_key_here
LLM_TIMEOUT=30
LLM_MAX_RETRIES=3
LLM_MODEL=gemma-2-27b-it
```

Add this section after the existing database configuration variables. Include a placeholder value for the API key that clearly indicates where users should input their actual key. Follow the same naming convention as existing environment variables.

### internal/mocks/llm_mocks.go(NEW)

References: 

- internal/llm/provider.go(NEW)
- internal/mocks/interfaces.go

Create mock implementations for testing the LLM module:

- Add `//go:generate mockgen -source=../llm/provider.go -destination=llm_mocks.go -package=mocks` directive for generating provider mocks
- Implement `MockLLMProvider` struct that implements the `LLMProvider` interface with configurable responses
- Include methods for setting up expected calls, return values, and error conditions
- Add helper methods like `SetParseTaskResponse`, `SetParseTaskError`, and `ExpectParseTaskCall` for test setup
- Implement `MockGemmaProvider` struct for integration testing scenarios
- Include factory methods for creating common test scenarios (success, API error, network error, invalid JSON)

The mocks should support both unit testing of the service layer and integration testing of the provider implementations. Reference the provider interface from `internal/llm/provider.go` and follow the existing mock patterns from `internal/mocks/interfaces.go`.

### internal/llm/errors.go(NEW)

References: 

- internal/llm/domain.go
- internal/common/types.go

Create comprehensive error handling for the LLM module:

- Define `LLMError` interface with methods `Code() string`, `Message() string`, and `Temporary() bool`
- Implement specific error types:
  - `APIError` for Gemma API failures with HTTP status codes
  - `NetworkError` for connection and timeout issues
  - `ParseError` for JSON parsing failures (extend existing ParseError)
  - `ConfigurationError` for invalid configuration
  - `RateLimitError` for API rate limiting
- Include error wrapping utilities that preserve context and correlation IDs
- Add error classification helpers like `IsRetryable(error) bool` and `IsTemporary(error) bool`
- Define error constants for common scenarios

The error system should provide clear categorization for different failure modes and support the retry logic in the provider implementation. Reference the existing error patterns from `internal/common/types.go` and the domain models from `internal/llm/domain.go`.