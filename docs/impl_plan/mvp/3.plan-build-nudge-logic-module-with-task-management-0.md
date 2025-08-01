I have created the following plan after thorough exploration and analysis of the codebase. Follow the below plan verbatim. Trust the files and references. Do not re-verify what's written in the plan. Explore only when absolutely necessary. First implement all the proposed file changes and then I'll review all the changes together at the end.

### Observations

The nudge module has excellent foundations with comprehensive domain models including proper GORM tags, well-designed repository interfaces following clean architecture patterns, and a service layer with event-driven communication. The main.go initializes services but passes nil for the nudge repository. The database connection is established using GORM with PostgreSQL, and the configuration system supports nudge-specific settings. The missing pieces are the actual GORM repository implementation, database migrations, and integration with the service initialization.

### Approach

I'll implement the nudge module with a complete GORM repository implementation, database migrations, and business logic. The approach focuses on:

- **GORM Repository Implementation**: Create a concrete repository that implements all interfaces using GORM for database operations
- **Database Migrations**: Implement auto-migration functionality to create tables for tasks, reminders, and nudge settings
- **CRUD Operations**: Full implementation of Create, Read, Update, Delete operations with proper error handling
- **Task Status Management**: Business logic for handling task status transitions (active, completed, snoozed, deleted)
- **Transaction Support**: Implement transaction handling for complex operations
- **Integration**: Wire the repository into the service initialization in main.go

The implementation will follow clean architecture principles with proper separation of concerns, comprehensive error handling, and support for the existing event-driven architecture.

### Reasoning

I explored the current nudge module structure and found well-defined domain models, repository interfaces, and service layer with event handling. I examined the database connection setup using GORM with PostgreSQL, configuration system, and main server initialization. The analysis revealed that while the foundation is solid with proper interfaces and domain models, the actual GORM implementation is missing, and the service currently uses mock implementations when repository is nil.

## Mermaid Diagram

sequenceDiagram
    participant Main as main.go
    participant DB as GORM Database
    participant Repo as GormNudgeRepository
    participant Service as NudgeService
    participant BL as Business Logic
    participant EB as EventBus
    
    Note over Main,EB: Server Initialization
    Main->>DB: NewPostgresConnection()
    Main->>DB: RunMigrations() - Create tables & indexes
    Main->>Repo: NewGormNudgeRepository(db, logger)
    Main->>Service: NewNudgeService(eventBus, logger, repository)
    Service->>BL: Initialize TaskValidator, ReminderManager, StatusManager
    Service->>EB: Subscribe to TaskParsed events
    
    Note over Service,EB: Task Creation Flow
    EB->>Service: TaskParsed event
    Service->>BL: ValidateTask(parsedTask)
    BL-->>Service: Validation result
    Service->>Repo: CreateTask(task)
    Repo->>DB: INSERT INTO tasks
    DB-->>Repo: Task created
    Repo-->>Service: Success
    Service->>BL: CalculateReminderTime(task, settings)
    Service->>Repo: CreateReminder(reminder)
    Repo->>DB: INSERT INTO reminders
    Service->>EB: Publish TaskCreated event
    
    Note over Service,DB: Task Status Management
    Service->>BL: TransitionStatus(task, newStatus)
    BL->>BL: ValidateStatusTransition()
    BL-->>Service: Transition approved
    Service->>Repo: UpdateTask(task)
    Repo->>DB: UPDATE tasks SET status, updated_at
    Service->>EB: Publish TaskCompleted event (if completed)
    
    Note over Repo,DB: Query Operations
    Service->>Repo: GetTasksByUserID(userID, filter)
    Repo->>DB: SELECT with WHERE, ORDER BY, LIMIT
    DB-->>Repo: Task results
    Repo->>Repo: Apply business logic filtering
    Repo-->>Service: Filtered tasks
    
    Note over Service,DB: Reminder Processing
    Service->>Repo: GetDueReminders(before)
    Repo->>DB: SELECT reminders JOIN tasks WHERE scheduled_at <= ?
    DB-->>Repo: Due reminders
    Service->>EB: Publish ReminderDue events
    Service->>Repo: MarkReminderSent(reminderID)
    Repo->>DB: UPDATE reminders SET sent_at

