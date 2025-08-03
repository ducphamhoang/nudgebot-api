package scheduler

import (
	"time"

	"nudgebot-api/internal/common"
	"nudgebot-api/internal/events"
	"nudgebot-api/internal/nudge"

	"go.uber.org/zap"
)

// reminderWorker handles the processing of due reminders
type reminderWorker struct {
	scheduler *scheduler
	workerID  int
	logger    *zap.Logger
}

// processReminders fetches and processes all due reminders
func (w *reminderWorker) processReminders() error {
	startTime := time.Now()
	w.logger.Debug("Starting reminder processing cycle")

	// Fetch due reminders
	reminders, err := w.scheduler.repository.GetDueReminders(time.Now())
	if err != nil {
		return WrapWorkerError(err, w.workerID, "fetch_due_reminders")
	}

	if len(reminders) == 0 {
		w.logger.Debug("No due reminders found")
		return nil
	}

	w.logger.Info("Processing due reminders", zap.Int("reminder_count", len(reminders)))

	processedCount := 0
	nudgesCreated := 0
	errorCount := 0

	// Process each reminder
	for _, reminder := range reminders {
		if err := w.processReminder(reminder); err != nil {
			w.logger.Error("Failed to process reminder",
				zap.String("reminder_id", string(reminder.ID)),
				zap.String("task_id", string(reminder.TaskID)),
				zap.Error(err))
			w.scheduler.metrics.RecordProcessingError(err)
			errorCount++
			continue
		}

		processedCount++

		// Check if we should create a nudge
		if w.shouldCreateNudge(reminder) {
			if err := w.createNudgeReminder(reminder); err != nil {
				w.logger.Error("Failed to create nudge reminder",
					zap.String("reminder_id", string(reminder.ID)),
					zap.String("task_id", string(reminder.TaskID)),
					zap.Error(err))
				w.scheduler.metrics.RecordProcessingError(err)
				errorCount++
			} else {
				nudgesCreated++
			}
		}
	}

	processingDuration := time.Since(startTime)

	// Record metrics
	w.scheduler.metrics.RecordReminderProcessed(processingDuration)
	for i := 0; i < nudgesCreated; i++ {
		w.scheduler.metrics.RecordNudgeCreated()
	}

	w.logger.Info("Reminder processing cycle completed",
		zap.Int("total_reminders", len(reminders)),
		zap.Int("processed_count", processedCount),
		zap.Int("nudges_created", nudgesCreated),
		zap.Int("error_count", errorCount),
		zap.Duration("processing_duration", processingDuration))

	return nil
}

// processReminder handles a single reminder
func (w *reminderWorker) processReminder(reminder *nudge.Reminder) error {
	// Publish ReminderDue event with proper ChatID from reminder data
	//
	// ChatID Resolution:
	// The ChatID is now properly stored in the reminder data structure, eliminating
	// the previous assumption that ChatID equals UserID. This ensures:
	// 1. Accurate delivery of reminders to the correct chat/conversation
	// 2. Support for group chats where ChatID != UserID
	// 3. Proper tracking of user-chat relationships
	//
	// The ChatID is populated when:
	// - Tasks are created from chat interactions (via TaskParsed events)
	// - Reminders are scheduled with the task's ChatID information
	// - Nudge reminders inherit ChatID from original reminders
	reminderDueEvent := events.ReminderDue{
		Event:  events.NewEvent(),
		TaskID: string(reminder.TaskID),
		UserID: string(reminder.UserID),
		ChatID: string(reminder.ChatID), // Use the actual ChatID from reminder data
	}

	if err := w.scheduler.eventBus.Publish(events.TopicReminderDue, reminderDueEvent); err != nil {
		return NewReminderProcessingError(string(reminder.ID), "publish_event", err)
	}

	// Mark reminder as sent
	if err := w.scheduler.repository.MarkReminderSent(reminder.ID); err != nil {
		return NewReminderProcessingError(string(reminder.ID), "mark_sent", err)
	}

	w.logger.Debug("Reminder processed successfully",
		zap.String("reminder_id", string(reminder.ID)),
		zap.String("task_id", string(reminder.TaskID)),
		zap.String("reminder_type", string(reminder.ReminderType)))

	return nil
}

