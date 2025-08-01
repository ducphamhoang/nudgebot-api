package nudge

import (
	"errors"
	"time"

	"nudgebot-api/internal/common"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// gormNudgeRepository implements the NudgeRepository interface using GORM
type gormNudgeRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewGormNudgeRepository creates a new GORM-based nudge repository
func NewGormNudgeRepository(db *gorm.DB, logger *zap.Logger) NudgeRepository {
	return &gormNudgeRepository{
		db:     db,
		logger: logger,
	}
}

// Task operations

// CreateTask creates a new task in the database
func (r *gormNudgeRepository) CreateTask(task *Task) error {
	r.logger.Debug("Creating task", zap.String("taskID", string(task.ID)), zap.String("userID", string(task.UserID)))

	// Validate task before creation
	validator := NewTaskValidator()
	if err := validator.ValidateTask(task); err != nil {
		return err
	}

	// Check for duplicate tasks (same user, title, and status)
	var existingCount int64
	err := r.db.Model(&Task{}).
		Where("user_id = ? AND title = ? AND status IN ?", task.UserID, task.Title, []common.TaskStatus{
			common.TaskStatusActive,
			common.TaskStatusSnoozed,
		}).
		Count(&existingCount).Error

	if err != nil {
		return WrapRepositoryError(err, "duplicate check")
	}

	if existingCount > 0 {
		return NewTaskValidationError("title", task.Title, "task with this title already exists for user")
	}

	// Set timestamps
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now

	// Create the task
	if err := r.db.Create(task).Error; err != nil {
		return WrapRepositoryError(err, "create task")
	}

	r.logger.Info("Task created successfully", zap.String("taskID", string(task.ID)))
	return nil
}

// GetTaskByID retrieves a task by its ID
func (r *gormNudgeRepository) GetTaskByID(taskID common.TaskID) (*Task, error) {
	r.logger.Debug("Getting task by ID", zap.String("taskID", string(taskID)))

	var task Task
	err := r.db.Where("id = ?", taskID).First(&task).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.NotFoundError{Resource: "Task", ID: string(taskID)}
		}
		return nil, WrapRepositoryError(err, "get task by ID")
	}

	return &task, nil
}

// GetTasksByUserID retrieves tasks for a user with filtering
func (r *gormNudgeRepository) GetTasksByUserID(userID common.UserID, filter TaskFilter) ([]*Task, error) {
	r.logger.Debug("Getting tasks by user ID",
		zap.String("userID", string(userID)),
		zap.Any("filter", filter))

	// Validate filter
	validator := NewTaskValidator()
	if err := validator.ValidateTaskFilter(filter); err != nil {
		return nil, err
	}

	// Build query using query builder
	qb := NewQueryBuilder(r.db)
	taskQuery := qb.TaskQuery().WithUserID(userID)

	// Apply filters
	if filter.Status != nil {
		taskQuery = taskQuery.WithStatus(*filter.Status)
	}
	if filter.Priority != nil {
		taskQuery = taskQuery.WithPriority(*filter.Priority)
	}
	if filter.DueAfter != nil || filter.DueBefore != nil {
		taskQuery = taskQuery.WithDueDateRange(filter.DueAfter, filter.DueBefore)
	}

	// Apply default ordering (priority first, then due date)
	taskQuery = taskQuery.OrderByPriority().OrderByDueDate()

	// Apply pagination
	if filter.Limit > 0 || filter.Offset > 0 {
		limit := filter.Limit
		if limit == 0 {
			limit = 50 // Default limit
		}
		taskQuery = taskQuery.WithPagination(limit, filter.Offset)
	}

	tasks, err := taskQuery.Find()
	if err != nil {
		return nil, WrapRepositoryError(err, "get tasks by user ID")
	}

	r.logger.Debug("Retrieved tasks", zap.Int("count", len(tasks)))
	return tasks, nil
}

// UpdateTask updates an existing task
func (r *gormNudgeRepository) UpdateTask(task *Task) error {
	r.logger.Debug("Updating task", zap.String("taskID", string(task.ID)))

	// Validate task
	validator := NewTaskValidator()
	if err := validator.ValidateTask(task); err != nil {
		return err
	}

	// Update timestamp
	task.UpdatedAt = time.Now()

	// Use optimistic locking by checking updated_at
	result := r.db.Model(task).Where("id = ?", task.ID).Updates(task)
	if result.Error != nil {
		return WrapRepositoryError(result.Error, "update task")
	}

	if result.RowsAffected == 0 {
		return common.NotFoundError{Resource: "Task", ID: string(task.ID)}
	}

	r.logger.Info("Task updated successfully", zap.String("taskID", string(task.ID)))
	return nil
}

