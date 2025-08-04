package nudge

import (
	"fmt"
	"sync"
	"time"

	"nudgebot-api/internal/common"
	"nudgebot-api/internal/events"

	"go.uber.org/zap"
)

// NudgeService defines the interface for nudge operations
type NudgeService interface {
	CreateTask(task *Task) error
	GetTasks(userID common.UserID, filter TaskFilter) ([]*Task, error)
	UpdateTaskStatus(taskID common.TaskID, status common.TaskStatus) error
	DeleteTask(taskID common.TaskID) error
	GetTaskStats(userID common.UserID) (*TaskStats, error)
	ScheduleReminder(taskID common.TaskID, scheduledAt time.Time, reminderType ReminderType) error
	GetNudgeSettings(userID common.UserID) (*NudgeSettings, error)
	UpdateNudgeSettings(settings *NudgeSettings) error

	// Additional methods
	SnoozeTask(taskID common.TaskID, snoozeUntil time.Time) error
	GetOverdueTasks(userID common.UserID) ([]*Task, error)
	BulkUpdateStatus(taskIDs []common.TaskID, status common.TaskStatus) error

	// Health check methods
	CheckSubscriptionHealth() error
}

// nudgeService implements the NudgeService interface
type nudgeService struct {
	eventBus        events.EventBus
	logger          *zap.Logger
	repository      NudgeRepository
	validator       *TaskValidator
	reminderManager *ReminderManager
	statusManager   *TaskStatusManager

	// Subscription tracking
	subscriptions map[string]bool
	mu            sync.RWMutex
}

// NewNudgeService creates a new instance of NudgeService
func NewNudgeService(eventBus events.EventBus, logger *zap.Logger, repository NudgeRepository) (NudgeService, error) {
	if repository == nil {
		logger.Warn("NudgeService initialized with nil repository - using mock behavior")
	}

	service := &nudgeService{
		eventBus:        eventBus,
		logger:          logger,
		repository:      repository,
		validator:       NewTaskValidator(),
		reminderManager: NewReminderManager(),
		statusManager:   NewTaskStatusManager(),
		subscriptions:   make(map[string]bool),
		mu:              sync.RWMutex{},
	}

	// Subscribe to relevant events with retry logic
	if err := service.setupEventSubscriptions(); err != nil {
		logger.Error("Failed to setup event subscriptions", zap.Error(err))
		return nil, err
	}

	return service, nil
}

// setupEventSubscriptions sets up event subscriptions for the nudge service with retry logic
func (s *nudgeService) setupEventSubscriptions() error {
	requiredSubscriptions := map[string]interface{}{
		events.TopicTaskParsed:          s.handleTaskParsed,
		events.TopicTaskListRequested:   s.handleTaskListRequested,
		events.TopicTaskActionRequested: s.handleTaskActionRequested,
	}

	maxRetries := 3
	baseDelay := 100 * time.Millisecond
	maxDelay := 5 * time.Second

	var failedSubscriptions []string

	for topic, handler := range requiredSubscriptions {
		if err := s.subscribeWithRetry(topic, handler, maxRetries, baseDelay, maxDelay); err != nil {
			s.logger.Error("Failed to subscribe to topic after retries",
				zap.String("topic", topic),
				zap.Error(err),
				zap.Int("max_retries", maxRetries))
			failedSubscriptions = append(failedSubscriptions, topic)
		} else {
			s.markSubscriptionActive(topic)
			s.logger.Info("Successfully subscribed to topic", zap.String("topic", topic))
		}
	}

	if len(failedSubscriptions) > 0 {
		return NewSubscriptionError(
			fmt.Sprintf("%d topics", len(failedSubscriptions)),
			fmt.Sprintf("failed to subscribe to critical topics: %v", failedSubscriptions),
			true,
		)
	}

	s.logger.Info("All event subscriptions established successfully")
	return nil
}