// shouldCreateNudge determines if a follow-up nudge should be created
func (w *reminderWorker) shouldCreateNudge(reminder *nudge.Reminder) bool {
	// Only create nudges for initial reminders
	if reminder.ReminderType != nudge.ReminderTypeInitial {
		return false
	}

	// Get the task to check its status
	task, err := w.scheduler.repository.GetTaskByID(reminder.TaskID)
	if err != nil {
		w.logger.Error("Failed to get task for nudge evaluation",
			zap.String("task_id", string(reminder.TaskID)),
			zap.Error(err))
		return false
	}

	// Don't nudge if task is not active
	if task.Status != common.TaskStatusActive {
		w.logger.Debug("Task is not active, skipping nudge creation",
			zap.String("task_id", string(reminder.TaskID)),
			zap.String("status", string(task.Status)))
		return false
	}

	// Get existing reminders for this task to count nudges
	existingReminders, err := w.scheduler.repository.GetRemindersByTaskID(reminder.TaskID)
	if err != nil {
		w.logger.Error("Failed to get existing reminders for nudge evaluation",
			zap.String("task_id", string(reminder.TaskID)),
			zap.Error(err))
		return false
	}

	// Count existing nudges
	nudgeCount := 0
	for _, r := range existingReminders {
		if r.ReminderType == nudge.ReminderTypeNudge {
			nudgeCount++
		}
	}

	// Get user's nudge settings (use defaults if not found)
	nudgeSettings, err := w.scheduler.repository.GetNudgeSettingsByUserID(reminder.UserID)
	if err != nil {
		// Use default settings if user settings not found
		nudgeSettings = &nudge.NudgeSettings{
			UserID:        reminder.UserID,
			NudgeInterval: time.Duration(w.scheduler.config.NudgeDelay) * time.Second,
			MaxNudges:     3, // Default max nudges
			Enabled:       true,
		}
	}

	// Use business logic to determine if nudge should be created
	reminderManager := nudge.NewReminderManager()
	shouldNudge := reminderManager.ShouldCreateNudge(task, nudgeCount, nudgeSettings)

	w.logger.Debug("Nudge evaluation completed",
		zap.String("task_id", string(reminder.TaskID)),
		zap.Int("existing_nudges", nudgeCount),
		zap.Int("max_nudges", nudgeSettings.MaxNudges),
		zap.Bool("should_create_nudge", shouldNudge))

	return shouldNudge
}

// createNudgeReminder creates a follow-up nudge reminder
func (w *reminderWorker) createNudgeReminder(originalReminder *nudge.Reminder) error {
	// Get nudge settings for the user
	nudgeSettings, err := w.scheduler.repository.GetNudgeSettingsByUserID(originalReminder.UserID)
	if err != nil {
		// Use default settings if not found
		nudgeSettings = &nudge.NudgeSettings{
			UserID:        originalReminder.UserID,
			NudgeInterval: time.Duration(w.scheduler.config.NudgeDelay) * time.Second,
			MaxNudges:     3, // Default max nudges
			Enabled:       true,
		}
	}

	// Use business logic to calculate next nudge time with exponential backoff
	reminderManager := nudge.NewReminderManager()
	nudgeTime := reminderManager.GetNextNudgeTime(originalReminder.ScheduledAt, nudgeSettings)

	// Create new nudge reminder with preserved ChatID
	// This ensures nudge reminders maintain the same chat context as the original reminder
	nudgeReminder := &nudge.Reminder{
		ID:           common.NewID(),
		TaskID:       originalReminder.TaskID,
		UserID:       originalReminder.UserID,
		ChatID:       originalReminder.ChatID, // Preserve ChatID from original reminder
		ScheduledAt:  nudgeTime,
		SentAt:       nil,
		ReminderType: nudge.ReminderTypeNudge,
	}

	// Save the nudge reminder
	if err := w.scheduler.repository.CreateReminder(nudgeReminder); err != nil {
		return NewNudgeCreationError(string(originalReminder.TaskID), "create_reminder", err)
	}

	w.logger.Info("Nudge reminder created successfully",
		zap.String("original_reminder_id", string(originalReminder.ID)),
		zap.String("nudge_reminder_id", string(nudgeReminder.ID)),
		zap.String("task_id", string(originalReminder.TaskID)),
		zap.Time("original_scheduled_at", originalReminder.ScheduledAt),
		zap.Time("nudge_scheduled_at", nudgeTime),
		zap.Duration("nudge_interval", nudgeSettings.NudgeInterval))

	return nil
}
