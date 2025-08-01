package chatbot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramProvider defines the contract for Telegram API operations
type TelegramProvider interface {
	// SendMessage sends a plain text message to the specified chat
	SendMessage(chatID int64, text string) error

	// SendMessageWithKeyboard sends a message with an inline keyboard
	SendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error

	// SetWebhook configures the webhook URL for receiving updates
	SetWebhook(webhookURL string) error

	// DeleteWebhook removes the configured webhook
	DeleteWebhook() error

	// GetMe returns information about the bot
	GetMe() (*tgbotapi.User, error)
}

// TelegramConfig holds configuration for Telegram provider
type TelegramConfig struct {
	Token      string `json:"token" yaml:"token"`
	WebhookURL string `json:"webhook_url" yaml:"webhook_url"`
	Timeout    int    `json:"timeout" yaml:"timeout"`
	MaxRetries int    `json:"max_retries" yaml:"max_retries"`
	RetryDelay int    `json:"retry_delay" yaml:"retry_delay"`
}