// subscribeWithRetry attempts to subscribe to a topic with exponential backoff retry logic
func (s *nudgeService) subscribeWithRetry(topic string, handler interface{}, maxRetries int, baseDelay, maxDelay time.Duration) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff
			multiplier := 1 << uint(attempt-1) // 2^(attempt-1)
			delay := time.Duration(int64(baseDelay) * int64(multiplier))
			if delay > maxDelay {
				delay = maxDelay
			}

			s.logger.Warn("Retrying subscription",
				zap.String("topic", topic),
				zap.Int("attempt", attempt),
				zap.Duration("delay", delay))

			time.Sleep(delay)
		}

		err := s.eventBus.Subscribe(topic, handler)
		if err == nil {
			if attempt > 0 {
				s.logger.Info("Subscription succeeded after retry",
					zap.String("topic", topic),
					zap.Int("attempt", attempt))
			}
			return nil
		}

		lastErr = err
		s.logger.Warn("Subscription attempt failed",
			zap.String("topic", topic),
			zap.Int("attempt", attempt+1),
			zap.Int("max_retries", maxRetries+1),
			zap.Error(err))
	}

	return fmt.Errorf("failed to subscribe to topic '%s' after %d attempts: %w", topic, maxRetries+1, lastErr)
}

// markSubscriptionActive marks a topic subscription as active
func (s *nudgeService) markSubscriptionActive(topic string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscriptions[topic] = true
}

// CheckSubscriptionHealth verifies that all required subscriptions are active
func (s *nudgeService) CheckSubscriptionHealth() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	requiredTopics := []string{
		events.TopicTaskParsed,
		events.TopicTaskListRequested,
		events.TopicTaskActionRequested,
	}

	var missingTopics []string
	for _, topic := range requiredTopics {
		if !s.subscriptions[topic] {
			missingTopics = append(missingTopics, topic)
		}
	}

	if len(missingTopics) > 0 {
		return NewSubscriptionHealthError(
			missingTopics,
			fmt.Sprintf("%d required subscriptions are not active", len(missingTopics)),
		)
	}

	s.logger.Debug("Subscription health check passed",
		zap.Int("active_subscriptions", len(s.subscriptions)))

	return nil
}

// CreateTask creates a new task
func (s *nudgeService) CreateTask(task *Task) error {
	s.logger.Info("Creating task",
		zap.String("userID", string(task.UserID)),
		zap.String("title", task.Title))

	// Validate task using business logic
	if err := s.validator.ValidateTask(task); err != nil {
		s.logger.Error("Task validation failed", zap.Error(err))
		return err
	}

	if task.ID == "" {
		task.ID = common.TaskID(common.NewID())
	}

	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	if s.repository != nil {
		err := s.repository.CreateTask(task)
		if err != nil {
			s.logger.Error("Failed to create task in repository", zap.Error(err))
			return err
		}

		// Schedule initial reminder if due date is set
		if task.DueDate != nil {
			go s.scheduleInitialReminder(task)
		}

		// Publish TaskCreated event
		event := events.TaskCreated{
			Event:     events.NewEvent(),
			TaskID:    string(task.ID),
			UserID:    string(task.UserID),
			Title:     task.Title,
			DueDate:   task.DueDate,
			Priority:  string(task.Priority),
			CreatedAt: task.CreatedAt,
		}
		s.eventBus.Publish(events.TopicTaskCreated, event)

		s.logger.Info("Task created successfully", zap.String("taskID", string(task.ID)))
		return nil
	}

	// Mock implementation when repository is nil
	s.logger.Info("Task created successfully (mock)", zap.String("taskID", string(task.ID)))
	return nil
}

