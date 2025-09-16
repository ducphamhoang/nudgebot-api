# AGENTS.md: A Guide for AI Assistants

Welcome, agent. This document is your primary guide to understanding and contributing to the **NudgeBot API** project. It provides essential context on product goals, architecture, development rules, and automated workflows.

## 1. Core Mission & Product Requirements

NudgeBot is a conversational task assistant for Telegram designed to provide proactive reminders and follow-ups to help users complete their tasks.

-   **Full Details**: `docs/mvp_prd.md`

### Key User Stories (MVP)
-   **US-01**: Onboard new users with a welcome message.
-   **US-02**: Add tasks using natural language (e.g., "call Sarah tomorrow at 3pm").
-   **US-04**: Receive a reminder when a task is due.
-   **US-05**: Mark tasks "Done" or "Snooze" via inline buttons.
-   **US-06**: Receive a follow-up "nudge" if a task is not completed.
-   **US-07**: View a list of all upcoming tasks.

## 2. Architecture & Codebase

The project uses **Go** and follows the **Clean Architecture** pattern, emphasizing a separation of concerns and an event-driven design.

-   **Full Details**: `docs/architecture.md`, `docs/codebase.md`

### Architectural Diagram
```
+-----------------+      +------------------+      +-----------------+
|   Telegram API  | <--> |  NudgeBot API    |      |   LLM API       |
+-----------------+      | (Gin)            | <--> +-----------------+
                         +------------------+
                                |
           +----------------------------------------+
           |   API Handlers (api/handlers)          |
           +----------------------------------------+
                                |
           +----------------------------------------+
           |   Chatbot Service (internal/chatbot)   |
           +----------------------------------------+
                                |
+-------------------------------------------------------------------+
|                          Event Bus (internal/events)              |
+-------------------------------------------------------------------+
     |                          |                          |
+------------------+ +--------------------+ +-----------------------+
| Nudge Service    | | Scheduler Service  | | ... other services    |
| (internal/nudge) | | (internal/scheduler)| |                       |
+------------------+ +--------------------+ +-----------------------+
     |
+----------------------------------------+
|   Nudge Repository (internal/nudge)    |
+----------------------------------------+
     |
+----------------------------------------+
|   PostgreSQL Database                  |
+----------------------------------------+
```

### Key Directories
-   `/api`: HTTP transport layer (handlers, middleware, routes).
-   `/cmd`: Main application entry point.
-   `/internal`: Core business logic, separated by domain (`chatbot`, `nudge`, `llm`, etc.).
-   `/pkg`: Shared libraries (e.g., logger).
-   `/test`: Integration and end-to-end tests.

## 3. Development Rules & Conventions

All contributions must adhere to the project's development standards to ensure code is idiomatic, modular, testable, and maintainable.

-   **Full Details**: `docs/rules.md`

### Core Principles
-   **Clean Architecture**: Structure code into handlers, services, repositories, and domain models.
-   **Interface-Driven**: All public functions should interact with interfaces, not concrete types. Use dependency injection.
-   **Error Handling**: Always check and handle errors explicitly using `fmt.Errorf("context: %w", err)`.
-   **Testing**: Write unit tests using table-driven patterns. Mock external interfaces.
-   **Observability**: Use OpenTelemetry for distributed tracing, metrics, and structured logging.

## 4. Testing Strategy & Status

The project uses a three-tier testing approach (Unit, Integration, E2E) to ensure full coverage of product requirements.

-   **Strategy**: `docs/test_strategy.md`
-   **Coverage Matrix**: `docs/test_coverage_matrix.md`
-   **Current Status**: `docs/test_status.md`

### Test Execution Commands
-   `make test-unit`: Runs fast unit tests with no external dependencies.
-   `make test-integration`: Runs integration tests using Docker testcontainers.
-   `make test-all`: Executes the complete test suite.

## 5. CI/CD Pipeline

The CI/CD pipeline automates code quality checks, testing, and builds.

-   **Full Details**: `.github/workflows/ci.yml`

### Main CI Jobs
-   `quality-checks`: Runs linters, formatters, and security audits.
-   `unit-tests`: Executes unit tests across multiple Go versions.
-   `integration-tests`: Runs integration tests with real database and Redis services.
-   `build-tests`: Compiles the application for different platforms.
-   `docker-tests`: Builds a Docker image and scans it for vulnerabilities.

## 6. Agent-Specific Commands & Workflows

These commands are designed to automate common development workflows.

-   **Full Details**: `.gemini/commands/`

### `specify`
-   **Description**: Create or update a feature specification from a natural language description.
-   **Workflow**:
    1.  Run `.specify/scripts/bash/create-new-feature.sh --json "{{args}}"` to create a branch and spec file.
    2.  Use `.specify/templates/spec-template.md` to structure the output.
    3.  Write the detailed specification to the newly created file.

### `plan`
-   **Description**: Execute the implementation planning workflow to generate design artifacts.
-   **Workflow**:
    1.  Run `.specify/scripts/bash/setup-plan.sh --json` to get file paths.
    2.  Analyze the feature specification (`spec.md`) and constitution (`constitution.md`).
    3.  Execute the `.specify/templates/plan-template.md` to generate `research.md`, `data-model.md`, `contracts/`, `quickstart.md`, and `tasks.md`.

### `tasks`
-   **Description**: Generate an actionable, dependency-ordered `tasks.md` file from design artifacts.
-   **Workflow**:
    1.  Run `.specify/scripts/bash/check-task-prerequisites.sh --json` to find available design documents.
    2.  Analyze `plan.md`, `data-model.md`, `contracts/`, etc.
    3.  Use `.specify/templates/tasks-template.md` to generate a dependency-ordered list of specific, executable tasks.
