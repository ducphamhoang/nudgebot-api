package chatbot

import (
	"fmt"
	"time"

	"nudgebot-api/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// telegramProvider implements the TelegramProvider interface using the telegram-bot-api library
type telegramProvider struct {
	bot    *tgbotapi.BotAPI
	logger *zap.Logger
	config config.ChatbotConfig
}

// NewTelegramProvider creates a new TelegramProvider instance
func NewTelegramProvider(config config.ChatbotConfig, logger *zap.Logger) (TelegramProvider, error) {
	if config.Token == "" {
		return nil, fmt.Errorf("telegram bot token is required")
	}

	bot, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	// Note: The telegram-bot-api library doesn't expose timeout configuration directly
	// The timeout is handled internally by the HTTP client

	// Validate bot by getting bot info
	_, err = bot.GetMe()
	if err != nil {
		return nil, fmt.Errorf("failed to validate bot token: %w", err)
	}

	logger.Info("Telegram bot initialized successfully", zap.String("username", bot.Self.UserName))

	return &telegramProvider{
		bot:    bot,
		logger: logger,
		config: config,
	}, nil
}

// SendMessage sends a plain text message to the specified chat
func (p *telegramProvider) SendMessage(chatID int64, text string) error {
	correlationID := fmt.Sprintf("msg_%d_%d", chatID, time.Now().Unix())

	p.logger.Debug("Sending message",
		zap.String("correlation_id", correlationID),
		zap.Int64("chat_id", chatID),
		zap.Int("text_length", len(text)))

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML

	_, err := p.bot.Send(msg)
	if err != nil {
		p.logger.Error("Failed to send message",
			zap.String("correlation_id", correlationID),
			zap.Int64("chat_id", chatID),
			zap.Error(err))
		return fmt.Errorf("failed to send message: %w", err)
	}

	p.logger.Debug("Message sent successfully",
		zap.String("correlation_id", correlationID),
		zap.Int64("chat_id", chatID))

	return nil
}

// SendMessageWithKeyboard sends a message with an inline keyboard
func (p *telegramProvider) SendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	correlationID := fmt.Sprintf("kbd_%d_%d", chatID, time.Now().Unix())

	p.logger.Debug("Sending message with keyboard",
		zap.String("correlation_id", correlationID),
		zap.Int64("chat_id", chatID),
		zap.Int("text_length", len(text)),
		zap.Int("keyboard_rows", len(keyboard.InlineKeyboard)))

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard

	_, err := p.bot.Send(msg)
	if err != nil {
		p.logger.Error("Failed to send message with keyboard",
			zap.String("correlation_id", correlationID),
			zap.Int64("chat_id", chatID),
			zap.Error(err))
		return fmt.Errorf("failed to send message with keyboard: %w", err)
	}

	p.logger.Debug("Message with keyboard sent successfully",
		zap.String("correlation_id", correlationID),
		zap.Int64("chat_id", chatID))

	return nil
}

// SetWebhook configures the webhook URL for receiving updates
func (p *telegramProvider) SetWebhook(webhookURL string) error {
	p.logger.Info("Setting webhook", zap.String("webhook_url", webhookURL))

	webhookConfig, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		p.logger.Error("Failed to create webhook config",
			zap.String("webhook_url", webhookURL),
			zap.Error(err))
		return fmt.Errorf("failed to create webhook config: %w", err)
	}

	_, err = p.bot.Request(webhookConfig)
	if err != nil {
		p.logger.Error("Failed to set webhook",
			zap.String("webhook_url", webhookURL),
			zap.Error(err))
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	p.logger.Info("Webhook set successfully", zap.String("webhook_url", webhookURL))
	return nil
}

// DeleteWebhook removes the configured webhook
func (p *telegramProvider) DeleteWebhook() error {
	p.logger.Info("Deleting webhook")

	_, err := p.bot.Request(tgbotapi.DeleteWebhookConfig{})
	if err != nil {
		p.logger.Error("Failed to delete webhook", zap.Error(err))
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	p.logger.Info("Webhook deleted successfully")
	return nil
}

// GetMe returns information about the bot
func (p *telegramProvider) GetMe() (*tgbotapi.User, error) {
	p.logger.Debug("Getting bot information")

	me, err := p.bot.GetMe()
	if err != nil {
		p.logger.Error("Failed to get bot information", zap.Error(err))
		return nil, fmt.Errorf("failed to get bot information: %w", err)
	}

	p.logger.Debug("Bot information retrieved successfully",
		zap.String("username", me.UserName),
		zap.String("first_name", me.FirstName))

	return &me, nil
}
