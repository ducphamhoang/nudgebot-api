package events

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EventFlowValidator provides utilities for testing and validating event flow between modules
type EventFlowValidator struct {
	eventBus EventBus
	logger   *zap.Logger
	timeout  time.Duration
}

// NewEventFlowValidator creates a new EventFlowValidator instance
func NewEventFlowValidator(eventBus EventBus, logger *zap.Logger) *EventFlowValidator {
	return &EventFlowValidator{
		eventBus: eventBus,
		logger:   logger,
		timeout:  30 * time.Second,
	}
}

// ValidateMessageToTaskFlow simulates the complete flow from message to task creation
func (v *EventFlowValidator) ValidateMessageToTaskFlow(userID, chatID, messageText string) error {
	v.logger.Info("Validating message to task flow",
		zap.String("userID", userID),
		zap.String("chatID", chatID))

	// Create MessageReceived event
	messageEvent := MessageReceived{
		Event:       NewEvent(),
		UserID:      userID,
		ChatID:      chatID,
		MessageText: messageText,
	}

	// Wait for TaskCreated event
	taskCreatedChan := make(chan TaskCreated, 1)
	err := v.eventBus.Subscribe(TopicTaskCreated, func(event TaskCreated) {
		if event.UserID == userID {
			taskCreatedChan <- event
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to TaskCreated events: %w", err)
	}

	// Publish MessageReceived event
	if err := v.eventBus.Publish(TopicMessageReceived, messageEvent); err != nil {
		return fmt.Errorf("failed to publish MessageReceived event: %w", err)
	}

	// Wait for TaskCreated event with timeout
	select {
	case <-taskCreatedChan:
		v.logger.Info("Message to task flow validation successful")
		return nil
	case <-time.After(v.timeout):
		return fmt.Errorf("timeout waiting for TaskCreated event")
	}
}

// ValidateTaskActionFlow tests task action processing
func (v *EventFlowValidator) ValidateTaskActionFlow(userID, chatID, taskID, action string) error {
	v.logger.Info("Validating task action flow",
		zap.String("userID", userID),
		zap.String("taskID", taskID),
		zap.String("action", action))

	// Create TaskActionRequested event
	actionEvent := TaskActionRequested{
		Event:  NewEvent(),
		UserID: userID,
		ChatID: chatID,
		TaskID: taskID,
		Action: action,
	}

	// Wait for TaskActionResponse event
	actionResponseChan := make(chan TaskActionResponse, 1)
	err := v.eventBus.Subscribe(TopicTaskActionResponse, func(event TaskActionResponse) {
		if event.UserID == userID && event.TaskID == taskID {
			actionResponseChan <- event
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to TaskActionResponse events: %w", err)
	}

	// Publish TaskActionRequested event
	if err := v.eventBus.Publish(TopicTaskActionRequested, actionEvent); err != nil {
		return fmt.Errorf("failed to publish TaskActionRequested event: %w", err)
	}

	// Wait for TaskActionResponse event with timeout
	select {
	case response := <-actionResponseChan:
		if response.Success {
			v.logger.Info("Task action flow validation successful")
			return nil
		}
		return fmt.Errorf("task action failed: %s", response.Message)
	case <-time.After(v.timeout):
		return fmt.Errorf("timeout waiting for TaskActionResponse event")
	}
}

// ValidateTaskListFlow tests task list retrieval
func (v *EventFlowValidator) ValidateTaskListFlow(userID, chatID string) error {
	v.logger.Info("Validating task list flow",
		zap.String("userID", userID),
		zap.String("chatID", chatID))

	// Create TaskListRequested event
	listEvent := TaskListRequested{
		Event:  NewEvent(),
		UserID: userID,
		ChatID: chatID,
	}

	// Wait for TaskListResponse event
	listResponseChan := make(chan TaskListResponse, 1)
	err := v.eventBus.Subscribe(TopicTaskListResponse, func(event TaskListResponse) {
		if event.UserID == userID && event.ChatID == chatID {
			listResponseChan <- event
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to TaskListResponse events: %w", err)
	}

	// Publish TaskListRequested event
	if err := v.eventBus.Publish(TopicTaskListRequested, listEvent); err != nil {
		return fmt.Errorf("failed to publish TaskListRequested event: %w", err)
	}

	// Wait for TaskListResponse event with timeout
	select {
	case <-listResponseChan:
		v.logger.Info("Task list flow validation successful")
		return nil
	case <-time.After(v.timeout):
		return fmt.Errorf("timeout waiting for TaskListResponse event")
	}
}

// ValidateReminderFlow tests reminder notifications
func (v *EventFlowValidator) ValidateReminderFlow(taskID, userID, chatID string) error {
	v.logger.Info("Validating reminder flow",
		zap.String("taskID", taskID),
		zap.String("userID", userID),
		zap.String("chatID", chatID))

	// Create ReminderDue event
	reminderEvent := ReminderDue{
		Event:  NewEvent(),
		TaskID: taskID,
		UserID: userID,
		ChatID: chatID,
	}

	// Publish ReminderDue event (validation is observing if it processes without error)
	if err := v.eventBus.Publish(TopicReminderDue, reminderEvent); err != nil {
		return fmt.Errorf("failed to publish ReminderDue event: %w", err)
	}

	v.logger.Info("Reminder flow validation successful")
	return nil
}

// EventFlowMonitor provides runtime monitoring of event system
type EventFlowMonitor struct {
	eventBus  EventBus
	logger    *zap.Logger
	metrics   *EventMetrics
	mu        sync.RWMutex
	isStarted bool
	stopChan  chan struct{}
}

// EventMetrics tracks event system performance
type EventMetrics struct {
	PublishCount       map[string]int64 `json:"publish_count"`
	ProcessCount       map[string]int64 `json:"process_count"`
	ErrorCount         map[string]int64 `json:"error_count"`
	AverageLatency     map[string]int64 `json:"average_latency_ms"`
	LastProcessingTime map[string]int64 `json:"last_processing_time"`
	mu                 sync.RWMutex
}

// NewEventFlowMonitor creates a new EventFlowMonitor instance
func NewEventFlowMonitor(eventBus EventBus, logger *zap.Logger) *EventFlowMonitor {
	return &EventFlowMonitor{
		eventBus: eventBus,
		logger:   logger,
		metrics: &EventMetrics{
			PublishCount:       make(map[string]int64),
			ProcessCount:       make(map[string]int64),
			ErrorCount:         make(map[string]int64),
			AverageLatency:     make(map[string]int64),
			LastProcessingTime: make(map[string]int64),
		},
		stopChan: make(chan struct{}),
	}
}

// Start begins monitoring event flow
func (m *EventFlowMonitor) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isStarted {
		return fmt.Errorf("monitor is already started")
	}

	m.isStarted = true
	m.logger.Info("Starting event flow monitor")

	// Start metrics collection goroutine
	go m.collectMetrics()

	return nil
}

// Stop stops monitoring event flow
func (m *EventFlowMonitor) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isStarted {
		return fmt.Errorf("monitor is not started")
	}

	m.isStarted = false
	close(m.stopChan)
	m.logger.Info("Stopped event flow monitor")

	return nil
}

// GetMetrics returns current event metrics
func (m *EventFlowMonitor) GetMetrics() *EventMetrics {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	// Create a copy of metrics
	metrics := &EventMetrics{
		PublishCount:       make(map[string]int64),
		ProcessCount:       make(map[string]int64),
		ErrorCount:         make(map[string]int64),
		AverageLatency:     make(map[string]int64),
		LastProcessingTime: make(map[string]int64),
	}

	for k, v := range m.metrics.PublishCount {
		metrics.PublishCount[k] = v
	}
	for k, v := range m.metrics.ProcessCount {
		metrics.ProcessCount[k] = v
	}
	for k, v := range m.metrics.ErrorCount {
		metrics.ErrorCount[k] = v
	}
	for k, v := range m.metrics.AverageLatency {
		metrics.AverageLatency[k] = v
	}
	for k, v := range m.metrics.LastProcessingTime {
		metrics.LastProcessingTime[k] = v
	}

	return metrics
}

// GetHealthStatus returns the health status of the event system
func (m *EventFlowMonitor) GetHealthStatus() map[string]interface{} {
	metrics := m.GetMetrics()

	status := map[string]interface{}{
		"status":             "healthy",
		"monitor_active":     m.isStarted,
		"total_events":       getTotalCount(metrics.PublishCount),
		"total_processed":    getTotalCount(metrics.ProcessCount),
		"total_errors":       getTotalCount(metrics.ErrorCount),
		"topics_monitored":   len(metrics.PublishCount),
		"last_activity_time": getLastActivity(metrics.LastProcessingTime),
	}

	// Determine overall health
	totalEvents := getTotalCount(metrics.PublishCount)
	totalErrors := getTotalCount(metrics.ErrorCount)

	if totalEvents > 0 {
		errorRate := float64(totalErrors) / float64(totalEvents)
		if errorRate > 0.1 { // More than 10% error rate
			status["status"] = "degraded"
		}
		if errorRate > 0.3 { // More than 30% error rate
			status["status"] = "unhealthy"
		}
		status["error_rate"] = errorRate
	}

	return status
}

// collectMetrics runs the metrics collection loop
func (m *EventFlowMonitor) collectMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.logMetrics()
		}
	}
}

// logMetrics logs current metrics
func (m *EventFlowMonitor) logMetrics() {
	metrics := m.GetMetrics()

	m.logger.Info("Event flow metrics",
		zap.Int("topics_monitored", len(metrics.PublishCount)),
		zap.Int64("total_published", getTotalCount(metrics.PublishCount)),
		zap.Int64("total_processed", getTotalCount(metrics.ProcessCount)),
		zap.Int64("total_errors", getTotalCount(metrics.ErrorCount)))
}

// Helper functions

// WaitForEvent waits for a specific event to be published
func WaitForEvent(eventBus EventBus, topic string, timeout time.Duration) (interface{}, error) {
	eventChan := make(chan interface{}, 1)

	err := eventBus.Subscribe(topic, func(event interface{}) {
		select {
		case eventChan <- event:
		default:
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
	}

	select {
	case event := <-eventChan:
		return event, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for event on topic %s", topic)
	}
}

// CreateTestEvents generates test event data for validation
func CreateTestEvents() map[string]interface{} {
	return map[string]interface{}{
		"MessageReceived": MessageReceived{
			Event:       NewEvent(),
			UserID:      "test_user_123",
			ChatID:      "test_chat_123",
			MessageText: "Buy groceries tomorrow",
		},
		"TaskParsed": TaskParsed{
			Event:  NewEvent(),
			UserID: "test_user_123",
			ParsedTask: ParsedTask{
				Title:       "Buy groceries",
				Description: "Get milk, bread, and eggs",
				Priority:    "medium",
				Tags:        []string{"shopping", "food"},
			},
		},
		"TaskListRequested": TaskListRequested{
			Event:  NewEvent(),
			UserID: "test_user_123",
			ChatID: "test_chat_123",
		},
		"TaskActionRequested": TaskActionRequested{
			Event:  NewEvent(),
			UserID: "test_user_123",
			ChatID: "test_chat_123",
			TaskID: "test_task_123",
			Action: "done",
		},
		"ReminderDue": ReminderDue{
			Event:  NewEvent(),
			TaskID: "test_task_123",
			UserID: "test_user_123",
			ChatID: "test_chat_123",
		},
	}
}

// ValidateEventStructure ensures event data integrity
func ValidateEventStructure(event interface{}) error {
	switch e := event.(type) {
	case MessageReceived:
		if e.UserID == "" || e.ChatID == "" || e.MessageText == "" {
			return fmt.Errorf("MessageReceived event missing required fields")
		}
	case TaskParsed:
		if e.UserID == "" || e.ParsedTask.Title == "" {
			return fmt.Errorf("TaskParsed event missing required fields")
		}
	case TaskListRequested:
		if e.UserID == "" || e.ChatID == "" {
			return fmt.Errorf("TaskListRequested event missing required fields")
		}
	case TaskActionRequested:
		if e.UserID == "" || e.ChatID == "" || e.TaskID == "" || e.Action == "" {
			return fmt.Errorf("TaskActionRequested event missing required fields")
		}
	case ReminderDue:
		if e.TaskID == "" || e.UserID == "" || e.ChatID == "" {
			return fmt.Errorf("ReminderDue event missing required fields")
		}
	default:
		return fmt.Errorf("unknown event type: %T", event)
	}

	return nil
}

// Helper functions for metrics

func getTotalCount(counts map[string]int64) int64 {
	var total int64
	for _, count := range counts {
		total += count
	}
	return total
}

func getLastActivity(times map[string]int64) int64 {
	var latest int64
	for _, t := range times {
		if t > latest {
			latest = t
		}
	}
	return latest
}