// GetTasks retrieves tasks for a user with optional filtering
func (s *nudgeService) GetTasks(userID common.UserID, filter TaskFilter) ([]*Task, error) {
	s.logger.Info("Getting tasks",
		zap.String("userID", string(userID)),
		zap.Any("filter", filter))

	// Validate filter using business logic
	if err := s.validator.ValidateTaskFilter(filter); err != nil {
		s.logger.Error("Task filter validation failed", zap.Error(err))
		return nil, err
	}

	if s.repository != nil {
		tasks, err := s.repository.GetTasksByUserID(userID, filter)
		if err != nil {
			s.logger.Error("Failed to get tasks from repository", zap.Error(err))
			return nil, err
		}

		// Enhance tasks with overdue detection
		for _, task := range tasks {
			if task.IsOverdue() {
				s.logger.Debug("Task is overdue", zap.String("taskID", string(task.ID)))
			}
		}

		return tasks, nil
	}

	// Mock implementation when repository is nil
	return []*Task{}, nil
}

// UpdateTaskStatus updates the status of a task
func (s *nudgeService) UpdateTaskStatus(taskID common.TaskID, status common.TaskStatus) error {
	s.logger.Info("Updating task status",
		zap.String("taskID", string(taskID)),
		zap.String("status", string(status)))

	if s.repository != nil {
		task, err := s.repository.GetTaskByID(taskID)
		if err != nil {
			s.logger.Error("Failed to get task for status update", zap.Error(err))
			return err
		}

		// Use status manager for proper status transitions
		if err := s.statusManager.TransitionStatus(task, status); err != nil {
			s.logger.Error("Status transition failed", zap.Error(err))
			return err
		}

		// Update task in repository
		if err := s.repository.UpdateTask(task); err != nil {
			s.logger.Error("Failed to update task in repository", zap.Error(err))
			return err
		}

		// Handle status-specific actions
		switch status {
		case common.TaskStatusCompleted:
			// Cancel future reminders for completed task
			go s.cancelTaskReminders(taskID)

			// Publish TaskCompleted event
			event := events.TaskCompleted{
				Event:       events.NewEvent(),
				TaskID:      string(taskID),
				UserID:      string(task.UserID),
				CompletedAt: *task.CompletedAt,
			}
			s.eventBus.Publish(events.TopicTaskCompleted, event)

		case common.TaskStatusDeleted:
			// Cancel all reminders for deleted task
			go s.cancelTaskReminders(taskID)

		case common.TaskStatusActive:
			// If reactivating, schedule new reminders
			if task.DueDate != nil {
				go s.scheduleInitialReminder(task)
			}
		}

		s.logger.Info("Task status updated successfully",
			zap.String("taskID", string(taskID)),
			zap.String("newStatus", string(status)))
		return nil
	}

	// Mock implementation when repository is nil
	s.logger.Info("Task status updated successfully (mock)")
	return nil
}

// DeleteTask deletes a task
func (s *nudgeService) DeleteTask(taskID common.TaskID) error {
	s.logger.Info("Deleting task", zap.String("taskID", string(taskID)))

	if s.repository != nil {
		return s.repository.DeleteTask(taskID)
	}

	// Mock implementation when repository is nil
	s.logger.Info("Task deleted successfully (mock)")
	return nil
}

// GetTaskStats retrieves task statistics for a user
func (s *nudgeService) GetTaskStats(userID common.UserID) (*TaskStats, error) {
	s.logger.Info("Getting task stats", zap.String("userID", string(userID)))

	if s.repository != nil {
		return s.repository.GetTaskStats(userID)
	}

	// Mock implementation when repository is nil
	return &TaskStats{
		TotalTasks:     0,
		CompletedTasks: 0,
		OverdueTasks:   0,
		ActiveTasks:    0,
	}, nil
}

