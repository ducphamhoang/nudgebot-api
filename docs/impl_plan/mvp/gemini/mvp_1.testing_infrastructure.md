I have created the following plan after thorough exploration and analysis of the codebase. Follow the below plan verbatim. Trust the files and references. Do not re-verify what's written in the plan. Explore only when absolutely necessary. First implement all the proposed file changes and then I'll review all the changes together at the end.

### Observations

The current centralized mock structure in `/internal/mocks` creates import cycles because mocks import from service packages while tests import from mocks. The most widely used mocks are `MockEventBus` (used in chatbot, llm, scheduler, and integration tests) and `EnhancedMockNudgeRepository` (used in scheduler tests). There's already a local mock pattern established in `/internal/nudge/mock_repository.go`. The integration tests reference a non-existent `MockEventStore`. Multiple compilation errors exist including unused variables, incorrect function signatures, and missing interface methods. The `go.mod` file is missing required dependencies for mockgen and testcontainers.

### Approach

The root cause of the import cycle issue is the centralized `/internal/mocks` directory that imports from service packages to implement their interfaces, while test files import from this central location. This creates circular dependencies that prevent compilation.

The solution is to move service-specific mocks to their respective local packages, breaking the import cycles. The most critical mocks to move are `MockEventBus` (used across all test files) and `EnhancedMockNudgeRepository` (used in scheduler tests). Additionally, we need to address the missing `MockEventStore` referenced in integration tests and fix compilation errors.

This approach follows the existing pattern shown in `/internal/nudge/mock_repository.go` where a local mock already exists within the service package.

### Reasoning

I analyzed the codebase structure and identified that all test files import from `internal/mocks` which creates import cycles. I examined the mock implementations and found that `MockEventBus` and `EnhancedMockNudgeRepository` are the primary mocks used across test files. I also discovered that `MockEventStore` is referenced in integration tests but doesn't exist anywhere in the codebase. I traced the import cycle paths and found that the central mocks directory imports from service packages while tests import from mocks, creating the circular dependency. I also identified specific compilation errors in test files that need to be fixed.

## Mermaid Diagram

sequenceDiagram
    participant Dev as Developer
    participant Tests as Test Files
    participant LocalMocks as Local Mocks
    participant Interfaces as Service Interfaces
    
    Note over Dev,Interfaces: Current State: Import Cycles
    Tests->>LocalMocks: ❌ import internal/mocks
    LocalMocks->>Interfaces: ❌ import service packages
    Note over Tests,Interfaces: Creates circular dependency
    
    Note over Dev,Interfaces: After Fix: Local Mocks
    Dev->>LocalMocks: 1. Move MockEventBus to internal/events/
    Dev->>LocalMocks: 2. Move EnhancedMockNudgeRepository to internal/nudge/
    Dev->>Tests: 3. Update imports in test files
    Tests->>LocalMocks: ✅ import from same/local packages
    LocalMocks->>Interfaces: ✅ implement local interfaces
    Note over Tests,Interfaces: No circular dependencies
    
    Dev->>Tests: 4. Fix compilation errors
    Dev->>Dev: 5. Add missing dependencies to go.mod
    Dev->>Dev: 6. Remove central mocks directory
    
    Note over Dev,Interfaces: Result: Green Build
    Tests->>Tests: ✅ go test ./... passes
    Tests->>Tests: ✅ No import cycles
    Tests->>Tests: ✅ All compilation errors fixed

## Proposed File Changes

### internal/events/mock_bus.go(NEW)

References: 

- internal/mocks/event_mocks.go
- internal/events/bus.go

Create a new file containing the `MockEventBus` implementation moved from `/workspaces/nudgebot-api/internal/mocks/event_mocks.go`. Copy the entire `MockEventBus` struct, its constructor `NewMockEventBus()`, all methods (Subscribe, Unsubscribe, Publish, Close, GetPublishedEvents, GetSubscriberCount, ClearEvents, WaitForEvent, SimulateEventDelivery, invokeHandler), the `MockEventHandler` struct and its methods, all factory methods for creating test events (CreateMessageReceivedEvent, CreateTaskParsedEvent, etc.), assertion helpers (AssertEventPublished, AssertEventCount, etc.), error types (TimeoutError, HandlerError), and helper functions (eventsEqual). Update the package declaration to `package events` and adjust imports to remove the dependency on `internal/events` since this file will be in the same package. This provides a local mock for the EventBus interface that can be used by test files without creating import cycles.

### internal/nudge/enhanced_mock_repository.go(NEW)

References: 

- internal/mocks/nudge_mocks.go
- internal/nudge/repository.go
- internal/nudge/mock_repository.go

Create a new file containing the `EnhancedMockNudgeRepository` implementation moved from `/workspaces/nudgebot-api/internal/mocks/nudge_mocks.go`. Copy the entire `EnhancedMockNudgeRepository` struct, its constructor `NewEnhancedMockNudgeRepository()`, all repository methods (CreateTask, GetTaskByID, GetTasksByUserID, UpdateTask, DeleteTask, GetTaskStats, CreateReminder, GetDueReminders, MarkReminderSent, GetRemindersByTaskID, DeleteReminder, GetNudgeSettingsByUserID, CreateOrUpdateNudgeSettings, DeleteNudgeSettings, WithTransaction), helper methods (SetupTestData, ClearData, SetError, GetCallCount, incrementCallCount, checkError), factory methods (CreateTestTask, CreateOverdueTask, CreateCompletedTask, createTestReminder), and the additional mock business logic components (MockTaskValidator, MockReminderManager, MockStatusManager) with all their methods. Update the package declaration to `package nudge` and adjust imports to remove the dependency on `internal/nudge` since this file will be in the same package. This provides an enhanced local mock for the NudgeRepository interface that can be used by test files without creating import cycles.

