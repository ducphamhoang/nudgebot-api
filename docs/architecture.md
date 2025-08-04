# NudgeBot API Architecture

This document describes the architecture of the NudgeBot API, a system designed to be modular, scalable, and maintainable.

## Architectural Style

The NudgeBot API is built upon the principles of **Clean Architecture**. This approach emphasizes a separation of concerns, creating a system that is independent of frameworks, UI, and the database. The core of the application is the business logic, with external dependencies and frameworks treated as plugins.

The architecture is also **event-driven**, using an internal event bus to decouple the different domains of the application.

## Core Components

The application is divided into several layers, consistent with Clean Architecture:

1.  **Domain Layer**: At the very center, this layer contains the core business logic and entities (e.g., `Task`, `Nudge`). It is the most independent part of the application. This is primarily located in the `internal/*/domain.go` files.

2.  **Application/Service Layer**: This layer orchestrates the use cases of the application. It contains the business logic that is specific to the application and depends on the domain layer. The services in `internal/nudge`, `internal/chatbot`, and `internal/llm` represent this layer.

3.  **Interface/Adapter Layer**: This layer adapts external technologies to be used by the application layer. This includes the database repository implementation (`internal/nudge/gorm_repository.go`) and the chatbot provider (`internal/chatbot/telegram_provider.go`).

4.  **Infrastructure/Frameworks Layer**: The outermost layer, containing the web framework (Gin), database drivers, and other external libraries. The `api` directory, `cmd/server`, and `internal/database` are part of this layer.

## Request Flow Example: New Task from Telegram

A typical request flow for creating a new task illustrates how the components interact:

1.  **Webhook**: A user sends a message to the Telegram bot. Telegram sends a webhook to the NudgeBot API at `/api/v1/telegram/webhook`.
2.  **API Layer**: The `webhookHandler` in the `api/handlers` directory receives the request.
3.  **Chatbot Service**: The handler calls the `chatbot.Service`, which parses the incoming webhook and extracts the user's message and chat ID.
4.  **Event Bus**: The `chatbot.Service` determines that a new task needs to be created and publishes a `TaskReceived` event to the event bus.
5.  **Nudge Service**: The `nudge.Service` is subscribed to `TaskReceived` events. Upon receiving the event, it processes the message.
6.  **LLM Service**: The `nudge.Service` may call the `llm.Service` to parse the natural language message into a structured task with a due date.
7.  **Repository**: Once the task is structured, the `nudge.Service` uses its `Repository` interface to save the task to the PostgreSQL database.
8.  **Response**: The `chatbot.Service` sends a confirmation message back to the user via the `chatbot.Provider`.

This decoupled flow, mediated by the event bus, allows for clear separation of concerns and makes the system easier to extend.

## Text-Based Diagram

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

## Technology Stack

-   **Language**: Go
-   **Web Framework**: Gin
-   **Database**: PostgreSQL
-   **ORM**: GORM
-   **Event Bus**: A simple in-memory event bus
-   **Configuration**: Viper
-   **Logging**: Zap
-   **Containerization**: Docker & Docker Compose
