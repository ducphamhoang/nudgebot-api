I have created the following plan after thorough exploration and analysis of the codebase. Follow the below plan verbatim. Trust the files and references. Do not re-verify what's written in the plan. Explore only when absolutely necessary. First implement all the proposed file changes and then I'll review all the changes together at the end.

### Observations

The codebase has a solid foundation for mock generation with existing `go:generate` directives, but the infrastructure isn't complete. The current `/internal/mocks/` location creates Go import restrictions that prevent cross-package mock usage. All existing test mocks are simple manual implementations that can be easily replaced with generated mocks. The event-driven architecture requires special handling to test handler registration and triggering. The `nudge/service_test.go` file uses sophisticated local mocks with advanced features that should be preserved. Tool dependency management is missing, requiring `tools.go` and `go.mod` updates.

### Approach

I'll implement a comprehensive mock generation system using `mockgen` and refactor the existing unit tests to use generated mocks instead of manual implementations. The key challenge is the current `/internal/mocks/` location which creates import restrictions for `/api/` packages. I'll solve this by moving all generated mocks to a root-level `/mocks` directory and establishing proper tool dependency management.

The approach involves: (1) Setting up proper tool dependencies with `tools.go` and `go.mod` updates, (2) Creating a new root-level `/mocks` directory with consolidated `go:generate` directives, (3) Generating mocks for all required interfaces (EventBus, TelegramProvider, NudgeRepository, and service interfaces), (4) Refactoring `webhook_test.go` and `llm/service_test.go` to use generated mocks with gomock patterns, and (5) Preserving the rich local mocks in `nudge/service_test.go` that provide advanced testing capabilities beyond simple interface mocking.

### Reasoning

I analyzed the current mock infrastructure and found that while `go:generate` directives exist in `/internal/mocks/interfaces.go`, none of the generated mock files actually exist. The codebase uses hand-written mocks instead. I examined the test files and discovered they use simple manual mocks with no advanced features, making migration to generated mocks straightforward. I identified import restriction issues with the current `/internal/mocks/` location that prevents `/api/` packages from importing mocks. I also found that the `mockgen` tool isn't properly managed as a dependency. I traced through the service initialization patterns to understand how event handlers are registered and tested, which is crucial for testing event-driven services with generated mocks.

## Mermaid Diagram

sequenceDiagram
    participant Dev as Developer
    participant Tools as Tool Management
    participant MockGen as Mock Generation
    participant Tests as Unit Tests
    participant CI as Build System
    
    Note over Dev,CI: Setup Phase
    Dev->>Tools: 1. Create tools.go with mockgen import
    Dev->>Tools: 2. Add golang/mock to go.mod
    Dev->>MockGen: 3. Create /mocks/generate.go with directives
    Dev->>CI: 4. Update Makefile targets
    
    Note over Dev,CI: Mock Generation Phase
    CI->>Tools: 5. Run install-tools (go install mockgen)
    CI->>MockGen: 6. Run generate-mocks (go generate ./mocks/...)
    MockGen->>MockGen: 7. Generate mock files for all interfaces
    
    Note over Dev,CI: Test Migration Phase
    Dev->>Tests: 8. Replace manual mocks in webhook_test.go
    Tests->>MockGen: Use gomock.Controller + EXPECT() patterns
    Dev->>Tests: 9. Replace manual mocks in llm/service_test.go
    Tests->>MockGen: Use DoAndReturn to capture event handlers
    Dev->>Tests: 10. Preserve rich mocks in nudge/service_test.go
    
    Note over Dev,CI: Cleanup Phase
    Dev->>MockGen: 11. Delete /internal/mocks directory
    Dev->>CI: 12. Verify all tests pass
    
    Note over Dev,CI: Result: Generated Mocks + Comprehensive Tests
    Tests->>Tests: ✅ All service interfaces mocked with gomock
    Tests->>Tests: ✅ Event handlers properly tested
    Tests->>Tests: ✅ No import cycle issues
    Tests->>Tests: ✅ Existing rich mocks preserved

## Proposed File Changes

### tools.go(NEW)

Create a new `tools.go` file with build tag `//go:build tools` to track the mockgen tool dependency. Import `github.com/golang/mock/mockgen` with a blank import (`_ "github.com/golang/mock/mockgen"`) to ensure the tool is available when running `go generate` commands. Add a comment explaining that this file tracks build-time tool dependencies and should not be included in the main build. This follows Go best practices for managing code generation tools.

### go.mod(MODIFY)

Add the required mockgen dependencies to the `require` section. Add `github.com/golang/mock/gomock v1.6.0` for the runtime mock support used in tests and `github.com/golang/mock/mockgen v1.6.0` for the code generation tool. These dependencies are necessary for the gomock framework and the mockgen tool to function properly. After adding these dependencies, run `go mod tidy` to resolve any transitive dependencies and update the `go.sum` file.

### mocks(NEW)

Create a new root-level `mocks` directory to house all generated mocks. This location avoids Go's internal import restrictions and allows all packages (including `/api/`, `/internal/`, and integration tests) to import mocks freely. This solves the current problem where `/api/handlers/webhook_test.go` cannot import from `/internal/mocks/` due to Go's internal package rules.

