package chatbot

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"nudgebot-api/internal/common"
	"nudgebot-api/internal/events"

	"go.uber.org/zap"
)

// CommandProcessor handles bot command processing
type CommandProcessor struct {
	eventBus       events.EventBus
	logger         *zap.Logger
	sessionManager *SessionManager
}

// NewCommandProcessor creates a new CommandProcessor instance
func NewCommandProcessor(eventBus events.EventBus, logger *zap.Logger) *CommandProcessor {
	return &CommandProcessor{
		eventBus:       eventBus,
		logger:         logger,
		sessionManager: NewSessionManager(),
	}
}

// ProcessStartCommand handles the /start command
func (cp *CommandProcessor) ProcessStartCommand(userID, chatID string) (string, error) {
	cp.logger.Info("Processing start command",
		zap.String("user_id", userID),
		zap.String("chat_id", chatID))

	// Create user session
	session := &ChatSession{
		UserID:       common.UserID(userID),
		ChatID:       common.ChatID(chatID),
		State:        SessionStateIdle,
		Context:      "",
		LastActivity: time.Now(),
	}

	cp.sessionManager.SetSession(userID, session)

	// Publish session started event
	sessionEvent := events.UserSessionStarted{
		Event:       events.NewEvent(),
		UserID:      userID,
		ChatID:      chatID,
		SessionType: "telegram",
	}

	cp.eventBus.Publish(events.TopicUserSessionStarted, sessionEvent)

	welcomeText := `🤖 <b>Welcome to NudgeBot!</b>

I'm here to help you manage your tasks and stay productive.

<b>What I can do:</b>
• Parse tasks from natural language
• Send reminders when tasks are due
• Help you mark tasks as complete
• Show your task list

<b>How to get started:</b>
Just send me a message describing what you need to do! For example:
"Finish the presentation by tomorrow"
"Call mom this evening"
"Buy groceries"

Use /help to see all available commands.`

	return welcomeText, nil
}

// ProcessHelpCommand handles the /help command
func (cp *CommandProcessor) ProcessHelpCommand(userID, chatID string) (string, error) {
	cp.logger.Info("Processing help command",
		zap.String("user_id", userID),
		zap.String("chat_id", chatID))

	helpText := `🆘 <b>NudgeBot Help</b>

<b>Available Commands:</b>
/start - Start or restart the bot
/help - Show this help message
/list - Show your active tasks
/done [task] - Mark a task as complete
/delete [task] - Delete a task

<b>How to use:</b>
• Send any message to create a new task
• Use the inline buttons to manage your tasks
• Tasks are automatically parsed from your messages

<b>Examples:</b>
"Meeting with John tomorrow at 3pm"
"Finish project report by Friday"
"Buy milk and bread"

The bot will extract the task details and ask for confirmation before adding them to your list.`

	return helpText, nil
}

// ProcessListCommand handles the /list command
func (cp *CommandProcessor) ProcessListCommand(userID, chatID string) error {
	cp.logger.Info("Processing list command",
		zap.String("user_id", userID),
		zap.String("chat_id", chatID))

	// Publish task list requested event
	listEvent := events.TaskListRequested{
		Event:  events.NewEvent(),
		UserID: userID,
		ChatID: chatID,
	}

	cp.eventBus.Publish(events.TopicTaskListRequested, listEvent)

	return nil
}

// ProcessDoneCommand handles the /done command
func (cp *CommandProcessor) ProcessDoneCommand(userID, chatID string, args []string) (string, error) {
	cp.logger.Info("Processing done command",
		zap.String("user_id", userID),
		zap.String("chat_id", chatID),
		zap.Strings("args", args))

	if len(args) == 0 {
		return "Please specify a task ID or use the inline buttons to mark tasks as complete.", nil
	}

	taskID := args[0]

	// Publish task action requested event
	actionEvent := events.TaskActionRequested{
		Event:  events.NewEvent(),
		UserID: userID,
		ChatID: chatID,
		TaskID: taskID,
		Action: "done",
	}

	cp.eventBus.Publish(events.TopicTaskActionRequested, actionEvent)

	return fmt.Sprintf("Marking task %s as complete...", taskID), nil
}

// ProcessDeleteCommand handles the /delete command
func (cp *CommandProcessor) ProcessDeleteCommand(userID, chatID string, args []string) (string, error) {
	cp.logger.Info("Processing delete command",
		zap.String("user_id", userID),
		zap.String("chat_id", chatID),
		zap.Strings("args", args))

	if len(args) == 0 {
		return "Please specify a task ID or use the inline buttons to delete tasks.", nil
	}

	taskID := args[0]

	// Publish task action requested event
	actionEvent := events.TaskActionRequested{
		Event:  events.NewEvent(),
		UserID: userID,
		ChatID: chatID,
		TaskID: taskID,
		Action: "delete",
	}

	cp.eventBus.Publish(events.TopicTaskActionRequested, actionEvent)

	return fmt.Sprintf("Deleting task %s...", taskID), nil
}

// HandleCallbackQuery processes inline keyboard button presses
func (cp *CommandProcessor) HandleCallbackQuery(callbackData *CallbackData, userID, chatID string) (string, error) {
	cp.logger.Info("Processing callback query",
		zap.String("user_id", userID),
		zap.String("chat_id", chatID),
		zap.String("action", callbackData.Action))

	switch callbackData.Action {
	case CallbackActionDone:
		return cp.handleDoneCallback(callbackData, userID, chatID)
	case CallbackActionDelete:
		return cp.handleDeleteCallback(callbackData, userID, chatID)
	case CallbackActionSnooze:
		return cp.handleSnoozeCallback(callbackData, userID, chatID)
	case CallbackActionList:
		return cp.handleListCallback(callbackData, userID, chatID)
	case CallbackActionConfirm:
		return cp.handleConfirmCallback(callbackData, userID, chatID)
	case CallbackActionCancel:
		return cp.handleCancelCallback(callbackData, userID, chatID)
	default:
		return "Unknown action.", nil
	}
}