// ScheduleReminder schedules a reminder for a task
func (s *nudgeService) ScheduleReminder(taskID common.TaskID, scheduledAt time.Time, reminderType ReminderType) error {
	s.logger.Info("Scheduling reminder",
		zap.String("taskID", string(taskID)),
		zap.Time("scheduledAt", scheduledAt),
		zap.String("reminderType", string(reminderType)))

	if s.repository != nil {
		// Get the task to get the user ID and chat ID
		task, err := s.repository.GetTaskByID(taskID)
		if err != nil {
			s.logger.Error("Failed to get task for reminder scheduling", zap.Error(err))
			return err
		}

		// Determine ChatID - use task's ChatID if available, otherwise use UserID as fallback
		//
		// ChatID Resolution Strategy:
		// 1. Primary: Use ChatID stored in the task (for tasks created from chat interactions)
		// 2. Fallback: Use UserID as ChatID (for backwards compatibility with existing tasks)
		//
		// This approach ensures:
		// - New tasks created from chat interactions have proper ChatID tracking
		// - Existing tasks without ChatID continue to work (assuming UserID == ChatID for private chats)
		// - Clear logging when fallback is used for debugging and migration purposes
		chatID := task.ChatID
		if chatID == "" {
			// Fallback: assume ChatID equals UserID for backwards compatibility
			// This handles cases where tasks were created before ChatID tracking
			chatID = common.ChatID(task.UserID)
			s.logger.Warn("No ChatID found in task, using UserID as fallback",
				zap.String("taskID", string(taskID)),
				zap.String("userID", string(task.UserID)),
				zap.String("fallback_chatID", string(chatID)))
		}

		reminder := &Reminder{
			ID:           common.ID(common.NewID()),
			TaskID:       taskID,
			UserID:       task.UserID,
			ChatID:       chatID, // Use the resolved ChatID
			ScheduledAt:  scheduledAt,
			ReminderType: reminderType,
		}

		if err := s.repository.CreateReminder(reminder); err != nil {
			s.logger.Error("Failed to create reminder", zap.Error(err))
			return NewReminderSchedulingError(taskID, "failed to create reminder", err)
		}

		s.logger.Info("Reminder scheduled successfully", zap.String("reminderID", string(reminder.ID)))
		return nil
	}

	// Mock implementation when repository is nil
	s.logger.Info("Reminder scheduled successfully (mock)")
	return nil
}

// GetNudgeSettings retrieves nudge settings for a user
func (s *nudgeService) GetNudgeSettings(userID common.UserID) (*NudgeSettings, error) {
	s.logger.Info("Getting nudge settings", zap.String("userID", string(userID)))

	if s.repository != nil {
		return s.repository.GetNudgeSettingsByUserID(userID)
	}

	// Mock implementation when repository is nil
	return &NudgeSettings{
		UserID:        userID,
		NudgeInterval: time.Hour,
		MaxNudges:     3,
		Enabled:       true,
	}, nil
}

// UpdateNudgeSettings updates nudge settings for a user
func (s *nudgeService) UpdateNudgeSettings(settings *NudgeSettings) error {
	s.logger.Info("Updating nudge settings", zap.String("userID", string(settings.UserID)))

	// Validate settings using business logic
	if err := ValidateNudgeSettings(settings); err != nil {
		s.logger.Error("Nudge settings validation failed", zap.Error(err))
		return err
	}

	if s.repository != nil {
		settings.UpdatedAt = time.Now()
		if err := s.repository.CreateOrUpdateNudgeSettings(settings); err != nil {
			s.logger.Error("Failed to update nudge settings", zap.Error(err))
			return err
		}

		s.logger.Info("Nudge settings updated successfully", zap.String("userID", string(settings.UserID)))
		return nil
	}

	// Mock implementation when repository is nil
	s.logger.Info("Nudge settings updated successfully (mock)")
	return nil
}

// handleTaskParsed handles TaskParsed events from the LLM service
func (s *nudgeService) handleTaskParsed(event events.TaskParsed) {
	s.logger.Info("Handling TaskParsed event",
		zap.String("correlationID", event.CorrelationID),
		zap.String("userID", event.UserID),
		zap.String("chatID", event.ChatID),
		zap.String("taskTitle", event.ParsedTask.Title))

	// Create a task from the parsed event
	task := &Task{
		ID:          common.TaskID(common.NewID()),
		UserID:      common.UserID(event.UserID),
		ChatID:      common.ChatID(event.ChatID), // Store ChatID from the event
		Title:       event.ParsedTask.Title,
		Description: event.ParsedTask.Description,
		DueDate:     event.ParsedTask.DueDate,
		Priority:    common.Priority(event.ParsedTask.Priority),
		Status:      common.TaskStatusActive,
	}

	err := s.CreateTask(task)
	if err != nil {
		s.logger.Error("Failed to create task from parsed event", zap.Error(err))
		return
	}

	s.logger.Info("Task created successfully from parsed event", zap.String("taskID", string(task.ID)))
}