### mocks/generate.go(NEW)

References: 

- internal/mocks/interfaces.go

Create the main mock generation file with build tag `//go:build generate` and package declaration `package mocks`. Add comprehensive `go:generate` directives for all required interfaces:

- `//go:generate mockgen -source=../internal/events/bus.go -destination=mock_eventbus.go -package=mocks EventBus`
- `//go:generate mockgen -source=../internal/chatbot/provider.go -destination=mock_telegram_provider.go -package=mocks TelegramProvider`
- `//go:generate mockgen -source=../internal/nudge/repository.go -destination=mock_nudge_repository.go -package=mocks NudgeRepository`
- `//go:generate mockgen -source=../internal/chatbot/service.go -destination=mock_chatbot_service.go -package=mocks ChatbotService`
- `//go:generate mockgen -source=../internal/llm/service.go -destination=mock_llm_service.go -package=mocks LLMService`
- `//go:generate mockgen -source=../internal/nudge/service.go -destination=mock_nudge_service.go -package=mocks NudgeService`

Add comments explaining that this file centralizes all mock generation directives and that running `go generate ./mocks/...` will create all required mocks.

### Makefile(MODIFY)

Update the mock generation workflow in the Makefile. Add a new `install-tools` target that runs `go mod download` and `go install github.com/golang/mock/mockgen` to ensure the mockgen tool is available. Modify the existing `generate-mocks` target to depend on `install-tools` and change the command from `go generate ./internal/mocks/...` to `go generate ./mocks/...` to point to the new mock location. Update any other targets that reference the old mock location. Add comments explaining the tool installation and mock generation process.

### api/handlers/webhook_test.go(MODIFY)

Replace the manual `mockChatbotService` struct with generated mocks using the gomock framework. Add imports for `nudgebot-api/mocks` and `github.com/golang/mock/gomock`. In each test function, create a `gomock.Controller` with `gomock.NewController(t)` and defer `ctrl.Finish()`. Replace the manual mock creation with `mocks.NewMockChatbotService(ctrl)`. Convert the existing mock behavior configuration (like `shouldFail` and `handleWebhookError` fields) to gomock expectations using `EXPECT().Method().Return()` patterns.

For example, replace `mockService := &mockChatbotService{shouldFail: false}` with:
```
ctrl := gomock.NewController(t)
defer ctrl.Finish()
mockService := mocks.NewMockChatbotService(ctrl)
mockService.EXPECT().HandleWebhook(gomock.Any()).Return(nil)
```

Preserve all existing test logic, assertions, and table-driven test structures. Only change the mock setup and configuration parts.

### internal/llm/service_test.go(MODIFY)

Replace the manual `mockLLMService` and `events.NewMockEventBus()` with generated mocks using gomock. Add imports for `nudgebot-api/mocks` and `github.com/golang/mock/gomock`. The key challenge is testing event handler registration and triggering.

For event handler testing, use gomock's `DoAndReturn` feature to capture handler functions during `Subscribe` calls:
```
var messageHandler func(events.MessageReceived)
mockBus.EXPECT().Subscribe(events.TopicMessageReceived, gomock.Any()).DoAndReturn(
    func(topic string, handler interface{}) error {
        messageHandler = handler.(func(events.MessageReceived))
        return nil
    }
)
```

After service initialization, trigger the captured handler directly: `messageHandler(testEvent)` and verify the expected `Publish` calls using `EXPECT().Publish(events.TopicTaskParsed, gomock.Any()).Times(1)`.

Replace the manual `mockLLMService` with `mocks.NewMockLLMService(ctrl)` and configure expectations for `ParseTask`, `ValidateTask`, and `GetSuggestions` methods. Preserve all existing test cases and logic, only changing the mock setup patterns.

### internal/nudge/service_test.go(MODIFY)

Preserve all existing local mock implementations (`mockEventBus`, `MockEventBus`, `NewMockEventBus`) as they provide rich testing capabilities beyond simple interface mocking, including event tracking, subscriber counts, synchronous/asynchronous modes, and event waiting functionality that generated mocks cannot easily replicate.

The only changes needed are:
1. Add import alias for the new generated mocks if there are naming conflicts: `import generatedmocks "nudgebot-api/mocks"`
2. Ensure the existing `NewMockEventBus()` function doesn't conflict with any generated mock constructors by using explicit package qualification if needed
3. Verify that all existing tests continue to pass without modification

Do not replace any of the sophisticated local mocks as they are specifically designed for testing the complex event-driven behavior of the nudge service and provide capabilities that simple generated mocks cannot match.

### internal/mocks(DELETE)

References: 

- internal/mocks/interfaces.go
- internal/mocks/http_client_mock.go

Remove the entire `/internal/mocks` directory and all its contents after successfully migrating to the new root-level `/mocks` directory. This includes deleting `interfaces.go`, `http_client_mock.go`, `llm_mocks.go`, `scheduler_mocks.go`, `chatbot_mocks.go`, and any other mock files. This cleanup eliminates the import restriction issues and ensures all mocks are centralized in the new location. Before deletion, verify that all tests are passing with the new generated mocks and that no code still references the old mock location.