// handleDoneCallback processes done button presses
func (cp *CommandProcessor) handleDoneCallback(callbackData *CallbackData, userID, chatID string) (string, error) {
	taskID, exists := callbackData.Data["task_id"]
	if !exists {
		return "Invalid task ID.", nil
	}

	// Publish task action requested event
	actionEvent := events.TaskActionRequested{
		Event:  events.NewEvent(),
		UserID: userID,
		ChatID: chatID,
		TaskID: taskID,
		Action: "done",
	}

	cp.eventBus.Publish(events.TopicTaskActionRequested, actionEvent)

	return "✅ Task marked as complete!", nil
}

// handleDeleteCallback processes delete button presses
func (cp *CommandProcessor) handleDeleteCallback(callbackData *CallbackData, userID, chatID string) (string, error) {
	taskID, exists := callbackData.Data["task_id"]
	if !exists {
		return "Invalid task ID.", nil
	}

	// Publish task action requested event
	actionEvent := events.TaskActionRequested{
		Event:  events.NewEvent(),
		UserID: userID,
		ChatID: chatID,
		TaskID: taskID,
		Action: "delete",
	}

	cp.eventBus.Publish(events.TopicTaskActionRequested, actionEvent)

	return "🗑️ Task deleted!", nil
}

// handleSnoozeCallback processes snooze button presses
func (cp *CommandProcessor) handleSnoozeCallback(callbackData *CallbackData, userID, chatID string) (string, error) {
	taskID, exists := callbackData.Data["task_id"]
	if !exists {
		return "Invalid task ID.", nil
	}

	// Publish task action requested event
	actionEvent := events.TaskActionRequested{
		Event:  events.NewEvent(),
		UserID: userID,
		ChatID: chatID,
		TaskID: taskID,
		Action: "snooze",
	}

	cp.eventBus.Publish(events.TopicTaskActionRequested, actionEvent)

	return "⏰ Task snoozed for 1 hour!", nil
}

// handleListCallback processes list button presses
func (cp *CommandProcessor) handleListCallback(callbackData *CallbackData, userID, chatID string) (string, error) {
	// Publish task list requested event
	listEvent := events.TaskListRequested{
		Event:  events.NewEvent(),
		UserID: userID,
		ChatID: chatID,
	}

	cp.eventBus.Publish(events.TopicTaskListRequested, listEvent)

	return "", nil // Response will be sent via event handler
}

// handleConfirmCallback processes confirmation button presses
func (cp *CommandProcessor) handleConfirmCallback(callbackData *CallbackData, userID, chatID string) (string, error) {
	action, exists := callbackData.Data["action"]
	if !exists {
		return "Invalid confirmation action.", nil
	}

	taskID, exists := callbackData.Data["task_id"]
	if !exists {
		return "Invalid task ID.", nil
	}

	// Execute the confirmed action
	actionEvent := events.TaskActionRequested{
		Event:  events.NewEvent(),
		UserID: userID,
		ChatID: chatID,
		TaskID: taskID,
		Action: action,
	}

	cp.eventBus.Publish(events.TopicTaskActionRequested, actionEvent)

	return fmt.Sprintf("✅ Action '%s' confirmed for task %s", action, taskID), nil
}

// handleCancelCallback processes cancel button presses
func (cp *CommandProcessor) handleCancelCallback(callbackData *CallbackData, userID, chatID string) (string, error) {
	return "❌ Action cancelled.", nil
}

// SessionManager manages user chat sessions
type SessionManager struct {
	sessions map[string]*ChatSession
	mutex    sync.RWMutex
}

// NewSessionManager creates a new SessionManager instance
func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*ChatSession),
	}

	// Start cleanup routine
	go sm.cleanupRoutine()

	return sm
}

// GetSession retrieves a user's session
func (sm *SessionManager) GetSession(userID string) (*ChatSession, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[userID]
	return session, exists
}

// SetSession stores a user's session
func (sm *SessionManager) SetSession(userID string, session *ChatSession) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.sessions[userID] = session
}

// UpdateLastActivity updates the last activity time for a session
func (sm *SessionManager) UpdateLastActivity(userID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if session, exists := sm.sessions[userID]; exists {
		session.LastActivity = time.Now()
	}
}

// cleanupRoutine removes inactive sessions
func (sm *SessionManager) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.cleanupInactiveSessions()
		}
	}
}

// cleanupInactiveSessions removes sessions inactive for more than 24 hours
func (sm *SessionManager) cleanupInactiveSessions() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)

	for userID, session := range sm.sessions {
		if session.LastActivity.Before(cutoff) {
			delete(sm.sessions, userID)
		}
	}
}

// parseCommandArgs extracts arguments from command text
func parseCommandArgs(text string) []string {
	parts := strings.Fields(text)
	if len(parts) <= 1 {
		return []string{}
	}
	return parts[1:]
}

// parseChatID converts string to int64 for Telegram API
func parseChatID(chatID string) (int64, error) {
	return strconv.ParseInt(chatID, 10, 64)
}