## Proposed File Changes

### internal/nudge/gorm_repository.go(NEW)

References: 

- internal/nudge/domain.go
- internal/nudge/repository.go
- internal/common/types.go

Create the GORM implementation of the NudgeRepository interface:

- Implement `gormNudgeRepository` struct with GORM database instance and logger
- Constructor `NewGormNudgeRepository(db *gorm.DB, logger *zap.Logger) NudgeRepository`
- **Task operations**: Implement all TaskRepository methods:
  - `CreateTask`: Insert new task with validation and duplicate checking
  - `GetTaskByID`: Retrieve task by ID with proper error handling for not found
  - `GetTasksByUserID`: Query tasks with filtering support (status, priority, due dates) and pagination
  - `UpdateTask`: Update existing task with optimistic locking using UpdatedAt
  - `DeleteTask`: Soft delete by updating status to deleted
  - `GetTaskStats`: Aggregate query to calculate task statistics by status
- **Reminder operations**: Implement all ReminderRepository methods:
  - `CreateReminder`: Insert reminder with validation
  - `GetDueReminders`: Query reminders due before specified time with joins to tasks
  - `MarkReminderSent`: Update reminder with sent timestamp
  - `GetRemindersByTaskID`: Retrieve all reminders for a specific task
  - `DeleteReminder`: Hard delete reminder record
- **Nudge settings operations**: Implement NudgeSettingsRepository methods:
  - `GetNudgeSettingsByUserID`: Retrieve user settings with defaults if not found
  - `CreateOrUpdateNudgeSettings`: Upsert operation using GORM's Save method
  - `DeleteNudgeSettings`: Remove user settings
- **Transaction support**: Implement `WithTransaction` method using GORM's transaction API
- Include comprehensive error handling, logging, and proper GORM query building
- Use preloading for related data and proper indexing for performance

Reference the domain models from `internal/nudge/domain.go`, repository interfaces from `internal/nudge/repository.go`, and common error types from `internal/common/types.go`.

### internal/nudge/migrations.go(NEW)

References: 

- internal/nudge/domain.go

Create database migration functionality for the nudge module:

- Implement `RunMigrations(db *gorm.DB) error` function that performs auto-migration for all nudge-related tables
- Use GORM's `AutoMigrate` method to create tables for `Task`, `Reminder`, and `NudgeSettings` models
- Add database indexes for performance:
  - Index on `tasks.user_id` for user-specific queries
  - Index on `tasks.status` for status filtering
  - Index on `tasks.due_date` for due date queries
  - Composite index on `tasks(user_id, status)` for common filter combinations
  - Index on `reminders.scheduled_at` for due reminder queries
  - Index on `reminders.task_id` for task-specific reminder lookups
- Include proper error handling and logging for migration failures
- Add validation to ensure all required tables and indexes are created successfully
- Include migration rollback functionality for development purposes
- Add helper function `DropTables(db *gorm.DB) error` for testing cleanup

The migration should be idempotent and safe to run multiple times. Reference the domain models from `internal/nudge/domain.go` for table structure.

### internal/nudge/business_logic.go(NEW)

References: 

- internal/nudge/domain.go
- internal/common/types.go

Create business logic utilities and validators for the nudge module:

