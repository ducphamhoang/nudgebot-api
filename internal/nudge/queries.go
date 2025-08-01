package nudge

import (
	"time"

	"nudgebot-api/internal/common"

	"gorm.io/gorm"
)

// QueryBuilder provides a fluent interface for building complex GORM queries
type QueryBuilder struct {
	db *gorm.DB
}

// NewQueryBuilder creates a new QueryBuilder
func NewQueryBuilder(db *gorm.DB) *QueryBuilder {
	return &QueryBuilder{db: db}
}

// TaskQueryBuilder provides task-specific query building
type TaskQueryBuilder struct {
	query *gorm.DB
}

// TaskQuery creates a new TaskQueryBuilder
func (qb *QueryBuilder) TaskQuery() *TaskQueryBuilder {
	return &TaskQueryBuilder{
		query: qb.db.Model(&Task{}),
	}
}

// WithUserID filters tasks by user ID
func (tqb *TaskQueryBuilder) WithUserID(userID common.UserID) *TaskQueryBuilder {
	tqb.query = tqb.query.Where("user_id = ?", userID)
	return tqb
}

// WithStatus filters tasks by status
func (tqb *TaskQueryBuilder) WithStatus(status common.TaskStatus) *TaskQueryBuilder {
	tqb.query = tqb.query.Where("status = ?", status)
	return tqb
}

// WithStatuses filters tasks by multiple statuses
func (tqb *TaskQueryBuilder) WithStatuses(statuses []common.TaskStatus) *TaskQueryBuilder {
	tqb.query = tqb.query.Where("status IN ?", statuses)
	return tqb
}

// WithPriority filters tasks by priority
func (tqb *TaskQueryBuilder) WithPriority(priority common.Priority) *TaskQueryBuilder {
	tqb.query = tqb.query.Where("priority = ?", priority)
	return tqb
}

// WithDueDateRange filters tasks by due date range
func (tqb *TaskQueryBuilder) WithDueDateRange(after, before *time.Time) *TaskQueryBuilder {
	if after != nil {
		tqb.query = tqb.query.Where("due_date >= ?", after)
	}
	if before != nil {
		tqb.query = tqb.query.Where("due_date <= ?", before)
	}
	return tqb
}

// WithOverdue filters for overdue tasks
func (tqb *TaskQueryBuilder) WithOverdue() *TaskQueryBuilder {
	now := time.Now()
	tqb.query = tqb.query.Where("due_date < ? AND status = ?", now, common.TaskStatusActive)
	return tqb
}

// WithDueSoon filters for tasks due within a specific duration
func (tqb *TaskQueryBuilder) WithDueSoon(within time.Duration) *TaskQueryBuilder {
	now := time.Now()
	dueSoon := now.Add(within)
	tqb.query = tqb.query.Where("due_date BETWEEN ? AND ? AND status = ?", now, dueSoon, common.TaskStatusActive)
	return tqb
}

// OrderByPriority orders tasks by priority (urgent first)
func (tqb *TaskQueryBuilder) OrderByPriority() *TaskQueryBuilder {
	tqb.query = tqb.query.Order("CASE priority WHEN 'urgent' THEN 4 WHEN 'high' THEN 3 WHEN 'medium' THEN 2 WHEN 'low' THEN 1 END DESC")
	return tqb
}

// OrderByDueDate orders tasks by due date (earliest first)
func (tqb *TaskQueryBuilder) OrderByDueDate() *TaskQueryBuilder {
	tqb.query = tqb.query.Order("due_date ASC NULLS LAST")
	return tqb
}

// OrderByCreatedAt orders tasks by creation date
func (tqb *TaskQueryBuilder) OrderByCreatedAt(ascending bool) *TaskQueryBuilder {
	if ascending {
		tqb.query = tqb.query.Order("created_at ASC")
	} else {
		tqb.query = tqb.query.Order("created_at DESC")
	}
	return tqb
}

// OrderByUpdatedAt orders tasks by update date
func (tqb *TaskQueryBuilder) OrderByUpdatedAt(ascending bool) *TaskQueryBuilder {
	if ascending {
		tqb.query = tqb.query.Order("updated_at ASC")
	} else {
		tqb.query = tqb.query.Order("updated_at DESC")
	}
	return tqb
}

// WithPagination applies pagination to the query
func (tqb *TaskQueryBuilder) WithPagination(limit, offset int) *TaskQueryBuilder {
	if limit > 0 {
		tqb.query = tqb.query.Limit(limit)
	}
	if offset > 0 {
		tqb.query = tqb.query.Offset(offset)
	}
	return tqb
}

// Find executes the query and returns tasks
func (tqb *TaskQueryBuilder) Find() ([]*Task, error) {
	var tasks []*Task
	err := tqb.query.Find(&tasks).Error
	return tasks, err
}

// Count returns the count of matching tasks
func (tqb *TaskQueryBuilder) Count() (int64, error) {
	var count int64
	err := tqb.query.Count(&count).Error
	return count, err
}

