package nudge

import (
	"fmt"

	"nudgebot-api/internal/user"

	"gorm.io/gorm"
)

// RunMigrations performs auto-migration for all nudge-related tables
func RunMigrations(db *gorm.DB) error {
	// Auto-migrate all nudge models
	err := db.AutoMigrate(
		&user.User{},
		&Task{},
		&Reminder{},
		&NudgeSettings{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto-migrate nudge tables: %w", err)
	}

	// Create database indexes for performance
	if err := createIndexes(db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// createIndexes creates performance indexes for nudge tables
func createIndexes(db *gorm.DB) error {
	// Task table indexes
	taskIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_chat_id ON tasks(chat_id)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks(due_date)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_updated_at ON tasks(updated_at)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_user_status ON tasks(user_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_user_priority ON tasks(user_id, priority)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_status_due_date ON tasks(status, due_date)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_user_chat ON tasks(user_id, chat_id)",
	}

	for _, index := range taskIndexes {
		if err := db.Exec(index).Error; err != nil {
			return fmt.Errorf("failed to create task index: %w", err)
		}
	}

	// Reminder table indexes
	reminderIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_reminders_task_id ON reminders(task_id)",
		"CREATE INDEX IF NOT EXISTS idx_reminders_user_id ON reminders(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_reminders_chat_id ON reminders(chat_id)",
		"CREATE INDEX IF NOT EXISTS idx_reminders_scheduled_at ON reminders(scheduled_at)",
		"CREATE INDEX IF NOT EXISTS idx_reminders_sent_at ON reminders(sent_at)",
		"CREATE INDEX IF NOT EXISTS idx_reminders_type ON reminders(reminder_type)",
		"CREATE INDEX IF NOT EXISTS idx_reminders_scheduled_sent ON reminders(scheduled_at, sent_at)",
		"CREATE INDEX IF NOT EXISTS idx_reminders_task_scheduled ON reminders(task_id, scheduled_at)",
		"CREATE INDEX IF NOT EXISTS idx_reminders_user_chat ON reminders(user_id, chat_id)",
	}

	for _, index := range reminderIndexes {
		if err := db.Exec(index).Error; err != nil {
			return fmt.Errorf("failed to create reminder index: %w", err)
		}
	}

	// Nudge settings table indexes
	settingsIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_nudge_settings_enabled ON nudge_settings(enabled)",
		"CREATE INDEX IF NOT EXISTS idx_nudge_settings_created_at ON nudge_settings(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_nudge_settings_updated_at ON nudge_settings(updated_at)",
	}

	for _, index := range settingsIndexes {
		if err := db.Exec(index).Error; err != nil {
			return fmt.Errorf("failed to create nudge settings index: %w", err)
		}
	}

	return nil
}

// DropTables drops all nudge-related tables (for testing cleanup)
func DropTables(db *gorm.DB) error {
	// Drop tables in reverse order to handle foreign key dependencies
	tables := []string{
		"reminders",
		"nudge_settings",
		"tasks",
	}

	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)).Error; err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	return nil
}

// ValidateMigrations checks if all required tables and indexes exist
func ValidateMigrations(db *gorm.DB) error {
	// Check if tables exist
	requiredTables := []string{"users", "tasks", "reminders", "nudge_settings"}

	for _, table := range requiredTables {
		var exists bool
		err := db.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = ?)", table).Scan(&exists).Error
		if err != nil {
			return fmt.Errorf("failed to check table existence for %s: %w", table, err)
		}
		if !exists {
			return fmt.Errorf("required table %s does not exist", table)
		}
	}

	// Check if key indexes exist
	requiredIndexes := []string{
		"idx_tasks_user_id",
		"idx_tasks_status",
		"idx_reminders_scheduled_at",
		"idx_reminders_task_id",
	}

	for _, index := range requiredIndexes {
		var exists bool
		err := db.Raw("SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = ?)", index).Scan(&exists).Error
		if err != nil {
			return fmt.Errorf("failed to check index existence for %s: %w", index, err)
		}
		if !exists {
			return fmt.Errorf("required index %s does not exist", index)
		}
	}

	return nil
}

// MigrateWithValidation runs migrations and validates the result
func MigrateWithValidation(db *gorm.DB) error {
	if err := RunMigrations(db); err != nil {
		return err
	}

	if err := ValidateMigrations(db); err != nil {
		return fmt.Errorf("migration validation failed: %w", err)
	}

	return nil
}

// GetTableStats returns statistics about nudge tables
func GetTableStats(db *gorm.DB) (map[string]int64, error) {
	stats := make(map[string]int64)

	// Count tasks
	var taskCount int64
	if err := db.Model(&Task{}).Count(&taskCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", err)
	}
	stats["tasks"] = taskCount

	// Count reminders
	var reminderCount int64
	if err := db.Model(&Reminder{}).Count(&reminderCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count reminders: %w", err)
	}
	stats["reminders"] = reminderCount

	// Count nudge settings
	var settingsCount int64
	if err := db.Model(&NudgeSettings{}).Count(&settingsCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count nudge settings: %w", err)
	}
	stats["nudge_settings"] = settingsCount

	return stats, nil
}