// handleTaskListRequested handles TaskListRequested events from the chatbot
func (s *nudgeService) handleTaskListRequested(event events.TaskListRequested) {
	s.logger.Info("Handling TaskListRequested event",
		zap.String("correlationID", event.CorrelationID),
		zap.String("userID", event.UserID),
		zap.String("chatID", event.ChatID))

	// Validate the request
	if err := s.validateTaskListRequest(event); err != nil {
		s.logger.Error("Task list request validation failed",
			zap.String("userID", event.UserID),
			zap.String("chatID", event.ChatID),
			zap.Error(err))
		s.publishTaskListErrorResponse(event, err)
		return
	}

	// Get tasks for the user
	filter := TaskFilter{
		UserID: common.UserID(event.UserID),
		Status: &[]common.TaskStatus{common.TaskStatusActive}[0], // Only active tasks
	}

	tasks, err := s.GetTasks(common.UserID(event.UserID), filter)
	if err != nil {
		s.logger.Error("Failed to get tasks for list request",
			zap.String("userID", event.UserID),
			zap.Error(err))

		// Create appropriate error based on the underlying cause
		var taskListErr TaskListError
		if err == ErrTaskNotFound {
			taskListErr = NewTaskListError(common.UserID(event.UserID), "No tasks found for user", err)
		} else {
			taskListErr = NewTaskListError(common.UserID(event.UserID), "Failed to retrieve tasks from database", err)
		}

		s.publishTaskListErrorResponse(event, taskListErr)
		return
	}

	// Convert tasks to TaskSummary format
	taskSummaries := make([]events.TaskSummary, len(tasks))
	for i, task := range tasks {
		taskSummaries[i] = events.TaskSummary{
			ID:          string(task.ID),
			Title:       task.Title,
			Description: task.Description,
			DueDate:     task.DueDate,
			Priority:    string(task.Priority),
			Status:      string(task.Status),
			IsOverdue:   task.IsOverdue(),
		}
	}

	// Publish successful TaskListResponse event
	response := events.TaskListResponse{
		Event:      events.NewEvent(),
		UserID:     event.UserID,
		ChatID:     event.ChatID,
		Tasks:      taskSummaries,
		TotalCount: len(taskSummaries),
		HasMore:    false, // Simple implementation - no pagination for now
		Success:    true,
		ErrorCode:  "",
		ErrorMsg:   "",
	}

	err = s.eventBus.Publish(events.TopicTaskListResponse, response)
	if err != nil {
		s.logger.Error("Failed to publish TaskListResponse event",
			zap.String("userID", event.UserID),
			zap.Error(err))
		return
	}

	s.logger.Info("TaskListResponse published successfully",
		zap.String("userID", event.UserID),
		zap.Int("taskCount", len(taskSummaries)))
}

