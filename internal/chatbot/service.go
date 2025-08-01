package chatbot

import (
	"nudgebot-api/internal/common"
	"nudgebot-api/internal/events"

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
	eventBus events.EventBus
	logger   *zap.Logger
}

// NewChatbotService creates a new instance of ChatbotService
func NewChatbotService(eventBus events.EventBus, logger *zap.Logger) ChatbotService {
	service := &chatbotService{
		eventBus: eventBus,
		logger:   logger,
	}

	// Subscribe to relevant events
	service.setupEventSubscriptions()

	return service
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
	s.logger.Info("Sending message",
		zap.String("chatID", string(chatID)),
		zap.String("text", text))

	// TODO: Implement actual Telegram Bot API call
	return nil
}

// SendMessageWithKeyboard sends a message with an inline keyboard to the specified chat
func (s *chatbotService) SendMessageWithKeyboard(chatID common.ChatID, text string, keyboard InlineKeyboard) error {
	s.logger.Info("Sending message with keyboard",
		zap.String("chatID", string(chatID)),
		zap.String("text", text),
		zap.Any("keyboard", keyboard))

	// TODO: Implement actual Telegram Bot API call with keyboard
	return nil
}

// HandleWebhook processes incoming webhook data from Telegram
func (s *chatbotService) HandleWebhook(webhookData []byte) error {
	s.logger.Info("Handling webhook", zap.Int("dataSize", len(webhookData)))

	// TODO: Parse webhook data and extract message information
	// For now, create a mock MessageReceived event
	event := events.MessageReceived{
		Event:       events.NewEvent(),
		UserID:      "mock_user_id",
		ChatID:      "mock_chat_id",
		MessageText: "mock message text",
	}

	return s.eventBus.Publish(events.TopicMessageReceived, event)
}

// ProcessCommand processes a specific command from a user
func (s *chatbotService) ProcessCommand(command Command, userID common.UserID, chatID common.ChatID) error {
	s.logger.Info("Processing command",
		zap.String("command", string(command)),
		zap.String("userID", string(userID)),
		zap.String("chatID", string(chatID)))

	// TODO: Implement command-specific logic
	switch command {
	case CommandStart:
		return s.SendMessage(common.ChatID(chatID), "Welcome! I'm your personal task nudge bot. Send me a task and I'll help you remember it!")
	case CommandHelp:
		return s.SendMessage(common.ChatID(chatID), "Available commands:\n/start - Start the bot\n/help - Show this help\n/list - List your tasks\n/done - Mark task as done\n/delete - Delete a task")
	case CommandList:
		return s.SendMessage(common.ChatID(chatID), "Listing your tasks... (not implemented yet)")
	case CommandDone:
		return s.SendMessage(common.ChatID(chatID), "Which task did you complete? (not implemented yet)")
	case CommandDelete:
		return s.SendMessage(common.ChatID(chatID), "Which task would you like to delete? (not implemented yet)")
	default:
		return s.SendMessage(common.ChatID(chatID), "Unknown command. Type /help for available commands.")
	}
}

// handleTaskParsed handles TaskParsed events from the LLM service
func (s *chatbotService) handleTaskParsed(event events.TaskParsed) {
	s.logger.Info("Handling TaskParsed event",
		zap.String("correlationID", event.CorrelationID),
		zap.String("userID", event.UserID),
		zap.String("taskTitle", event.ParsedTask.Title))

	// TODO: Send confirmation message to user about the parsed task
}

// handleReminderDue handles ReminderDue events from the nudge service
func (s *chatbotService) handleReminderDue(event events.ReminderDue) {
	s.logger.Info("Handling ReminderDue event",
		zap.String("correlationID", event.CorrelationID),
		zap.String("taskID", event.TaskID),
		zap.String("userID", event.UserID),
		zap.String("chatID", event.ChatID))

	// TODO: Send reminder message to user
}