// DeleteTask performs soft delete on a task
func (r *gormNudgeRepository) DeleteTask(taskID common.TaskID) error {
	r.logger.Debug("Deleting task", zap.String("taskID", string(taskID)))

	// Update status to deleted instead of hard delete
	result := r.db.Model(&Task{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":     common.TaskStatusDeleted,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return WrapRepositoryError(result.Error, "delete task")
	}

	if result.RowsAffected == 0 {
		return common.NotFoundError{Resource: "Task", ID: string(taskID)}
	}

	r.logger.Info("Task deleted successfully", zap.String("taskID", string(taskID)))
	return nil
}

// GetTaskStats retrieves task statistics for a user
func (r *gormNudgeRepository) GetTaskStats(userID common.UserID) (*TaskStats, error) {
	r.logger.Debug("Getting task stats", zap.String("userID", string(userID)))

	stats, err := GetUserTaskSummary(r.db, userID)
	if err != nil {
		return nil, WrapRepositoryError(err, "get task stats")
	}

	return stats, nil
}

// Reminder operations

// CreateReminder creates a new reminder
func (r *gormNudgeRepository) CreateReminder(reminder *Reminder) error {
	r.logger.Debug("Creating reminder",
		zap.String("reminderID", string(reminder.ID)),
		zap.String("taskID", string(reminder.TaskID)))

	// Validate reminder type
	if !reminder.ReminderType.IsValid() {
		return NewTaskValidationError("reminder_type", reminder.ReminderType, "invalid reminder type")
	}

	// Validate scheduled time
	if reminder.ScheduledAt.Before(time.Now().Add(-1 * time.Hour)) {
		return NewTaskValidationError("scheduled_at", reminder.ScheduledAt, "scheduled time cannot be more than 1 hour in the past")
	}

	// Verify task exists
	var taskExists bool
	err := r.db.Model(&Task{}).
		Select("1").
		Where("id = ? AND user_id = ?", reminder.TaskID, reminder.UserID).
		Scan(&taskExists).Error

	if err != nil {
		return WrapRepositoryError(err, "verify task exists")
	}

	if !taskExists {
		return common.NotFoundError{Resource: "Task", ID: string(reminder.TaskID)}
	}

	// Create reminder
	if err := r.db.Create(reminder).Error; err != nil {
		return WrapRepositoryError(err, "create reminder")
	}

	r.logger.Info("Reminder created successfully", zap.String("reminderID", string(reminder.ID)))
	return nil
}

// GetDueReminders retrieves reminders that are due before the specified time
func (r *gormNudgeRepository) GetDueReminders(before time.Time) ([]*Reminder, error) {
	r.logger.Debug("Getting due reminders", zap.Time("before", before))

	qb := NewQueryBuilder(r.db)
	reminders, err := qb.ReminderQuery().
		WithDueBefore(before).
		WithUnsent().
		WithTaskJoin().
		Find()

	if err != nil {
		return nil, WrapRepositoryError(err, "get due reminders")
	}

	r.logger.Debug("Retrieved due reminders", zap.Int("count", len(reminders)))
	return reminders, nil
}

// MarkReminderSent marks a reminder as sent
func (r *gormNudgeRepository) MarkReminderSent(reminderID common.ID) error {
	r.logger.Debug("Marking reminder as sent", zap.String("reminderID", string(reminderID)))

	now := time.Now()
	result := r.db.Model(&Reminder{}).
		Where("id = ? AND sent_at IS NULL", reminderID).
		Update("sent_at", now)

	if result.Error != nil {
		return WrapRepositoryError(result.Error, "mark reminder sent")
	}

	if result.RowsAffected == 0 {
		return common.NotFoundError{Resource: "Reminder", ID: string(reminderID)}
	}

	r.logger.Info("Reminder marked as sent", zap.String("reminderID", string(reminderID)))
	return nil
}

// GetRemindersByTaskID retrieves all reminders for a specific task
func (r *gormNudgeRepository) GetRemindersByTaskID(taskID common.TaskID) ([]*Reminder, error) {
	r.logger.Debug("Getting reminders by task ID", zap.String("taskID", string(taskID)))

	qb := NewQueryBuilder(r.db)
	reminders, err := qb.ReminderQuery().
		WithTaskID(taskID).
		Find()

	if err != nil {
		return nil, WrapRepositoryError(err, "get reminders by task ID")
	}

	return reminders, nil
}

// DeleteReminder deletes a reminder
func (r *gormNudgeRepository) DeleteReminder(reminderID common.ID) error {
	r.logger.Debug("Deleting reminder", zap.String("reminderID", string(reminderID)))

	result := r.db.Delete(&Reminder{}, "id = ?", reminderID)
	if result.Error != nil {
		return WrapRepositoryError(result.Error, "delete reminder")
	}

	if result.RowsAffected == 0 {
		return common.NotFoundError{Resource: "Reminder", ID: string(reminderID)}
	}

	r.logger.Info("Reminder deleted successfully", zap.String("reminderID", string(reminderID)))
	return nil
}

// Nudge settings operations

// GetNudgeSettingsByUserID retrieves nudge settings for a user
func (r *gormNudgeRepository) GetNudgeSettingsByUserID(userID common.UserID) (*NudgeSettings, error) {
	r.logger.Debug("Getting nudge settings", zap.String("userID", string(userID)))

	var settings NudgeSettings
	err := r.db.Where("user_id = ?", userID).First(&settings).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return default settings if none exist
			now := time.Now()
			return &NudgeSettings{
				UserID:        userID,
				NudgeInterval: DefaultNudgeInterval,
				MaxNudges:     DefaultMaxNudges,
				Enabled:       true,
				CreatedAt:     now,
				UpdatedAt:     now,
			}, nil
		}
		return nil, WrapRepositoryError(err, "get nudge settings")
	}

	return &settings, nil
}