// handleTaskActionRequested handles TaskActionRequested events from the chatbot
func (s *nudgeService) handleTaskActionRequested(event events.TaskActionRequested) {
	s.logger.Info("Handling TaskActionRequested event",
		zap.String("correlationID", event.CorrelationID),
		zap.String("userID", event.UserID),
		zap.String("chatID", event.ChatID),
		zap.String("taskID", event.TaskID),
		zap.String("action", event.Action))

	var err error
	var message string
	success := true

	// Validate the event structure first
	if err = s.validateTaskActionRequest(event); err != nil {
		s.logger.Error("Task action request validation failed",
			zap.String("taskID", event.TaskID),
			zap.String("userID", event.UserID),
			zap.String("action", event.Action),
			zap.Error(err))
		message = "Invalid request: " + err.Error()
		success = false
		s.publishTaskActionResponse(event, success, message)
		return
	}

	// Process the requested action
	switch event.Action {
	case "done", "complete":
		err = s.UpdateTaskStatus(common.TaskID(event.TaskID), common.TaskStatusCompleted)
		if err == nil {
			message = "Task marked as completed successfully!"
		} else {
			message = "Failed to mark task as completed: " + err.Error()
			success = false
		}

	case "delete":
		err = s.DeleteTask(common.TaskID(event.TaskID))
		if err == nil {
			message = "Task deleted successfully!"
		} else {
			message = "Failed to delete task: " + err.Error()
			success = false
		}

	case "snooze":
		// Snooze for 1 hour by default
		snoozeUntil := time.Now().Add(time.Hour)
		err = s.SnoozeTask(common.TaskID(event.TaskID), snoozeUntil)
		if err == nil {
			message = "Task snoozed for 1 hour!"
		} else {
			message = "Failed to snooze task: " + err.Error()
			success = false
		}

	default:
		err = NewInvalidTaskActionError(event.Action)
		message = "Invalid action: " + event.Action
		success = false
	}

	if err != nil {
		s.logger.Error("Failed to process task action",
			zap.String("action", event.Action),
			zap.String("taskID", event.TaskID),
			zap.Error(err))
	}

	s.publishTaskActionResponse(event, success, message)
}

// Additional service methods

// SnoozeTask snoozes a task until a specific time
func (s *nudgeService) SnoozeTask(taskID common.TaskID, snoozeUntil time.Time) error {
	s.logger.Info("Snoozing task",
		zap.String("taskID", string(taskID)),
		zap.Time("snoozeUntil", snoozeUntil))

	if s.repository != nil {
		task, err := s.repository.GetTaskByID(taskID)
		if err != nil {
			return err
		}

		// Use status manager for snoozing
		if err := s.statusManager.SnoozeTask(task, snoozeUntil); err != nil {
			return err
		}

		// Update task in repository
		if err := s.repository.UpdateTask(task); err != nil {
			return err
		}

		// Cancel existing reminders and schedule new ones
		go s.cancelTaskReminders(taskID)
		go s.scheduleInitialReminder(task)

		s.logger.Info("Task snoozed successfully", zap.String("taskID", string(taskID)))
		return nil
	}

	// Mock implementation
	s.logger.Info("Task snoozed successfully (mock)")
	return nil
}

// GetOverdueTasks retrieves overdue tasks for a user
func (s *nudgeService) GetOverdueTasks(userID common.UserID) ([]*Task, error) {
	s.logger.Info("Getting overdue tasks", zap.String("userID", string(userID)))

	if s.repository != nil {
		// Use specialized repository method if available
		if repo, ok := s.repository.(*gormNudgeRepository); ok {
			return repo.GetOverdueTasks(userID)
		}

		// Fallback to filtered query
		filter := TaskFilter{
			UserID: userID,
			Status: &[]common.TaskStatus{common.TaskStatusActive}[0],
		}
		tasks, err := s.repository.GetTasksByUserID(userID, filter)
		if err != nil {
			return nil, err
		}

		// Filter for overdue tasks
		var overdueTasks []*Task
		for _, task := range tasks {
			if task.IsOverdue() {
				overdueTasks = append(overdueTasks, task)
			}
		}

		return overdueTasks, nil
	}

	// Mock implementation
	return []*Task{}, nil
}

// BulkUpdateStatus updates multiple tasks' status
func (s *nudgeService) BulkUpdateStatus(taskIDs []common.TaskID, status common.TaskStatus) error {
	s.logger.Info("Bulk updating task status",
		zap.Int("count", len(taskIDs)),
		zap.String("status", string(status)))

	if s.repository != nil {
		// Use specialized repository method if available
		if repo, ok := s.repository.(*gormNudgeRepository); ok {
			return repo.BulkUpdateTaskStatus(taskIDs, status)
		}

		// Fallback to individual updates
		for _, taskID := range taskIDs {
			if err := s.UpdateTaskStatus(taskID, status); err != nil {
				s.logger.Error("Failed to update task status in bulk operation",
					zap.String("taskID", string(taskID)),
					zap.Error(err))
				return err
			}
		}

		return nil
	}

	// Mock implementation
	s.logger.Info("Bulk status update completed (mock)")
	return nil
}

