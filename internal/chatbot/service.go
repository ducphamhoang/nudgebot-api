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

	// Create confirmation message with task action keyboard
	confirmText := fmt.Sprintf("📋 <b>Task Created!</b>\n\n<b>Title:</b> %s\n<b>Priority:</b> %s",
		event.ParsedTask.Title,
		event.ParsedTask.Priority)

	if event.ParsedTask.Description != "" {
		confirmText += fmt.Sprintf("\n<b>Description:</b> %s", event.ParsedTask.Description)
	}

	if event.ParsedTask.DueDate != nil {
		confirmText += fmt.Sprintf("\n<b>Due:</b> %s", event.ParsedTask.DueDate.Format("Jan 2, 2006 at 3:04 PM"))
	}

	if len(event.ParsedTask.Tags) > 0 {
		confirmText += fmt.Sprintf("\n<b>Tags:</b> %s", strings.Join(event.ParsedTask.Tags, ", "))
	}

	// For now, we don't have a task ID, so we'll send without keyboard
	err := s.SendMessage(common.ChatID(event.UserID), confirmText)
	if err != nil {
		s.logger.Error("Failed to send task confirmation",
			zap.String("correlation_id", event.CorrelationID),
			zap.Error(err))
	}
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