- Implement `TaskValidator` struct with methods:
  - `ValidateTask(task *Task) error`: Comprehensive task validation including required fields, due date logic, priority validation
  - `ValidateStatusTransition(from, to TaskStatus) error`: Ensure valid status transitions (e.g., can't go from completed to active)
  - `ValidateTaskFilter(filter TaskFilter) error`: Validate filter parameters for queries
- Implement `ReminderManager` struct with methods:
  - `CalculateReminderTime(task *Task, settings *NudgeSettings) time.Time`: Calculate when to send reminders based on due date and user preferences
  - `ShouldCreateNudge(task *Task, reminderCount int, settings *NudgeSettings) bool`: Determine if a nudge should be created
  - `GetNextNudgeTime(lastNudge time.Time, settings *NudgeSettings) time.Time`: Calculate next nudge time with exponential backoff
- Implement `TaskStatusManager` with methods:
  - `TransitionStatus(task *Task, newStatus TaskStatus) error`: Handle status transitions with business rules
  - `CompleteTask(task *Task) error`: Complete task and set completion timestamp
  - `SnoozeTask(task *Task, snoozeUntil time.Time) error`: Snooze task and reschedule reminders
  - `DeleteTask(task *Task) error`: Soft delete task and cancel associated reminders
- Include business rule constants and configuration
- Add comprehensive error handling with specific business logic error types
- Include helper functions for common operations like overdue detection and priority sorting

Reference the domain models from `internal/nudge/domain.go` and common types from `internal/common/types.go`.

### internal/nudge/service.go(MODIFY)

References: 

- internal/nudge/business_logic.go(NEW)
- internal/nudge/gorm_repository.go(NEW)
- internal/nudge/repository.go

Update the nudge service to use the new business logic utilities and enhance the existing implementation:

- Add `validator *TaskValidator`, `reminderManager *ReminderManager`, and `statusManager *TaskStatusManager` fields to the `nudgeService` struct
- Update the `NewNudgeService` constructor to initialize the business logic components and accept a non-nil repository
- Enhance the `CreateTask` method to:
  - Use `validator.ValidateTask()` before creation
  - Use `reminderManager.CalculateReminderTime()` for automatic reminder scheduling
  - Publish appropriate events after successful creation
- Update the `UpdateTaskStatus` method to:
  - Use `statusManager.TransitionStatus()` for proper status transitions
  - Handle completion, snoozing, and deletion with appropriate business logic
  - Update reminder schedules when status changes
- Enhance the `GetTasks` method to:
  - Use `validator.ValidateTaskFilter()` for input validation
  - Add sorting and pagination support
  - Include overdue task detection
- Add new methods:
  - `SnoozeTask(taskID, snoozeUntil)` for task snoozing functionality
  - `GetOverdueTasks(userID)` for retrieving overdue tasks
  - `BulkUpdateStatus(taskIDs, status)` for batch operations
- Enhance the `handleTaskParsed` event handler to use the new validation and business logic
- Remove all mock implementations and ensure all operations use the repository
- Add comprehensive error handling and logging throughout

Maintain backward compatibility with the existing service interface while adding the new functionality. Reference the business logic from `internal/nudge/business_logic.go` and repository from `internal/nudge/gorm_repository.go`.

### cmd/server/main.go(MODIFY)

References: 

- internal/nudge/migrations.go(NEW)
- internal/nudge/gorm_repository.go(NEW)

Update the main server initialization to integrate the nudge repository and run migrations:

- After the database connection is established (line 40-43), add migration execution:
  ```go
  // Run nudge module migrations
  if err := nudge.RunMigrations(db); err != nil {
      logger.Fatal("Failed to run nudge migrations", "error", err)
  }
  ```
- Update the nudge service initialization (line 51) to create and pass the GORM repository:
  ```go
  nudgeRepository := nudge.NewGormNudgeRepository(db, zapLogger)
  nudgeService := nudge.NewNudgeService(eventBus, zapLogger, nudgeRepository)
  ```
- Ensure the database instance is available for repository creation
- Maintain the existing error handling and logging patterns
- Keep all other service initializations unchanged
- Add import for the nudge package if not already present

The changes should be minimal and focused on integrating the repository while maintaining the existing server startup flow. Reference the migration function from `internal/nudge/migrations.go` and repository constructor from `internal/nudge/gorm_repository.go`.

### internal/nudge/errors.go(NEW)

References: 

- internal/common/types.go

Create comprehensive error handling for the nudge module:

- Define `NudgeError` interface with methods `Code() string`, `Message() string`, and `Temporary() bool`
- Implement specific error types:
  - `TaskValidationError` for task validation failures with field-specific details
  - `StatusTransitionError` for invalid status transitions with current and target status
  - `ReminderSchedulingError` for reminder creation failures
  - `BusinessRuleError` for business logic violations
  - `RepositoryError` for database operation failures with operation context
- Define error codes as constants:
  - `ErrCodeTaskNotFound`, `ErrCodeInvalidStatus`, `ErrCodeInvalidTransition`
  - `ErrCodeValidationFailed`, `ErrCodeDuplicateTask`, `ErrCodeReminderFailed`
- Implement error wrapping utilities:
  - `WrapRepositoryError(err error, operation string) error`
  - `WrapValidationError(err error, field string) error`
  - `NewBusinessRuleError(rule string, details string) error`
- Add error classification helpers:
  - `IsNotFoundError(err error) bool`
  - `IsValidationError(err error) bool`
  - `IsTemporaryError(err error) bool`
- Include error context preservation for debugging and correlation ID tracking
- Add error conversion utilities for HTTP status code mapping

The error system should provide clear categorization for different failure modes and integrate with the existing common error types. Reference the common error patterns from `internal/common/types.go`.

### internal/mocks/nudge_mocks.go(NEW)

References: 

- internal/nudge/repository.go
- internal/nudge/domain.go
- internal/mocks/interfaces.go

Create comprehensive mock implementations for testing the nudge module:

- Add `//go:generate mockgen -source=../nudge/repository.go -destination=nudge_repository_mocks.go -package=mocks` directive
- Implement `MockNudgeRepository` struct with in-memory storage using maps:
  - `tasks map[string]*Task` for task storage
  - `reminders map[string]*Reminder` for reminder storage
  - `settings map[string]*NudgeSettings` for user settings
  - `mutex sync.RWMutex` for thread safety
- Implement all repository interface methods with realistic behavior:
  - Task CRUD operations with proper error simulation
  - Reminder management with due date filtering
  - Settings management with upsert behavior
  - Transaction support with rollback simulation
- Add helper methods for test setup:
  - `SetupTestData()` to populate with sample data
  - `ClearData()` to reset state between tests
  - `SetError(operation, error)` to simulate specific failures
  - `GetCallCount(operation)` to verify method calls
- Implement `MockTaskValidator`, `MockReminderManager`, and `MockStatusManager` for business logic testing
- Include factory methods for creating test scenarios:
  - `CreateTestTask(userID, title)` for quick task creation
  - `CreateOverdueTask(userID)` for overdue scenarios
  - `CreateCompletedTask(userID)` for completed task testing
- Add assertion helpers for common test patterns

The mocks should support both unit testing of individual components and integration testing of the service layer. Reference the repository interfaces from `internal/nudge/repository.go` and domain models from `internal/nudge/domain.go`.

### internal/nudge/queries.go(NEW)

References: 

- internal/nudge/domain.go
- internal/nudge/gorm_repository.go(NEW)

Create optimized database queries and query builders for the nudge module:

- Implement `QueryBuilder` struct with methods for building complex GORM queries:
  - `TaskQuery(db *gorm.DB) *TaskQueryBuilder` for task-specific query building
  - `ReminderQuery(db *gorm.DB) *ReminderQueryBuilder` for reminder queries
  - Support for method chaining: `.WithUserID()`, `.WithStatus()`, `.WithPriority()`, `.WithDueDateRange()`
- Define optimized query methods:
  - `GetTasksWithReminders(userID, filter)` with proper joins and preloading
  - `GetOverdueTasksWithCounts(userID)` for dashboard statistics
  - `GetTasksDueSoon(userID, within)` for proactive reminders
  - `GetUserTaskSummary(userID)` for comprehensive user statistics
- Implement batch operations:
  - `BulkUpdateTaskStatus(taskIDs, status)` for efficient bulk updates
  - `BulkCreateReminders(reminders)` for batch reminder creation
  - `CleanupOldReminders(before)` for maintenance operations
- Add query optimization utilities:
  - Proper indexing hints and query planning
  - Pagination helpers with cursor-based pagination support
  - Sorting utilities with multiple sort criteria
- Include raw SQL queries for complex analytics:
  - Task completion rate calculations
  - User engagement metrics
  - Reminder effectiveness statistics
- Add query performance monitoring and logging
- Include query result caching strategies for frequently accessed data

The queries should be optimized for performance with proper indexing and minimal N+1 query problems. Reference the domain models from `internal/nudge/domain.go` and repository patterns from `internal/nudge/gorm_repository.go`.