// Helper methods

// scheduleInitialReminder schedules the initial reminder for a task
func (s *nudgeService) scheduleInitialReminder(task *Task) {
	if s.repository == nil {
		return
	}

	// Get user settings
	settings, err := s.repository.GetNudgeSettingsByUserID(task.UserID)
	if err != nil {
		s.logger.Error("Failed to get nudge settings for reminder scheduling", zap.Error(err))
		return
	}

	// Calculate reminder time
	reminderTime := s.reminderManager.CalculateReminderTime(task, settings)

	// Schedule the reminder
	err = s.ScheduleReminder(task.ID, reminderTime, ReminderTypeInitial)
	if err != nil {
		s.logger.Error("Failed to schedule initial reminder",
			zap.String("taskID", string(task.ID)),
			zap.Error(err))
	}
}

// cancelTaskReminders cancels all future reminders for a task
func (s *nudgeService) cancelTaskReminders(taskID common.TaskID) {
	if s.repository == nil {
		return
	}

	reminders, err := s.repository.GetRemindersByTaskID(taskID)
	if err != nil {
		s.logger.Error("Failed to get reminders for cancellation",
			zap.String("taskID", string(taskID)),
			zap.Error(err))
		return
	}

	for _, reminder := range reminders {
		if reminder.SentAt == nil { // Only cancel unsent reminders
			err := s.repository.DeleteReminder(reminder.ID)
			if err != nil {
				s.logger.Error("Failed to cancel reminder",
					zap.String("reminderID", string(reminder.ID)),
					zap.Error(err))
			}
		}
	}
}

// validateTaskActionRequest validates TaskActionRequested events
func (s *nudgeService) validateTaskActionRequest(event events.TaskActionRequested) error {
	// Validate required event fields
	if event.UserID == "" {
		return fmt.Errorf("userID is required")
	}
	if event.ChatID == "" {
		return fmt.Errorf("chatID is required")
	}
	if event.TaskID == "" {
		return fmt.Errorf("taskID is required")
	}
	if event.Action == "" {
		return fmt.Errorf("action is required")
	}

	// Validate TaskID format
	if !common.ID(event.TaskID).IsValid() {
		return fmt.Errorf("taskID must be a valid UUID: %s", event.TaskID)
	}

	// Validate UserID format
	if !common.ID(event.UserID).IsValid() {
		return fmt.Errorf("userID must be a valid UUID: %s", event.UserID)
	}

	// Validate action is allowed
	validActions := map[string]bool{
		"done":     true,
		"complete": true,
		"delete":   true,
		"snooze":   true,
	}
	if !validActions[event.Action] {
		return NewInvalidTaskActionError(event.Action)
	}

	// Skip repository validation if repository is nil (mock mode)
	if s.repository == nil {
		s.logger.Debug("Skipping task validation - repository is nil (mock mode)")
		return nil
	}

	// Validate task exists and belongs to user
	taskID := common.TaskID(event.TaskID)
	task, err := s.repository.GetTaskByID(taskID)
	if err != nil {
		if err == ErrTaskNotFound {
			return fmt.Errorf("task not found: %s", event.TaskID)
		}
		return fmt.Errorf("failed to retrieve task: %w", err)
	}

	// Validate task belongs to the requesting user
	if string(task.UserID) != event.UserID {
		return fmt.Errorf("task %s does not belong to user %s", event.TaskID, event.UserID)
	}

	// Validate the action is valid for the current task status
	if err := s.validateActionForTaskStatus(event.Action, task.Status); err != nil {
		return fmt.Errorf("action validation failed: %w", err)
	}

	return nil
}

