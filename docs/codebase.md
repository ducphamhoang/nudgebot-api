# NudgeBot API Codebase

This document provides a summary of the NudgeBot API codebase, outlining the structure and purpose of its various components.

## Project Overview

NudgeBot is a conversational task assistant for Telegram that helps users manage tasks with proactive reminders. This codebase implements the backend services for NudgeBot, including the API for handling Telegram webhooks, core business logic for task management, and integration with an LLM for natural language processing.

## Directory Structure

The project follows a structure inspired by the standard Go project layout and Clean Architecture principles.

### Top-Level Directories

-   **/api**: Contains the HTTP transport layer, including request handlers, middleware, and route definitions. It is the primary entry point for external requests.
-   **/cmd**: The main application entry point. The `server` subdirectory initializes and starts the web server.
-   **/configs**: Holds configuration files, with `config.yaml` being the primary configuration source.
-   **/demo**: Contains demonstration code and examples.
-   **/docs**: Project documentation, including architecture, codebase summary, and implementation plans.
-   **/internal**: Core application logic, separated by domain. This is where the main business logic resides.
-   **/pkg**: Shared libraries and utilities that can be used across the application, such as the logger.
-   **/test**: Integration and end-to-end tests.

### `internal` Directory

The `internal` directory is organized by domain, each with a specific responsibility:

-   **/internal/chatbot**: Manages interactions with the chatbot platform (Telegram), including webhook parsing, command processing, and sending messages.
-   **/internal/common**: Provides common types and utility functions used across different domains.
-   **/internal/config**: Handles application configuration loading and management.
-e   **/internal/database**: Manages the database connection and GORM setup.
-   **/internal/events**: Implements an event bus for decoupled communication between different parts of the application.
-   **/internal/llm**: Integrates with a Large Language Model (LLM) for natural language understanding and processing tasks.
-   **/internal/mocks**: Contains generated mocks for testing purposes.
-   **/internal/nudge**: Implements the core business logic for managing nudges and tasks, including the repository for database interactions.
-   **/internal/scheduler**: A background job scheduler for sending proactive nudges and reminders.

### Testing

-   **Unit Tests**: Each package in the `internal` directory contains its own unit tests (e.g., `service_test.go`).
-   **Integration Tests**: The root directory contains integration tests (e.g., `integration_test.go`) that test the interaction between different components.