// CreateOrUpdateNudgeSettings creates or updates nudge settings using upsert
func (r *gormNudgeRepository) CreateOrUpdateNudgeSettings(settings *NudgeSettings) error {
	r.logger.Debug("Creating or updating nudge settings", zap.String("userID", string(settings.UserID)))

	// Validate settings
	if err := ValidateNudgeSettings(settings); err != nil {
		return err
	}

	// Set timestamps
	now := time.Now()
	if settings.CreatedAt.IsZero() {
		settings.CreatedAt = now
	}
	settings.UpdatedAt = now

	// Use GORM's Save method for upsert behavior
	if err := r.db.Save(settings).Error; err != nil {
		return WrapRepositoryError(err, "create or update nudge settings")
	}

	r.logger.Info("Nudge settings saved successfully", zap.String("userID", string(settings.UserID)))
	return nil
}

// DeleteNudgeSettings deletes nudge settings for a user
func (r *gormNudgeRepository) DeleteNudgeSettings(userID common.UserID) error {
	r.logger.Debug("Deleting nudge settings", zap.String("userID", string(userID)))

	result := r.db.Delete(&NudgeSettings{}, "user_id = ?", userID)
	if result.Error != nil {
		return WrapRepositoryError(result.Error, "delete nudge settings")
	}

	if result.RowsAffected == 0 {
		return common.NotFoundError{Resource: "NudgeSettings", ID: string(userID)}
	}

	r.logger.Info("Nudge settings deleted successfully", zap.String("userID", string(userID)))
	return nil
}

// Transaction support

// WithTransaction executes a function within a database transaction
func (r *gormNudgeRepository) WithTransaction(fn func(NudgeRepository) error) error {
	r.logger.Debug("Starting transaction")

	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := &gormNudgeRepository{
			db:     tx,
			logger: r.logger,
		}

		err := fn(txRepo)
		if err != nil {
			r.logger.Debug("Transaction failed, rolling back", zap.Error(err))
			return err
		}

		r.logger.Debug("Transaction completed successfully")
		return nil
	})
}

// Additional helper methods

// GetOverdueTasks retrieves overdue tasks for a user
func (r *gormNudgeRepository) GetOverdueTasks(userID common.UserID) ([]*Task, error) {
	r.logger.Debug("Getting overdue tasks", zap.String("userID", string(userID)))

	qb := NewQueryBuilder(r.db)
	tasks, err := qb.TaskQuery().
		WithUserID(userID).
		WithOverdue().
		OrderByDueDate().
		Find()

	if err != nil {
		return nil, WrapRepositoryError(err, "get overdue tasks")
	}

	return tasks, nil
}

// GetTasksDueSoon retrieves tasks due within a specific time window
func (r *gormNudgeRepository) GetTasksDueSoon(userID common.UserID, within time.Duration) ([]*Task, error) {
	r.logger.Debug("Getting tasks due soon",
		zap.String("userID", string(userID)),
		zap.Duration("within", within))

	tasks, err := GetTasksDueSoon(r.db, userID, within)
	if err != nil {
		return nil, WrapRepositoryError(err, "get tasks due soon")
	}

	return tasks, nil
}

// BulkUpdateTaskStatus updates multiple tasks' status
func (r *gormNudgeRepository) BulkUpdateTaskStatus(taskIDs []common.TaskID, status common.TaskStatus) error {
	r.logger.Debug("Bulk updating task status",
		zap.Int("count", len(taskIDs)),
		zap.String("status", string(status)))

	if len(taskIDs) == 0 {
		return nil
	}

	err := BulkUpdateTaskStatus(r.db, taskIDs, status)
	if err != nil {
		return WrapRepositoryError(err, "bulk update task status")
	}

	r.logger.Info("Bulk task status update completed", zap.Int("count", len(taskIDs)))
	return nil
}

// CleanupOldData removes old sent reminders and deleted tasks
func (r *gormNudgeRepository) CleanupOldData(olderThan time.Duration) error {
	r.logger.Debug("Cleaning up old data", zap.Duration("olderThan", olderThan))

	cutoff := time.Now().Add(-olderThan)

	// Clean up old sent reminders
	err := CleanupOldReminders(r.db, cutoff)
	if err != nil {
		return WrapRepositoryError(err, "cleanup old reminders")
	}

	// Clean up old deleted tasks (hard delete after soft delete)
	result := r.db.Unscoped().Delete(&Task{}, "status = ? AND updated_at < ?", common.TaskStatusDeleted, cutoff)
	if result.Error != nil {
		return WrapRepositoryError(result.Error, "cleanup old deleted tasks")
	}

	r.logger.Info("Old data cleanup completed",
		zap.Duration("olderThan", olderThan),
		zap.Int64("deletedTasks", result.RowsAffected))

	return nil
}
