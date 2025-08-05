//go:build test || integration

package chatbot

import (
	"nudgebot-api/internal/config"
	"nudgebot-api/internal/events"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// NewChatbotServiceWithProvider creates a ChatbotService with a custom provider for testing
func NewChatbotServiceWithProvider(eventBus events.EventBus, logger *zap.Logger, provider TelegramProvider, cfg config.ChatbotConfig) (ChatbotService, error) {
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

	return service, nil
}

// StubTelegramProvider is a stub implementation of TelegramProvider for testing
type StubTelegramProvider struct {
	logger       *zap.Logger
	sentMessages []SentMessage
}

// SentMessage represents a message sent through the stub provider for verification
type SentMessage struct {
	ChatID   int64
	Text     string
	Keyboard *tgbotapi.InlineKeyboardMarkup
}

// NewStubTelegramProvider creates a new stub Telegram provider for testing
func NewStubTelegramProvider(logger *zap.Logger) TelegramProvider {
	return &StubTelegramProvider{
		logger:       logger,
		sentMessages: make([]SentMessage, 0),
	}
}

// SendMessage implements TelegramProvider interface (logs message but doesn't send)
func (s *StubTelegramProvider) SendMessage(chatID int64, text string) error {
	s.logger.Info("Stub Telegram provider sending message",
		zap.Int64("chat_id", chatID),
		zap.String("text", text))

	// Store the message for verification
	s.sentMessages = append(s.sentMessages, SentMessage{
		ChatID:   chatID,
		Text:     text,
		Keyboard: nil,
	})

	return nil
}

// SendMessageWithKeyboard implements TelegramProvider interface (logs message but doesn't send)
func (s *StubTelegramProvider) SendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	s.logger.Info("Stub Telegram provider sending message with keyboard",
		zap.Int64("chat_id", chatID),
		zap.String("text", text),
		zap.Int("keyboard_rows", len(keyboard.InlineKeyboard)))

	// Store the message for verification
	s.sentMessages = append(s.sentMessages, SentMessage{
		ChatID:   chatID,
		Text:     text,
		Keyboard: &keyboard,
	})

	return nil
}

// SetWebhook implements TelegramProvider interface (logs webhook URL but doesn't set)
func (s *StubTelegramProvider) SetWebhook(webhookURL string) error {
	s.logger.Info("Stub Telegram provider setting webhook",
		zap.String("webhook_url", webhookURL))
	return nil
}

// DeleteWebhook implements TelegramProvider interface (logs action but doesn't delete)
func (s *StubTelegramProvider) DeleteWebhook() error {
	s.logger.Info("Stub Telegram provider deleting webhook")
	return nil
}

// GetMe implements TelegramProvider interface with mock bot information
func (s *StubTelegramProvider) GetMe() (*tgbotapi.User, error) {
	s.logger.Info("Stub Telegram provider getting bot info")
	
	mockUser := &tgbotapi.User{
		ID:        123456789,
		IsBot:     true,
		FirstName: "NudgeBot",
		UserName:  "nudgebot_test",
	}

	return mockUser, nil
}

// GetSentMessages returns all messages sent through this stub provider (for test verification)
func (s *StubTelegramProvider) GetSentMessages() []SentMessage {
	return s.sentMessages
}

// ClearSentMessages clears the sent messages list (for test cleanup)
func (s *StubTelegramProvider) ClearSentMessages() {
	s.sentMessages = make([]SentMessage, 0)
}