// validateActionForTaskStatus validates if an action is valid for the current task status
func (s *nudgeService) validateActionForTaskStatus(action string, currentStatus common.TaskStatus) error {
	switch action {
	case "done", "complete":
		// Can only complete active or snoozed tasks
		if currentStatus != common.TaskStatusActive && currentStatus != common.TaskStatusSnoozed {
			return fmt.Errorf("cannot complete task with status %s", currentStatus)
		}
	case "delete":
		// Can delete tasks in any status except already deleted
		if currentStatus == common.TaskStatusDeleted {
			return fmt.Errorf("task is already deleted")
		}
	case "snooze":
		// Can only snooze active tasks
		if currentStatus != common.TaskStatusActive {
			return fmt.Errorf("can only snooze active tasks, current status is %s", currentStatus)
		}
	}
	return nil
}

// publishTaskActionResponse publishes a TaskActionResponse event
func (s *nudgeService) publishTaskActionResponse(event events.TaskActionRequested, success bool, message string) {
	response := events.TaskActionResponse{
		Event:   events.NewEvent(),
		UserID:  event.UserID,
		ChatID:  event.ChatID,
		TaskID:  event.TaskID,
		Action:  event.Action,
		Success: success,
		Message: message,
	}

	publishErr := s.eventBus.Publish(events.TopicTaskActionResponse, response)
	if publishErr != nil {
		s.logger.Error("Failed to publish TaskActionResponse event",
			zap.Error(publishErr),
			zap.String("taskID", event.TaskID),
			zap.String("action", event.Action))
		return
	}

	s.logger.Info("TaskActionResponse published successfully",
		zap.String("taskID", event.TaskID),
		zap.String("action", event.Action),
		zap.Bool("success", success))
}

// validateTaskListRequest validates TaskListRequested events
func (s *nudgeService) validateTaskListRequest(event events.TaskListRequested) error {
	// Validate required event fields
	if event.UserID == "" {
		return NewTaskListValidationError("", "userID is required")
	}
	if event.ChatID == "" {
		return NewTaskListValidationError(common.UserID(event.UserID), "chatID is required")
	}

	// Validate UserID format
	if !common.ID(event.UserID).IsValid() {
		return NewTaskListValidationError(common.UserID(event.UserID),
			fmt.Sprintf("userID must be a valid UUID: %s", event.UserID))
	}

	// Validate ChatID is not empty (format validation depends on provider)
	if len(event.ChatID) == 0 {
		return NewTaskListValidationError(common.UserID(event.UserID), "chatID cannot be empty")
	}

	return nil
}

// publishTaskListErrorResponse publishes a TaskListResponse event with error information
func (s *nudgeService) publishTaskListErrorResponse(event events.TaskListRequested, err error) {
	var errorCode, errorMsg string

	// Extract error details from different error types
	if taskListErr, ok := err.(TaskListError); ok {
		errorCode = taskListErr.Code()
		errorMsg = taskListErr.Message()
	} else if nudgeErr, ok := err.(NudgeError); ok {
		errorCode = nudgeErr.Code()
		errorMsg = nudgeErr.Message()
	} else {
		errorCode = ErrCodeTaskListFailed
		errorMsg = err.Error()
	}

	response := events.TaskListResponse{
		Event:      events.NewEvent(),
		UserID:     event.UserID,
		ChatID:     event.ChatID,
		Tasks:      []events.TaskSummary{}, // Empty task list on error
		TotalCount: 0,
		HasMore:    false,
		Success:    false,
		ErrorCode:  errorCode,
		ErrorMsg:   errorMsg,
	}

	publishErr := s.eventBus.Publish(events.TopicTaskListResponse, response)
	if publishErr != nil {
		s.logger.Error("Failed to publish TaskListResponse error event",
			zap.Error(publishErr),
			zap.String("userID", event.UserID),
			zap.String("originalError", errorMsg))
		return
	}

	s.logger.Info("TaskListResponse error published successfully",
		zap.String("userID", event.UserID),
		zap.String("errorCode", errorCode),
		zap.String("errorMessage", errorMsg))
}