// First returns the first matching task
func (tqb *TaskQueryBuilder) First() (*Task, error) {
	var task Task
	err := tqb.query.First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// ReminderQueryBuilder provides reminder-specific query building
type ReminderQueryBuilder struct {
	query *gorm.DB
}

// ReminderQuery creates a new ReminderQueryBuilder
func (qb *QueryBuilder) ReminderQuery() *ReminderQueryBuilder {
	return &ReminderQueryBuilder{
		query: qb.db.Model(&Reminder{}),
	}
}

// WithTaskID filters reminders by task ID
func (rqb *ReminderQueryBuilder) WithTaskID(taskID common.TaskID) *ReminderQueryBuilder {
	rqb.query = rqb.query.Where("task_id = ?", taskID)
	return rqb
}

// WithUserID filters reminders by user ID
func (rqb *ReminderQueryBuilder) WithUserID(userID common.UserID) *ReminderQueryBuilder {
	rqb.query = rqb.query.Where("user_id = ?", userID)
	return rqb
}

// WithDueBefore filters reminders due before a specific time
func (rqb *ReminderQueryBuilder) WithDueBefore(before time.Time) *ReminderQueryBuilder {
	rqb.query = rqb.query.Where("scheduled_at <= ?", before)
	return rqb
}

// WithUnsent filters for reminders that haven't been sent
func (rqb *ReminderQueryBuilder) WithUnsent() *ReminderQueryBuilder {
	rqb.query = rqb.query.Where("sent_at IS NULL")
	return rqb
}

// WithReminderType filters by reminder type
func (rqb *ReminderQueryBuilder) WithReminderType(reminderType ReminderType) *ReminderQueryBuilder {
	rqb.query = rqb.query.Where("reminder_type = ?", reminderType)
	return rqb
}

// WithTaskJoin joins with tasks table
func (rqb *ReminderQueryBuilder) WithTaskJoin() *ReminderQueryBuilder {
	rqb.query = rqb.query.Joins("JOIN tasks ON reminders.task_id = tasks.id")
	return rqb
}

// Find executes the query and returns reminders
func (rqb *ReminderQueryBuilder) Find() ([]*Reminder, error) {
	var reminders []*Reminder
	err := rqb.query.Find(&reminders).Error
	return reminders, err
}

// Optimized query methods

// GetTasksWithReminders retrieves tasks with their associated reminders
func GetTasksWithReminders(db *gorm.DB, userID common.UserID, filter TaskFilter) ([]*Task, error) {
	query := db.Model(&Task{}).Preload("Reminders").Where("user_id = ?", userID)

	// Apply filters
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.Priority != nil {
		query = query.Where("priority = ?", *filter.Priority)
	}
	if filter.DueBefore != nil {
		query = query.Where("due_date <= ?", *filter.DueBefore)
	}
	if filter.DueAfter != nil {
		query = query.Where("due_date >= ?", *filter.DueAfter)
	}

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	var tasks []*Task
	err := query.Find(&tasks).Error
	return tasks, err
}

// GetOverdueTasksWithCounts retrieves overdue tasks with reminder counts
func GetOverdueTasksWithCounts(db *gorm.DB, userID common.UserID) ([]*Task, error) {
	var tasks []*Task
	now := time.Now()

	err := db.Model(&Task{}).
		Select("tasks.*, COUNT(reminders.id) as reminder_count").
		Joins("LEFT JOIN reminders ON tasks.id = reminders.task_id").
		Where("tasks.user_id = ? AND tasks.status = ? AND tasks.due_date < ?", userID, common.TaskStatusActive, now).
		Group("tasks.id").
		Order("tasks.due_date ASC").
		Find(&tasks).Error

	return tasks, err
}

// GetTasksDueSoon retrieves tasks due within a specific time window
func GetTasksDueSoon(db *gorm.DB, userID common.UserID, within time.Duration) ([]*Task, error) {
	now := time.Now()
	dueSoon := now.Add(within)

	var tasks []*Task
	err := db.Model(&Task{}).
		Where("user_id = ? AND status = ? AND due_date BETWEEN ? AND ?", userID, common.TaskStatusActive, now, dueSoon).
		Order("due_date ASC").
		Find(&tasks).Error

	return tasks, err
}

// GetUserTaskSummary retrieves comprehensive task statistics for a user
func GetUserTaskSummary(db *gorm.DB, userID common.UserID) (*TaskStats, error) {
	var stats TaskStats

	// Get total tasks
	err := db.Model(&Task{}).Where("user_id = ? AND status != ?", userID, common.TaskStatusDeleted).Count(&stats.TotalTasks).Error
	if err != nil {
		return nil, err
	}

	// Get completed tasks
	err = db.Model(&Task{}).Where("user_id = ? AND status = ?", userID, common.TaskStatusCompleted).Count(&stats.CompletedTasks).Error
	if err != nil {
		return nil, err
	}

	// Get active tasks
	err = db.Model(&Task{}).Where("user_id = ? AND status = ?", userID, common.TaskStatusActive).Count(&stats.ActiveTasks).Error
	if err != nil {
		return nil, err
	}

	// Get overdue tasks
	now := time.Now()
	err = db.Model(&Task{}).Where("user_id = ? AND status = ? AND due_date < ?", userID, common.TaskStatusActive, now).Count(&stats.OverdueTasks).Error
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// Batch operations

// BulkUpdateTaskStatus updates multiple tasks' status in a single operation
func BulkUpdateTaskStatus(db *gorm.DB, taskIDs []common.TaskID, status common.TaskStatus) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": now,
	}

	if status == common.TaskStatusCompleted {
		updates["completed_at"] = now
	}

	return db.Model(&Task{}).Where("id IN ?", taskIDs).Updates(updates).Error
}

