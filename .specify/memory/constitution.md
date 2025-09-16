# NudgeBot API Constitution

## Core Principles

### I. Clean Architecture & Domain-Driven Design (DDD)
NudgeBot is built upon the principles of Clean Architecture, ensuring a clear separation of concerns. The core business logic (domain) is independent of frameworks, UI, and the database. This is complemented by DDD patterns, with the application's core logic organized into domains like `chatbot`, `nudge`, `llm`, and `scheduler`.

### II. Event-Driven for Decoupling
The system is event-driven, using an internal event bus (`internal/events`) to decouple domains. This allows services to communicate asynchronously, promoting scalability and maintainability. For example, a new task from Telegram is processed by the `chatbot` service, which then publishes an event that the `nudge` service consumes.

### III. Comprehensive & Multi-Tiered Testing
Testing is a cornerstone of the project, with a three-tier approach:
1.  **Unit Tests**: Focused on isolated logic within a service, using mocks for dependencies.
2.  **Integration Tests**: Validate cross-service flows and interactions with real services like PostgreSQL (via Testcontainers) and the event bus.
3.  **End-to-End Tests**: Cover complete user journeys, ensuring the system works cohesively.
A high test coverage (>90% for unit tests) is expected, and all PRs must include appropriate tests.

### IV. Makefile-Driven Workflow
The `Makefile` serves as the primary entry point for all common development tasks. It provides a consistent and reproducible interface for building, testing, linting, and running the application. Key commands include `make dev`, `make test-all`, `make lint`, and `make precommit`.

### V. CI/CD for Quality Assurance
A comprehensive CI/CD pipeline on GitHub Actions automatically enforces quality gates on every push and pull request. The pipeline includes steps for code formatting (`make fmt`), linting (`golangci-lint`), security audits (`nancy`), unit tests, and integration tests. All checks must pass before code can be merged.

## Technology Stack

The project utilizes a modern, robust technology stack:
- **Language**: Go (1.21+)
- **Web Framework**: Gin
- **Database**: PostgreSQL (with GORM)
- **Caching & Messaging**: Redis
- **Testing**: Testify, Gomock, and Testcontainers
- **Configuration**: Viper
- **Logging**: Zap
- **Containerization**: Docker and Docker Compose

## Development Workflow & Quality Gates

The standard development workflow is as follows:
1.  Create a feature branch.
2.  Implement changes, adding or updating tests accordingly.
3.  Use `make test-essential-services` or `make test-unit` for quick validation during development.
4.  Before committing, run `make precommit` which executes `make fmt`, `make lint`, and `make test-unit`.
5.  For pull requests, `make test-all` should be run to ensure all unit and integration tests pass.
6.  The CI pipeline will run all checks again, and must pass for a PR to be merged.

## Governance

- This Constitution supersedes all other practices.
- All pull requests and code reviews must verify compliance with these principles.
- All new features must be accompanied by tests that cover the new functionality.
- The `README.md` and `docs/` directory should be kept up-to-date with any changes to the architecture or workflow.

**Version**: 1.0.0 | **Ratified**: 2025-09-16 | **Last Amended**: 2025-09-16
