package chatbot

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"nudgebot-api/internal/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// WebhookParser provides utilities for parsing Telegram webhook updates
type WebhookParser struct{}

// NewWebhookParser creates a new WebhookParser instance
func NewWebhookParser() *WebhookParser {
	return &WebhookParser{}
}

// ParseUpdate unmarshals webhook data into a Telegram Update struct
func (p *WebhookParser) ParseUpdate(updateData []byte) (*tgbotapi.Update, error) {
	if len(updateData) == 0 {
		return nil, fmt.Errorf("empty update data")
	}

	var update tgbotapi.Update
	if err := json.Unmarshal(updateData, &update); err != nil {
		return nil, fmt.Errorf("failed to unmarshal update data: %w", err)
	}

	// Basic validation
	if update.UpdateID == 0 {
		return nil, fmt.Errorf("invalid update: missing update ID")
	}

	return &update, nil
}

// telegramIDToUUID converts a Telegram numeric ID to a deterministic UUID
func telegramIDToUUID(telegramID int64) string {
	// Create a deterministic UUID based on the Telegram ID
	hash := md5.Sum([]byte(fmt.Sprintf("telegram_id_%d", telegramID)))
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		hash[0:4], hash[4:6], hash[6:8], hash[8:10], hash[10:16])
}

// ExtractMessage converts a Telegram message to domain Message struct
func (p *WebhookParser) ExtractMessage(update *tgbotapi.Update) (*Message, error) {
	if update == nil {
		return nil, fmt.Errorf("update is nil")
	}

	if update.Message == nil {
		return nil, fmt.Errorf("update does not contain a message")
	}

	msg := update.Message

	// Validate required fields
	if msg.From == nil {
		return nil, fmt.Errorf("message does not contain sender information")
	}

	if msg.Chat == nil {
		return nil, fmt.Errorf("message does not contain chat information")
	}

	text := msg.Text
	if text == "" && msg.Caption != "" {
		text = msg.Caption // Use caption for media messages
	}

	messageType := p.DetermineMessageType(update)

	return &Message{
		ID:          common.ID(strconv.Itoa(msg.MessageID)),
		UserID:      common.UserID(telegramIDToUUID(msg.From.ID)),
		ChatID:      common.ChatID(telegramIDToUUID(msg.Chat.ID)),
		Text:        text,
		Timestamp:   time.Unix(int64(msg.Date), 0),
		MessageType: messageType,
	}, nil
}

// ExtractCallbackQuery parses inline keyboard callback data
func (p *WebhookParser) ExtractCallbackQuery(update *tgbotapi.Update) (*CallbackData, error) {
	if update == nil {
		return nil, fmt.Errorf("update is nil")
	}

	if update.CallbackQuery == nil {
		return nil, fmt.Errorf("update does not contain a callback query")
	}

	callbackQuery := update.CallbackQuery

	if callbackQuery.Data == "" {
		return nil, fmt.Errorf("callback query does not contain data")
	}

	// Try to parse as JSON first
	var callbackData CallbackData
	if err := json.Unmarshal([]byte(callbackQuery.Data), &callbackData); err == nil {
		return &callbackData, nil
	}

	// Fallback to simple string format
	return &CallbackData{
		Action: callbackQuery.Data,
		Data:   make(map[string]string),
	}, nil
}

// DetermineMessageType classifies the message type
func (p *WebhookParser) DetermineMessageType(update *tgbotapi.Update) MessageType {
	if update.CallbackQuery != nil {
		return MessageTypeCallback
	}

	if update.Message != nil && update.Message.IsCommand() {
		return MessageTypeCommand
	}

	return MessageTypeText
}

// ExtractCommand parses bot commands from messages
func (p *WebhookParser) ExtractCommand(message *tgbotapi.Message) (Command, error) {
	if message == nil {
		return "", fmt.Errorf("message is nil")
	}

	if !message.IsCommand() {
		return "", fmt.Errorf("message is not a command")
	}

	commandText := message.Command()
	switch commandText {
	case "start":
		return CommandStart, nil
	case "help":
		return CommandHelp, nil
	case "list":
		return CommandList, nil
	case "done":
		return CommandDone, nil
	case "delete":
		return CommandDelete, nil
	default:
		return "", fmt.Errorf("unknown command: %s", commandText)
	}
}

// BuildCorrelationID generates a unique correlation ID for tracking
func (p *WebhookParser) BuildCorrelationID(update *tgbotapi.Update) string {
	if update == nil {
		return fmt.Sprintf("corr_%d", time.Now().UnixNano())
	}

	updateID := update.UpdateID
	timestamp := time.Now().Unix()

	if update.Message != nil {
		return fmt.Sprintf("msg_%d_%d_%d", updateID, update.Message.MessageID, timestamp)
	}

	if update.CallbackQuery != nil {
		return fmt.Sprintf("cb_%d_%s_%d", updateID, update.CallbackQuery.ID, timestamp)
	}

	return fmt.Sprintf("upd_%d_%d", updateID, timestamp)
}

// GetUserID extracts user ID from update
func (p *WebhookParser) GetUserID(update *tgbotapi.Update) (common.UserID, error) {
	if update == nil {
		return "", fmt.Errorf("update is nil")
	}

	var userID int64

	if update.Message != nil && update.Message.From != nil {
		userID = update.Message.From.ID
	} else if update.CallbackQuery != nil && update.CallbackQuery.From != nil {
		userID = update.CallbackQuery.From.ID
	} else {
		return "", fmt.Errorf("no user information found in update")
	}

	return common.UserID(telegramIDToUUID(userID)), nil
}

// GetChatID extracts chat ID from update
func (p *WebhookParser) GetChatID(update *tgbotapi.Update) (common.ChatID, error) {
	if update == nil {
		return "", fmt.Errorf("update is nil")
	}

	var chatID int64

	if update.Message != nil && update.Message.Chat != nil {
		chatID = update.Message.Chat.ID
	} else if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
		chatID = update.CallbackQuery.Message.Chat.ID
	} else {
		return "", fmt.Errorf("no chat information found in update")
	}

	return common.ChatID(telegramIDToUUID(chatID)), nil
}