// BulkCreateReminders creates multiple reminders in a single operation
func BulkCreateReminders(db *gorm.DB, reminders []*Reminder) error {
	if len(reminders) == 0 {
		return nil
	}

	return db.CreateInBatches(reminders, 100).Error
}

// CleanupOldReminders deletes sent reminders older than the specified time
func CleanupOldReminders(db *gorm.DB, before time.Time) error {
	return db.Where("sent_at IS NOT NULL AND sent_at < ?", before).Delete(&Reminder{}).Error
}

// Analytics queries

// GetTaskCompletionRate calculates task completion rate for a user
func GetTaskCompletionRate(db *gorm.DB, userID common.UserID, since time.Time) (float64, error) {
	var total, completed int64

	// Get total tasks created since the specified time
	err := db.Model(&Task{}).Where("user_id = ? AND created_at >= ?", userID, since).Count(&total).Error
	if err != nil {
		return 0, err
	}

	if total == 0 {
		return 0, nil
	}

	// Get completed tasks created since the specified time
	err = db.Model(&Task{}).Where("user_id = ? AND created_at >= ? AND status = ?", userID, since, common.TaskStatusCompleted).Count(&completed).Error
	if err != nil {
		return 0, err
	}

	return float64(completed) / float64(total) * 100, nil
}

// GetUserEngagementMetrics calculates user engagement metrics
func GetUserEngagementMetrics(db *gorm.DB, userID common.UserID, since time.Time) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// Tasks created
	var tasksCreated int64
	err := db.Model(&Task{}).Where("user_id = ? AND created_at >= ?", userID, since).Count(&tasksCreated).Error
	if err != nil {
		return nil, err
	}
	metrics["tasks_created"] = tasksCreated

	// Tasks completed
	var tasksCompleted int64
	err = db.Model(&Task{}).Where("user_id = ? AND completed_at >= ?", userID, since).Count(&tasksCompleted).Error
	if err != nil {
		return nil, err
	}
	metrics["tasks_completed"] = tasksCompleted

	// Average completion time (in hours)
	var avgCompletionTime float64
	err = db.Model(&Task{}).
		Select("AVG(EXTRACT(EPOCH FROM (completed_at - created_at))/3600) as avg_hours").
		Where("user_id = ? AND completed_at >= ? AND completed_at IS NOT NULL", userID, since).
		Scan(&avgCompletionTime).Error
	if err != nil {
		return nil, err
	}
	metrics["avg_completion_time_hours"] = avgCompletionTime

	// Reminders sent
	var remindersSent int64
	err = db.Model(&Reminder{}).Where("user_id = ? AND sent_at >= ?", userID, since).Count(&remindersSent).Error
	if err != nil {
		return nil, err
	}
	metrics["reminders_sent"] = remindersSent

	return metrics, nil
}

// GetReminderEffectivenessStats calculates reminder effectiveness
func GetReminderEffectivenessStats(db *gorm.DB, userID common.UserID, since time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Tasks completed within 24 hours of reminder
	var quickCompletions int64
	err := db.Model(&Task{}).
		Joins("JOIN reminders ON tasks.id = reminders.task_id").
		Where("tasks.user_id = ? AND reminders.sent_at >= ? AND tasks.completed_at IS NOT NULL AND tasks.completed_at <= reminders.sent_at + INTERVAL '24 hours'", userID, since).
		Count(&quickCompletions).Error
	if err != nil {
		return nil, err
	}
	stats["quick_completions"] = quickCompletions

	// Total reminders sent
	var totalReminders int64
	err = db.Model(&Reminder{}).Where("user_id = ? AND sent_at >= ?", userID, since).Count(&totalReminders).Error
	if err != nil {
		return nil, err
	}
	stats["total_reminders"] = totalReminders

	// Effectiveness rate
	if totalReminders > 0 {
		stats["effectiveness_rate"] = float64(quickCompletions) / float64(totalReminders) * 100
	} else {
		stats["effectiveness_rate"] = 0.0
	}

	return stats, nil
}
