I have created the following plan after thorough exploration and analysis of the codebase. Follow the below plan verbatim. Trust the files and references. Do not re-verify what's written in the plan. Explore only when absolutely necessary. First implement all the proposed file changes and then I'll review all the changes together at the end.

### Observations

The current codebase is a minimal Go HTTP server with Gin, GORM/PostgreSQL, and Zap logging. The project follows standard Go layout patterns and uses Viper for configuration. The existing code is clean and follows good practices with proper error handling, graceful shutdown, and structured logging. The user wants to establish a modular architecture foundation with three core modules and an event bus system for inter-module communication.

### Approach

I'll create a modular architecture foundation by establishing three main modules (`chatbot/`, `nudge/`, `llm/`) in the `internal/` directory, each with clear interfaces and domain models. An event bus system will be implemented in `internal/events/` to enable decoupled communication between modules. The approach focuses on creating the structural foundation without implementing business logic, ensuring clean separation of concerns and preparing for the subsequent implementation phases. Dependencies will be added to support event-driven architecture and common utilities. I'll include repository interface definitions and a basic testing strategy with mock interfaces to ensure comprehensive foundation setup.

### Reasoning

I explored the current codebase structure and examined key files including the main server entry point, configuration management, database setup, logging implementation, and existing API routes. I analyzed the current dependencies in go.mod and understood the existing patterns for error handling, configuration, and project organization. This analysis revealed a clean, minimal foundation that's ready for modular architecture implementation.

## Mermaid Diagram

sequenceDiagram
    participant Main as main.go
    participant EB as EventBus
    participant CS as ChatbotService
    participant LS as LLMService
    participant NS as NudgeService
    participant NR as NudgeRepository
    participant Mocks as Test Mocks
    
    Main->>EB: Create EventBus
    Main->>CS: NewChatbotService(eventBus, logger)
    Main->>LS: NewLLMService(eventBus, logger)
    Main->>NS: NewNudgeService(eventBus, logger, repository)
    
    Note over CS,NS: Event Subscriptions Setup
    CS->>EB: Subscribe to TaskParsed, ReminderDue
    LS->>EB: Subscribe to MessageReceived
    NS->>EB: Subscribe to TaskParsed
    
    Note over Main: Server Running - Event Flow Example
    CS->>EB: Publish MessageReceived
    EB->>LS: Handle MessageReceived
    LS->>EB: Publish TaskParsed
    EB->>NS: Handle TaskParsed
    NS->>NR: Create Task (via repository interface)
    EB->>CS: Handle TaskParsed
    
    Note over Mocks: Testing Layer
    Mocks->>CS: Mock EventBus for unit tests
    Mocks->>NS: Mock Repository for service tests
    Mocks->>EB: Mock Event handlers
    
    Note over Main: Graceful Shutdown
    Main->>EB: Close()
    EB->>CS: Unsubscribe handlers
    EB->>LS: Unsubscribe handlers
    EB->>NS: Unsubscribe handlers

## Proposed File Changes

### go.mod(MODIFY)

Update the go.mod file to include new dependencies required for the modular architecture:

- Add `github.com/google/uuid v1.4.0` for generating unique identifiers across modules
- Add `github.com/asaskevich/EventBus v0.0.0-20200907212545-49d423059eef` for lightweight in-memory event bus implementation
- Add `golang.org/x/sync v0.5.0` for advanced synchronization primitives needed for event handling
- Add `github.com/stretchr/testify v1.8.4` for comprehensive testing framework with mocks and assertions
- Add `github.com/golang/mock v1.6.0` for generating mocks from interfaces

These dependencies will support the event-driven architecture, provide utilities for the modular system, and enable comprehensive testing without adding unnecessary complexity.

### internal/events(NEW)

Create the events directory to house the event bus system and event definitions for inter-module communication.

### internal/events/bus.go(NEW)

Create the event bus interface and implementation using the EventBus library. Define the `EventBus` interface with methods for `Publish(topic string, data interface{})`, `Subscribe(topic string, handler interface{})`, `Unsubscribe(topic string, handler interface{})`, and `Close()`. Implement a wrapper around the EventBus library that provides type safety and proper error handling. Include context support for graceful shutdown and correlation ID tracking. The implementation should be thread-safe and support asynchronous event processing.

### internal/events/types.go(NEW)

Define the core event types that will be used for inter-module communication. Create structs for:

- `MessageReceived` event with fields: CorrelationID, UserID, ChatID, MessageText, Timestamp
- `TaskParsed` event with fields: CorrelationID, UserID, ParsedTask (embedded struct), Timestamp
- `ReminderDue` event with fields: CorrelationID, TaskID, UserID, ChatID, Timestamp
- `TaskCompleted` event with fields: CorrelationID, TaskID, UserID, CompletedAt

Each event struct should include proper JSON tags for serialization and validation tags. Include a base `Event` struct with common fields like CorrelationID and Timestamp that other events can embed.

### internal/common(NEW)