### internal/chatbot/service_test.go(MODIFY)

References: 

- internal/events/mock_bus.go(NEW)

Update the import statement to remove `nudgebot-api/internal/mocks` and add `nudgebot-api/internal/events` if not already present. Replace all occurrences of `mocks.NewMockEventBus()` with `events.NewMockEventBus()` throughout the file. Remove or properly use the unused `mockChatbotService` variables at lines 55, 144, 215, 238, 291, and 324 - either delete the variable declarations if they're not needed or actually use them in the test logic. Update the `createMockChatbotService` helper function to use the new local mock location. Ensure all test functions that create mock event buses use the new `events.NewMockEventBus()` constructor. This fixes the import cycle issue and compilation errors related to unused variables.

### internal/llm/service_test.go(MODIFY)

References: 

- internal/events/mock_bus.go(NEW)

Update the import statement to remove `nudgebot-api/internal/mocks` and add `nudgebot-api/internal/events` if not already present. Replace all occurrences of `mocks.NewMockEventBus()` with `events.NewMockEventBus()` throughout the file. Remove or properly use the unused `mockLLMService` variables at multiple lines and the unused `capturedTaskParsed` variable at line 415. Fix the assignment mismatch at line 453 by correcting the number of variables to match what `NewLLMService` returns (change from 2 variables to 1). Fix the function call at line 453 by providing the correct number of arguments to `NewLLMService` (add the missing third argument). Fix the interface implementation error by adding the missing `GetSuggestions` method to the `mockLLMService` struct or use a different approach that doesn't require this method. Update the `createMockLLMService` helper function to use the new local mock location.

### internal/scheduler/scheduler_test.go(MODIFY)

References: 

- internal/events/mock_bus.go(NEW)
- internal/nudge/enhanced_mock_repository.go(NEW)

Update the import statement to remove `nudgebot-api/internal/mocks` and add `nudgebot-api/internal/events` if not already present. Replace all occurrences of `mocks.NewMockEventBus()` with `events.NewMockEventBus()` throughout the file. Replace all occurrences of `mocks.NewEnhancedMockNudgeRepository()` with `nudge.NewEnhancedMockNudgeRepository()` throughout the file. Remove or properly use the unused `scheduler` variable at line 303. Update all test functions that use these mocks to call the new constructors from their local packages. Update function signatures in test helper functions that reference the mock types to use the new package-qualified names (e.g., `*nudge.EnhancedMockNudgeRepository` instead of `*mocks.EnhancedMockNudgeRepository`). This fixes both the import cycle issue and ensures the scheduler tests use the locally available mocks.

### integration_test.go(MODIFY)

References: 

- internal/events/mock_bus.go(NEW)

Update the import statement to remove `nudgebot-api/internal/mocks` and add `nudgebot-api/internal/events` if not already present. Replace all occurrences of `mocks.NewMockEventBus()` with `events.NewMockEventBus()` throughout the file. Address the missing `MockEventStore` issue by either: (1) removing all references to `mocks.NewMockEventStore()` and `MockEventStore` if event store functionality is not needed for the tests, or (2) creating a simple mock event store interface and implementation in the `internal/events` package if the functionality is required. Update the helper functions `setupMockNudgeService`, `setupMockLLMService`, and `setupMockChatbotService` to use the new local mock locations. If keeping event store functionality, create a minimal `EventStore` interface in `/workspaces/nudgebot-api/internal/events/store.go` and a corresponding `MockEventStore` in the mock_bus.go file.

### go.mod(MODIFY)

Add the missing dependencies to the `require` section of the go.mod file. Add `github.com/golang/mock/mockgen v1.6.0` (or the latest stable version) for mock generation capabilities. Add `github.com/testcontainers/testcontainers-go v0.22.0` (or the latest stable version) for integration testing with containers. These dependencies are required for the testing infrastructure as specified in the task requirements. After adding these dependencies, run `go mod tidy` to ensure all transitive dependencies are properly resolved and the go.sum file is updated.

### internal/mocks(DELETE)

Delete the entire `/workspaces/nudgebot-api/internal/mocks` directory and all its contents since the mocks have been moved to their respective local packages. This includes removing `event_mocks.go`, `nudge_mocks.go`, `interfaces.go`, `mocks_test.go`, and all other mock files. This eliminates the central mock location that was causing import cycles. If any other mock files in this directory are still needed by other parts of the codebase not covered in this task, they should be moved to appropriate local packages following the same pattern established for EventBus and NudgeRepository mocks.

### api/handlers/handlers_test.go(MODIFY)

Create a placeholder test file for the handlers package if it doesn't exist, or update it if it does exist to remove any references to `internal/mocks` and use local mocks instead. This file was mentioned in the task requirements but was not found during analysis. If the file exists and has import cycle issues, update the imports to use local mocks following the same pattern as the other test files. If the file doesn't exist, create a basic test structure that can be expanded later without import cycle issues.

### internal/common/event_utils_test.go(MODIFY)

Create a placeholder test file for the common package's event utilities if it doesn't exist, or update it if it does exist to remove any references to `internal/mocks` and use local mocks instead. This file was mentioned in the task requirements but was not found during analysis. If the file exists and has import cycle issues, update the imports to use local mocks following the same pattern as the other test files. If the file doesn't exist, create a basic test structure for event utilities that can be expanded later without import cycle issues.