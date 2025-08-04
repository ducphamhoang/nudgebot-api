# NudgeBot MVP - Revised Test Plan (Phased Approach)

This document outlines a prioritized, phased approach for implementing a comprehensive test suite for the NudgeBot MVP. The plan focuses on delivering testing value iteratively, starting with the most critical product features as defined in the PRD.

---

### **Phase 1: Core Infrastructure & "Message-to-Task" Flow**

**Goal:** Establish the essential testing infrastructure and create a full end-to-end test for the most critical user path: a user sending a message and a task being created in the database.

1.  **Setup Testing Dependencies (`go.mod`):**
    *   Add the necessary dependencies for mocking and integration testing:
        *   `github.com/golang/mock/mockgen`
        *   `github.com/testcontainers/testcontainers-go` and its `postgres` module.

2.  **Generate Core Mocks (`mockgen`):**
    *   **Action:** Replace existing manual mocks with `mockgen`-generated ones for consistency.
    *   **Priority Mocks:**
        *   `NudgeRepository`: To isolate service logic from the database in unit tests.
        *   `EventBus`: To test services that publish or subscribe to events.
        *   `TelegramProvider`: To test the `chatbot.service` without making real API calls.
        *   An `HTTPClient` mock: To test the `llm.service` without calling the actual Gemma API.

3.  **Create "Message-to-Task" Integration Test (`integration/message_flow_test.go`):**
    *   **Action:** Create a *new* integration test file in a new `integration/` directory. This test will not affect the existing root-level integration tests.
    *   **Scenario:**
        1.  Start a real PostgreSQL database using `testcontainers-go`.
        2.  Initialize all services (`chatbot`, `llm`, `nudge`) connected to a real event bus and the test database.
        3.  Simulate a Telegram webhook arriving at the `/api/v1/telegram/webhook` endpoint with a natural language task (e.g., "call mom tomorrow at 5pm").
        4.  **Assert:** Verify that a corresponding `Task` is created correctly in the test database with the right `title`, `due_date`, and `user_id`.
    *   **Coverage:** This single test will validate the entire event-driven flow: `api` -> `chatbot` -> `events` -> `llm` -> `events` -> `nudge` -> `database`.

4.  **Add Supporting Unit Tests:**
    *   `api/handlers/webhook_test.go`: Unit test the webhook handler, using a mock `ChatbotService` to ensure it correctly processes requests and handles errors.
    *   `internal/llm/service_test.go`: Enhance tests to ensure the `MessageReceived` event handler works correctly, using a mock `LLMProvider`.

---

### **Phase 2: Reminder, Nudge & Task Action Flows**

**Goal:** Build upon the infrastructure from Phase 1 to test the scheduler-driven logic and user interactions with tasks.

1.  **Create "Reminder & Nudge" Integration Test (`integration/reminder_flow_test.go`):**
    *   **Action:** Add a new integration test to the `integration/` directory.
    *   **Scenario:**
        1.  Set up the test database and services as in Phase 1.
        2.  Manually insert a `Task` into the database that is due in the near future.
        3.  Start the `scheduler` service.
        4.  **Assert (Reminder):** Verify that the scheduler picks up the task and a `ReminderDue` event is published. Use a mock `TelegramProvider` to confirm a reminder message would be sent.
        5.  **Assert (Nudge):** Manually set a task's `due_date` to be 3 hours in the past. Run the scheduler again and verify that a "nudge" reminder is triggered.

2.  **Create "Task Actions" Integration Test (`integration/task_actions_test.go`):**
    *   **Action:** Add a new integration test to the `integration/` directory.
    *   **Scenario:**
        1.  Insert a task into the test database.
        2.  Simulate incoming callback queries for actions like "Done" and "Snooze".
        3.  **Assert:** Verify the task's `status` or `due_date` is updated correctly in the database.
        4.  Simulate a `/list` command and verify the correct tasks are returned.

3.  **Add Supporting Unit Tests:**
    *   `internal/scheduler/scheduler_test.go`: Create comprehensive unit tests for the scheduler. Use a mock `NudgeRepository` to provide it with test data (e.g., due tasks, overdue tasks) and a mock `EventBus` to verify that it publishes events correctly.
    *   `internal/nudge/service_test.go`: Enhance tests to cover the event handlers for `TaskListRequested` and `TaskActionRequested`.

---

### **Phase 3: Comprehensive Coverage, CI & Documentation**

**Goal:** Finalize the test suite by covering utility packages, implementing the CI pipeline, and updating documentation.

1.  **Add Remaining Unit Tests:**
    *   Create the unit tests for the lower-level, less critical (but still important) packages as outlined in the original plan:
        *   `internal/events/bus_test.go`
        *   `internal/common/`
        *   `pkg/logger/logger_test.go`
        *   `internal/config/config_test.go`
        *   `internal/database/postgres_test.go` (can use `testcontainers` here too).

2.  **Implement CI/CD Pipeline (`.github/workflows/ci.yml`):**
    *   Create the GitHub Actions workflow to automate the running of all tests (unit and integration) on each push.

3.  **Enhance Build System (`Makefile`):**
    *   Implement the new targets (`test-integration`, `test-coverage-html`, `test-all`, etc.) to make running tests locally easy and consistent.

4.  **Update Documentation (`README.md`):**
    *   Rewrite the `README.md` to reflect the modular architecture and provide clear instructions on how to run the newly created test suite.
