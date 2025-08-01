package nudge

import (
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
}

// nudgeService implements the NudgeService interface
type nudgeService struct {
	eventBus        events.EventBus
	logger          *zap.Logger
	repository      NudgeRepository
	validator       *TaskValidator
	reminderManager *ReminderManager
	statusManager   *TaskStatusManager
}

// NewNudgeService creates a new instance of NudgeService
func NewNudgeService(eventBus events.EventBus, logger *zap.Logger, repository NudgeRepository) NudgeService {
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
	}

	// Subscribe to relevant events
	service.setupEventSubscriptions()

	return service
}

// setupEventSubscriptions sets up event subscriptions for the nudge service
func (s *nudgeService) setupEventSubscriptions() {
	// Subscribe to TaskParsed events from the LLM service
	err := s.eventBus.Subscribe(events.TopicTaskParsed, s.handleTaskParsed)
	if err != nil {
		s.logger.Error("Failed to subscribe to TaskParsed events", zap.Error(err))
	}
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
		// Get the task to get the user ID
		task, err := s.repository.GetTaskByID(taskID)
		if err != nil {
			s.logger.Error("Failed to get task for reminder scheduling", zap.Error(err))
			return err
		}

		reminder := &Reminder{
			ID:           common.ID(common.NewID()),
			TaskID:       taskID,
			UserID:       task.UserID,
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
		zap.String("taskTitle", event.ParsedTask.Title))

	// Create a task from the parsed event
	task := &Task{
		ID:          common.TaskID(common.NewID()),
		UserID:      common.UserID(event.UserID),
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