Create the common directory for shared types and utilities used across modules.

### internal/common/types.go(NEW)

Define common types and utilities shared across modules:

- `ID` type as string wrapper for UUIDs with helper methods `NewID()` and `IsValid()`
- `UserID`, `ChatID`, `TaskID` as typed aliases of ID for type safety
- Common error types: `ValidationError`, `NotFoundError`, `InternalError`
- `TaskStatus` enum with values: Active, Completed, Snoozed, Deleted
- `Priority` enum with values: Low, Medium, High, Urgent
- Helper functions for UUID generation and validation

Include proper string methods and JSON marshaling for all types.

### internal/chatbot(NEW)

Create the chatbot module directory that will handle all Telegram Bot API interactions and user interface logic.

### internal/chatbot/domain.go(NEW)

References: 

- internal/common/types.go(NEW)

Define domain models for the chatbot module:

- `Message` struct with fields: ID, UserID, ChatID, Text, Timestamp, MessageType (command, text, callback)
- `InlineKeyboard` struct for Telegram inline keyboards with buttons
- `ChatSession` struct to track user conversation state
- `Command` enum for supported commands: Start, Help, List, Done, Delete
- `CallbackData` struct for handling inline keyboard callbacks

Include validation tags and JSON serialization tags. Reference the common types from `internal/common/types.go` for UserID and ChatID.

### internal/chatbot/service.go(NEW)

References: 

- internal/events/types.go(NEW)
- internal/events/bus.go(NEW)
- pkg/logger/logger.go

Define the ChatbotService interface and basic implementation structure:

- `ChatbotService` interface with methods: `SendMessage(chatID, text)`, `SendMessageWithKeyboard(chatID, text, keyboard)`, `HandleWebhook(webhookData)`, `ProcessCommand(command, userID, chatID)`
- `chatbotService` struct implementing the interface with fields for eventBus, logger, and config
- Constructor function `NewChatbotService(eventBus, logger)` that returns the interface
- Method stubs that log the operation and publish appropriate events to the event bus
- Event subscription setup in the constructor for `TaskParsed` and `ReminderDue` events

The implementation should be event-driven, publishing `MessageReceived` events and subscribing to relevant events from other modules. Reference the event types from `internal/events/types.go`.

### internal/llm(NEW)

Create the LLM module directory that will handle natural language processing and task parsing using external LLM APIs.

### internal/llm/domain.go(NEW)

References: 

- internal/common/types.go(NEW)

Define domain models for the LLM module:

- `ParseRequest` struct with fields: Text, UserID, Context (for conversation context)
- `ParsedTask` struct with fields: Title, Description, DueDate (optional), Priority, Tags (slice of strings)
- `LLMResponse` struct with fields: ParsedTask, Confidence (float), Reasoning (string)
- `ParseError` struct for handling parsing failures with error codes and messages
- `ContextData` struct for maintaining conversation context

Include validation tags for required fields and JSON serialization. Reference common types from `internal/common/types.go` for UserID and Priority.

### internal/llm/service.go(NEW)

References: 

- internal/events/types.go(NEW)
- internal/events/bus.go(NEW)
- pkg/logger/logger.go

Define the LLMService interface and basic implementation structure:

- `LLMService` interface with methods: `ParseTask(text, userID)`, `ValidateTask(parsedTask)`, `GetSuggestions(partialText, userID)`
- `llmService` struct implementing the interface with fields for eventBus, logger, httpClient, and config
- Constructor function `NewLLMService(eventBus, logger)` that returns the interface
- Method stubs that log the operation and simulate task parsing
- Event subscription setup for `MessageReceived` events from the chatbot module
- Event publishing for `TaskParsed` events when parsing is complete

The service should subscribe to `MessageReceived` events, process the natural language text, and publish `TaskParsed` events. Include error handling for parsing failures. Reference event types from `internal/events/types.go`.

### internal/nudge(NEW)

Create the nudge module directory that will handle task management, storage, and reminder logic.

### internal/nudge/domain.go(NEW)

References: 

- internal/common/types.go(NEW)

Define domain models for the nudge module:

- `Task` struct with fields: ID, UserID, Title, Description, DueDate, Priority, Status, CreatedAt, UpdatedAt, CompletedAt
- `Reminder` struct with fields: ID, TaskID, UserID, ScheduledAt, SentAt, ReminderType (initial, nudge)
- `TaskFilter` struct for querying tasks with fields: UserID, Status, Priority, DueBefore, DueAfter
- `TaskStats` struct with fields: TotalTasks, CompletedTasks, OverdueTasks, ActiveTasks
- `NudgeSettings` struct with fields: UserID, NudgeInterval, MaxNudges, Enabled

Include GORM tags for database mapping, validation tags, and JSON serialization. Reference common types from `internal/common/types.go` for ID types, TaskStatus, and Priority.

### internal/nudge/repository.go(NEW)

References: 

- internal/nudge/domain.go(NEW)
- internal/common/types.go(NEW)

