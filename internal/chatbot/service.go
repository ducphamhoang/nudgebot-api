package chatbot

import (
	"fmt"
	"strconv"
	"strings"

	"nudgebot-api/internal/common"
	"nudgebot-api/internal/config"
	"nudgebot-api/internal/events"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// ChatbotService defines the interface for chatbot operations
type ChatbotService interface {
	SendMessage(chatID common.ChatID, text string) error
	SendMessageWithKeyboard(chatID common.ChatID, text string, keyboard InlineKeyboard) error
	HandleWebhook(webhookData []byte) error
	ProcessCommand(command Command, userID common.UserID, chatID common.ChatID) error
}

// chatbotService implements the ChatbotService interface
type chatbotService struct {
	eventBus         events.EventBus
	logger           *zap.Logger
	provider         TelegramProvider
	parser           *WebhookParser
	keyboardBuilder  *KeyboardBuilder
	commandProcessor *CommandProcessor
	config           config.ChatbotConfig
}

// NewChatbotService creates a new instance of ChatbotService
func NewChatbotService(eventBus events.EventBus, logger *zap.Logger, cfg config.ChatbotConfig) (ChatbotService, error) {
	// Create Telegram provider
	provider, err := NewTelegramProvider(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram provider: %w", err)
	}

	service := &chatbotService{
		eventBus:         eventBus,
		logger:           logger,
		provider:         provider,
		parser:           NewWebhookParser(),
		keyboardBuilder:  NewKeyboardBuilder(),
		commandProcessor: NewCommandProcessor(eventBus, logger),
		config:           cfg,
	}

	// Subscribe to relevant events
	service.setupEventSubscriptions()

	// Setup webhook if configured
	if cfg.WebhookURL != "" {
		if err := provider.SetWebhook(cfg.WebhookURL); err != nil {
			logger.Warn("Failed to set webhook", zap.Error(err))
		}
	}

	return service, nil
}

// setupEventSubscriptions sets up event subscriptions for the chatbot service
func (s *chatbotService) setupEventSubscriptions() {
	// Subscribe to TaskParsed events
	err := s.eventBus.Subscribe(events.TopicTaskParsed, s.handleTaskParsed)
	if err != nil {
		s.logger.Error("Failed to subscribe to TaskParsed events", zap.Error(err))
	}

	// Subscribe to ReminderDue events
	err = s.eventBus.Subscribe(events.TopicReminderDue, s.handleReminderDue)
	if err != nil {
		s.logger.Error("Failed to subscribe to ReminderDue events", zap.Error(err))
	}

	// Subscribe to TaskListResponse events from the nudge service
	err = s.eventBus.Subscribe(events.TopicTaskListResponse, s.handleTaskListResponse)
	if err != nil {
		s.logger.Error("Failed to subscribe to TaskListResponse events", zap.Error(err))
	}

	// Subscribe to TaskActionResponse events from the nudge service
	err = s.eventBus.Subscribe(events.TopicTaskActionResponse, s.handleTaskActionResponse)
	if err != nil {
		s.logger.Error("Failed to subscribe to TaskActionResponse events", zap.Error(err))
	}

	// Subscribe to TaskCreated events for confirmation messages
	err = s.eventBus.Subscribe(events.TopicTaskCreated, s.handleTaskCreated)
	if err != nil {
		s.logger.Error("Failed to subscribe to TaskCreated events", zap.Error(err))
	}
}

// SendMessage sends a text message to the specified chat
func (s *chatbotService) SendMessage(chatID common.ChatID, text string) error {
	s.logger.Debug("Sending message",
		zap.String("chat_id", string(chatID)),
		zap.Int("text_length", len(text)))

	chatIDInt, err := strconv.ParseInt(string(chatID), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	return s.provider.SendMessage(chatIDInt, text)
}

// SendMessageWithKeyboard sends a message with an inline keyboard to the specified chat
func (s *chatbotService) SendMessageWithKeyboard(chatID common.ChatID, text string, keyboard InlineKeyboard) error {
	s.logger.Debug("Sending message with keyboard",
		zap.String("chat_id", string(chatID)),
		zap.Int("text_length", len(text)),
		zap.Int("keyboard_rows", len(keyboard.Buttons)))

	chatIDInt, err := strconv.ParseInt(string(chatID), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	// Convert domain keyboard to Telegram format
	tgKeyboard := s.keyboardBuilder.ConvertDomainKeyboard(keyboard)

	return s.provider.SendMessageWithKeyboard(chatIDInt, text, tgKeyboard)
}

// HandleWebhook processes incoming webhook data from Telegram
func (s *chatbotService) HandleWebhook(webhookData []byte) error {
	correlationID := fmt.Sprintf("webhook_%d", len(webhookData))
	s.logger.Debug("Handling webhook",
		zap.String("correlation_id", correlationID),
		zap.Int("data_size", len(webhookData)))

	// Parse the webhook update
	update, err := s.parser.ParseUpdate(webhookData)
	if err != nil {
		s.logger.Error("Failed to parse webhook update",
			zap.String("correlation_id", correlationID),
			zap.Error(err))
		return WrapParsingError(err, "telegram_update")
	}

	// Get correlation ID from update
	correlationID = s.parser.BuildCorrelationID(update)

	// Extract user and chat information
	userID, err := s.parser.GetUserID(update)
	if err != nil {
		s.logger.Error("Failed to extract user ID",
			zap.String("correlation_id", correlationID),
			zap.Error(err))
		return WrapParsingError(err, "user_id")
	}

	chatID, err := s.parser.GetChatID(update)
	if err != nil {
		s.logger.Error("Failed to extract chat ID",
			zap.String("correlation_id", correlationID),
			zap.Error(err))
		return WrapParsingError(err, "chat_id")
	}

	// Determine the type of update and handle accordingly
	messageType := s.parser.DetermineMessageType(update)

	switch messageType {
	case MessageTypeCommand:
		return s.handleCommand(update, string(userID), string(chatID), correlationID)
	case MessageTypeText:
		return s.handleTextMessage(update, string(userID), string(chatID), correlationID)
	case MessageTypeCallback:
		return s.handleCallbackQuery(update, string(userID), string(chatID), correlationID)
	default:
		s.logger.Warn("Unknown message type",
			zap.String("correlation_id", correlationID),
			zap.String("message_type", string(messageType)))
		return nil
	}
}

// handleCommand processes bot commands
func (s *chatbotService) handleCommand(update *tgbotapi.Update, userID, chatID, correlationID string) error {
	command, err := s.parser.ExtractCommand(update.Message)
	if err != nil {
		s.logger.Error("Failed to extract command",
			zap.String("correlation_id", correlationID),
			zap.Error(err))
		return err
	}

	s.logger.Info("Processing command",
		zap.String("correlation_id", correlationID),
		zap.String("command", string(command)),
		zap.String("user_id", userID),
		zap.String("chat_id", chatID))

	// Parse command arguments
	args := strings.Fields(update.Message.Text)
	if len(args) > 1 {
		args = args[1:] // Remove command itself
	} else {
		args = []string{}
	}

	var response string

	switch command {
	case CommandStart:
		response, err = s.commandProcessor.ProcessStartCommand(userID, chatID)
	case CommandHelp:
		response, err = s.commandProcessor.ProcessHelpCommand(userID, chatID)
	case CommandList:
		err = s.commandProcessor.ProcessListCommand(userID, chatID)
		return err // Response will be sent via event
	case CommandDone:
		response, err = s.commandProcessor.ProcessDoneCommand(userID, chatID, args)
	case CommandDelete:
		response, err = s.commandProcessor.ProcessDeleteCommand(userID, chatID, args)
	default:
		response = "Unknown command. Type /help for available commands."
	}

	if err != nil {
		s.logger.Error("Command processing failed",
			zap.String("correlation_id", correlationID),
			zap.String("command", string(command)),
			zap.Error(err))
		response = "Sorry, there was an error processing your command."
	}

	if response != "" {
		return s.SendMessage(common.ChatID(chatID), response)
	}

	return nil
}

// handleTextMessage processes regular text messages
func (s *chatbotService) handleTextMessage(update *tgbotapi.Update, userID, chatID, correlationID string) error {
	message, err := s.parser.ExtractMessage(update)
	if err != nil {
		s.logger.Error("Failed to extract message",
			zap.String("correlation_id", correlationID),
			zap.Error(err))
		return err
	}

	s.logger.Info("Processing text message",
		zap.String("correlation_id", correlationID),
		zap.String("user_id", userID),
		zap.String("chat_id", chatID),
		zap.Int("text_length", len(message.Text)))

	// Publish MessageReceived event for task parsing
	messageEvent := events.MessageReceived{
		Event:       events.NewEvent(),
		UserID:      userID,
		ChatID:      chatID,
		MessageText: message.Text,
	}

	return s.eventBus.Publish(events.TopicMessageReceived, messageEvent)
}

// handleCallbackQuery processes inline keyboard button presses
func (s *chatbotService) handleCallbackQuery(update *tgbotapi.Update, userID, chatID, correlationID string) error {
	callbackData, err := s.parser.ExtractCallbackQuery(update)
	if err != nil {
		s.logger.Error("Failed to extract callback query",
			zap.String("correlation_id", correlationID),
			zap.Error(err))
		return err
	}

	s.logger.Info("Processing callback query",
		zap.String("correlation_id", correlationID),
		zap.String("user_id", userID),
		zap.String("chat_id", chatID),
		zap.String("action", callbackData.Action))

	response, err := s.commandProcessor.HandleCallbackQuery(callbackData, userID, chatID)
	if err != nil {
		s.logger.Error("Callback query processing failed",
			zap.String("correlation_id", correlationID),
			zap.Error(err))
		response = "Sorry, there was an error processing your request."
	}

	if response != "" {
		return s.SendMessage(common.ChatID(chatID), response)
	}

	return nil
}

// ProcessCommand processes a specific command from a user
func (s *chatbotService) ProcessCommand(command Command, userID common.UserID, chatID common.ChatID) error {
	s.logger.Info("Processing command",
		zap.String("command", string(command)),
		zap.String("user_id", string(userID)),
		zap.String("chat_id", string(chatID)))

	var response string
	var err error

	switch command {
	case CommandStart:
		response, err = s.commandProcessor.ProcessStartCommand(string(userID), string(chatID))
	case CommandHelp:
		response, err = s.commandProcessor.ProcessHelpCommand(string(userID), string(chatID))
	case CommandList:
		return s.commandProcessor.ProcessListCommand(string(userID), string(chatID))
	case CommandDone:
		response, err = s.commandProcessor.ProcessDoneCommand(string(userID), string(chatID), []string{})
	case CommandDelete:
		response, err = s.commandProcessor.ProcessDeleteCommand(string(userID), string(chatID), []string{})
	default:
		response = "Unknown command. Type /help for available commands."
	}

	if err != nil {
		s.logger.Error("Command processing failed", zap.Error(err))
		response = "Sorry, there was an error processing your command."
	}

	return s.SendMessage(chatID, response)
}

// handleTaskParsed handles TaskParsed events from the LLM service
func (s *chatbotService) handleTaskParsed(event events.TaskParsed) {
	s.logger.Info("Handling TaskParsed event",
		zap.String("correlation_id", event.CorrelationID),
		zap.String("user_id", event.UserID),
		zap.String("task_title", event.ParsedTask.Title))

	// TaskCreated events will handle the confirmation message with proper task ID
	// This handler is kept for future processing needs or logging
	s.logger.Debug("Task parsing completed, waiting for TaskCreated event for confirmation")
}

// handleReminderDue handles ReminderDue events from the nudge service
func (s *chatbotService) handleReminderDue(event events.ReminderDue) {
	s.logger.Info("Handling ReminderDue event",
		zap.String("correlation_id", event.CorrelationID),
		zap.String("task_id", event.TaskID),
		zap.String("user_id", event.UserID),
		zap.String("chat_id", event.ChatID))

	// Create reminder message with task action keyboard
	reminderText := fmt.Sprintf("⏰ <b>Task Reminder!</b>\n\nYou have a task that needs attention.\n\nTask ID: %s", event.TaskID)

	// Create action keyboard for the task
	keyboard := s.keyboardBuilder.BuildTaskActionKeyboard(event.TaskID)

	// Convert to domain keyboard format
	domainKeyboard := InlineKeyboard{
		Buttons: make([][]InlineKeyboardButton, len(keyboard.InlineKeyboard)),
	}

	for i, row := range keyboard.InlineKeyboard {
		domainKeyboard.Buttons[i] = make([]InlineKeyboardButton, len(row))
		for j, button := range row {
			domainKeyboard.Buttons[i][j] = InlineKeyboardButton{
				Text:         button.Text,
				CallbackData: *button.CallbackData,
			}
		}
	}

	err := s.SendMessageWithKeyboard(common.ChatID(event.ChatID), reminderText, domainKeyboard)
	if err != nil {
		s.logger.Error("Failed to send reminder",
			zap.String("correlation_id", event.CorrelationID),
			zap.Error(err))
	}
}

// handleTaskListResponse handles TaskListResponse events from the nudge service
func (s *chatbotService) handleTaskListResponse(event events.TaskListResponse) {
	s.logger.Info("Handling TaskListResponse event",
		zap.String("correlation_id", event.CorrelationID),
		zap.String("user_id", event.UserID),
		zap.String("chat_id", event.ChatID),
		zap.Int("task_count", len(event.Tasks)),
		zap.Bool("success", event.Success))

	var messageText string

	// Handle error responses
	if !event.Success {
		s.logger.Warn("Received error in TaskListResponse",
			zap.String("error_code", event.ErrorCode),
			zap.String("error_message", event.ErrorMsg))

		messageText = s.formatTaskListErrorMessage(event.ErrorCode, event.ErrorMsg)

		// Send error message to user
		err := s.SendMessage(common.ChatID(event.ChatID), messageText)
		if err != nil {
			s.logger.Error("Failed to send task list error message",
				zap.String("correlation_id", event.CorrelationID),
				zap.Error(err))
		}
		return
	}

	// Handle successful responses
	if len(event.Tasks) == 0 {
		messageText = "📝 <b>Your Task List</b>\n\nYou have no active tasks. Great job! 🎉\n\nSend me a message to create a new task."
	} else {
		messageText = fmt.Sprintf("📝 <b>Your Task List</b>\n\nYou have %d active task(s):\n\n", len(event.Tasks))

		for i, task := range event.Tasks {
			taskNumber := i + 1
			priority := strings.ToUpper(string(task.Priority[:1])) + strings.ToLower(string(task.Priority[1:]))

			// Format task entry
			taskEntry := fmt.Sprintf("<b>%d.</b> %s\n   🏷 <i>%s Priority</i>", taskNumber, task.Title, priority)

			if task.Description != "" {
				taskEntry += fmt.Sprintf("\n   📝 %s", task.Description)
			}

			if task.DueDate != nil {
				dueText := task.DueDate.Format("Jan 2, 15:04")
				if task.IsOverdue {
					taskEntry += fmt.Sprintf("\n   ⏰ <b>OVERDUE:</b> %s", dueText)
				} else {
					taskEntry += fmt.Sprintf("\n   📅 Due: %s", dueText)
				}
			}

			messageText += taskEntry + "\n\n"
		}

		// Convert event tasks to keyboard task format
		keyboardTasks := make([]TaskSummary, len(event.Tasks))
		for i, task := range event.Tasks {
			keyboardTasks[i] = TaskSummary{
				ID:      common.TaskID(task.ID),
				Title:   task.Title,
				DueDate: task.DueDate,
				Status:  task.Status,
			}
		}

		// Create task list keyboard with actions for each task (simple pagination for now)
		currentPage := 0
		totalPages := 1
		keyboard := s.keyboardBuilder.BuildTaskListKeyboard(keyboardTasks, currentPage, totalPages)

		// Convert to domain keyboard format
		domainKeyboard := InlineKeyboard{
			Buttons: make([][]InlineKeyboardButton, len(keyboard.InlineKeyboard)),
		}

		for i, row := range keyboard.InlineKeyboard {
			domainKeyboard.Buttons[i] = make([]InlineKeyboardButton, len(row))
			for j, button := range row {
				domainKeyboard.Buttons[i][j] = InlineKeyboardButton{
					Text:         button.Text,
					CallbackData: *button.CallbackData,
				}
			}
		}

		// Send message with keyboard
		err := s.SendMessageWithKeyboard(common.ChatID(event.ChatID), messageText, domainKeyboard)
		if err != nil {
			s.logger.Error("Failed to send task list with keyboard",
				zap.String("correlation_id", event.CorrelationID),
				zap.Error(err))
		}
		return
	}

	// Send simple message without keyboard for empty list
	err := s.SendMessage(common.ChatID(event.ChatID), messageText)
	if err != nil {
		s.logger.Error("Failed to send task list message",
			zap.String("correlation_id", event.CorrelationID),
			zap.Error(err))
	}
}

// handleTaskActionResponse handles TaskActionResponse events from the nudge service
func (s *chatbotService) handleTaskActionResponse(event events.TaskActionResponse) {
	s.logger.Info("Handling TaskActionResponse event",
		zap.String("correlation_id", event.CorrelationID),
		zap.String("user_id", event.UserID),
		zap.String("chat_id", event.ChatID),
		zap.String("task_id", event.TaskID),
		zap.String("action", event.Action),
		zap.Bool("success", event.Success))

	var messageText string
	var emoji string

	// Set emoji and message based on action and success
	if event.Success {
		switch event.Action {
		case "done", "complete":
			emoji = "✅"
			messageText = fmt.Sprintf("%s <b>Task Completed!</b>\n\n%s", emoji, event.Message)
		case "delete":
			emoji = "🗑️"
			messageText = fmt.Sprintf("%s <b>Task Deleted!</b>\n\n%s", emoji, event.Message)
		case "snooze":
			emoji = "😴"
			messageText = fmt.Sprintf("%s <b>Task Snoozed!</b>\n\n%s", emoji, event.Message)
		default:
			emoji = "✅"
			messageText = fmt.Sprintf("%s <b>Action Completed!</b>\n\n%s", emoji, event.Message)
		}
	} else {
		emoji = "❌"
		messageText = fmt.Sprintf("%s <b>Action Failed</b>\n\n%s", emoji, event.Message)
	}

	err := s.SendMessage(common.ChatID(event.ChatID), messageText)
	if err != nil {
		s.logger.Error("Failed to send task action response",
			zap.String("correlation_id", event.CorrelationID),
			zap.Error(err))
	}
}

// handleTaskCreated handles TaskCreated events from the nudge service
func (s *chatbotService) handleTaskCreated(event events.TaskCreated) {
	s.logger.Info("Handling TaskCreated event",
		zap.String("correlation_id", event.CorrelationID),
		zap.String("task_id", event.TaskID),
		zap.String("user_id", event.UserID),
		zap.String("task_title", event.Title))

	// Create confirmation message with task details
	confirmText := fmt.Sprintf("📋 <b>Task Created!</b>\n\n<b>Title:</b> %s\n<b>Priority:</b> %s",
		event.Title,
		event.Priority)

	if event.DueDate != nil {
		confirmText += fmt.Sprintf("\n<b>Due:</b> %s", event.DueDate.Format("Jan 2, 2006 at 3:04 PM"))
	}

	confirmText += fmt.Sprintf("\n<b>Created:</b> %s", event.CreatedAt.Format("Jan 2, 15:04"))

	// Create action keyboard for immediate task actions
	keyboard := s.keyboardBuilder.BuildTaskActionKeyboard(event.TaskID)

	// Convert to domain keyboard format
	domainKeyboard := InlineKeyboard{
		Buttons: make([][]InlineKeyboardButton, len(keyboard.InlineKeyboard)),
	}

	for i, row := range keyboard.InlineKeyboard {
		domainKeyboard.Buttons[i] = make([]InlineKeyboardButton, len(row))
		for j, button := range row {
			domainKeyboard.Buttons[i][j] = InlineKeyboardButton{
				Text:         button.Text,
				CallbackData: *button.CallbackData,
			}
		}
	}

	// Determine chat ID from user ID (for now they're the same in Telegram)
	chatID := event.UserID

	err := s.SendMessageWithKeyboard(common.ChatID(chatID), confirmText, domainKeyboard)
	if err != nil {
		s.logger.Error("Failed to send task creation confirmation",
			zap.String("correlation_id", event.CorrelationID),
			zap.Error(err))
	}
}

// formatTaskListErrorMessage creates user-friendly error messages based on error codes
func (s *chatbotService) formatTaskListErrorMessage(errorCode, errorMsg string) string {
	switch errorCode {
	case "VALIDATION_FAILED":
		return "❌ <b>Invalid Request</b>\n\nThere was an issue with your request. Please try again."

	case "UNAUTHORIZED":
		return "🔒 <b>Access Denied</b>\n\nYou don't have permission to view these tasks."

	case "USER_NOT_FOUND":
		return "👤 <b>User Not Found</b>\n\nCould not find your user account. Please try signing in again."

	case "REPOSITORY_ERROR":
		return "🔧 <b>System Temporarily Unavailable</b>\n\nWe're experiencing technical difficulties. Please try again in a few moments."

	case "TASK_LIST_FAILED":
		return "📝 <b>Unable to Retrieve Tasks</b>\n\nSorry, we couldn't get your task list right now. Please try again."

	default:
		// Generic error message for unknown error codes
		if errorMsg != "" {
			return fmt.Sprintf("⚠️ <b>Something went wrong</b>\n\nError: %s\n\nPlease try again.", errorMsg)
		}
		return "⚠️ <b>Something went wrong</b>\n\nPlease try again."
	}
}