Define the repository interface for the nudge module to establish clear data access contracts:

- `TaskRepository` interface with methods: `Create(task)`, `GetByID(taskID)`, `GetByUserID(userID, filter)`, `Update(task)`, `Delete(taskID)`, `GetStats(userID)`
- `ReminderRepository` interface with methods: `Create(reminder)`, `GetDueReminders(before time.Time)`, `MarkSent(reminderID)`, `GetByTaskID(taskID)`
- `NudgeRepository` interface that embeds both TaskRepository and ReminderRepository for unified data access

Include error definitions for common repository operations like `ErrTaskNotFound`, `ErrDuplicateTask`. The interfaces should use the domain models from `domain.go` and common types from `internal/common/types.go`. This establishes the contract that will be implemented with GORM in later phases.

### internal/nudge/service.go(NEW)

References: 

- internal/nudge/repository.go(NEW)
- internal/events/types.go(NEW)
- internal/events/bus.go(NEW)
- pkg/logger/logger.go

Define the NudgeService interface and basic implementation structure:

- `NudgeService` interface with methods: `CreateTask(task)`, `GetTasks(userID, filter)`, `UpdateTaskStatus(taskID, status)`, `DeleteTask(taskID)`, `GetTaskStats(userID)`
- `nudgeService` struct implementing the interface with fields for eventBus, logger, and repository (NudgeRepository interface)
- Constructor function `NewNudgeService(eventBus, logger, repository)` that returns the interface
- Method stubs that log operations and simulate task management using the repository interface
- Event subscription setup for `TaskParsed` events from the LLM module
- Event publishing for `ReminderDue` and `TaskCompleted` events

The service should subscribe to `TaskParsed` events to create tasks and publish `ReminderDue` events for the scheduler. The repository dependency should be injected through the constructor, following dependency inversion principle. Reference the repository interface from `repository.go` and event types from `internal/events/types.go`.

### internal/config/config.go(MODIFY)

Extend the existing configuration structure to support the new modular architecture:

- Add `Chatbot` configuration section with fields for webhook URL, token placeholder, and timeout settings
- Add `LLM` configuration section with fields for API endpoint, timeout, retry settings, and model parameters
- Add `Events` configuration section with fields for buffer size, worker count, and shutdown timeout
- Add `Nudge` configuration section with fields for default reminder interval, max nudges, and cleanup settings

Update the `setDefaults()` function to include sensible defaults for all new configuration sections. Maintain backward compatibility with existing server and database configurations. The configuration should follow the same pattern as the existing `ServerConfig` and `DatabaseConfig` structs.

### cmd/server/main.go(MODIFY)

References: 

- internal/events/bus.go(NEW)
- internal/chatbot/service.go(NEW)
- internal/llm/service.go(NEW)
- internal/nudge/service.go(NEW)
- internal/config/config.go(MODIFY)

Update the main server entry point to initialize the modular architecture:

- Import the new modules: `internal/events`, `internal/chatbot`, `internal/llm`, `internal/nudge`
- After loading configuration and before setting up Gin router, create the event bus instance using `events.NewEventBus()`
- Initialize the three service modules by calling their constructors with the event bus and logger (repository will be nil for nudge service in this phase)
- Add graceful shutdown handling for the event bus by calling `eventBus.Close()` in the shutdown sequence
- Store service instances in variables for potential future use (webhook endpoints, health checks)

The initialization should happen after database connection but before HTTP server setup. Maintain the existing error handling patterns and logging. Reference the new service constructors from each module's `service.go` file.

### internal/mocks(NEW)

Create the mocks directory to house generated and manual mocks for testing purposes.

### internal/mocks/interfaces.go(NEW)

References: 

- internal/events/bus.go(NEW)
- internal/chatbot/service.go(NEW)
- internal/llm/service.go(NEW)
- internal/nudge/service.go(NEW)
- internal/nudge/repository.go(NEW)

Define mock generation directives and manual mock interfaces for testing:

- Add `//go:generate` directives for mockgen to generate mocks for EventBus, ChatbotService, LLMService, NudgeService, and repository interfaces
- Create `MockEventBus` struct with methods that implement the EventBus interface for unit testing
- Create `MockNudgeRepository` struct implementing the NudgeRepository interface with in-memory storage for testing
- Include helper methods for setting up test expectations and verifying mock calls

The mocks should support both successful operations and error scenarios for comprehensive testing. Reference the interfaces from their respective service and repository files.

### Makefile(MODIFY)

Update the Makefile to include targets for the new modular architecture:

- Add `generate-mocks` target that runs `go generate ./internal/mocks/...` to generate interface mocks
- Add `test-unit` target that runs unit tests with coverage for individual modules
- Add `test-integration` target placeholder for future integration tests
- Add `lint-modules` target that runs linting specifically for the new modules
- Update existing targets to include the new module directories

Include proper dependency management between targets (e.g., generate-mocks should run before tests). The targets should follow the existing Makefile patterns and maintain compatibility with current build processes.