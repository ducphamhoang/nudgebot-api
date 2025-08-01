package chatbot

import (
	"time"

	"nudgebot-api/internal/common"
)

// MessageType represents the type of message received
type MessageType string

const (
	MessageTypeCommand  MessageType = "command"
	MessageTypeText     MessageType = "text"
	MessageTypeCallback MessageType = "callback"
)

// Message represents a message from a user
type Message struct {
	ID          common.ID     `json:"id" validate:"required"`
	UserID      common.UserID `json:"user_id" validate:"required"`
	ChatID      common.ChatID `json:"chat_id" validate:"required"`
	Text        string        `json:"text" validate:"required"`
	Timestamp   time.Time     `json:"timestamp" validate:"required"`
	MessageType MessageType   `json:"message_type" validate:"required"`
}

// InlineKeyboard represents a Telegram inline keyboard
type InlineKeyboard struct {
	Buttons [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// InlineKeyboardButton represents a single button in an inline keyboard
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

// ChatSession represents the current state of a user's conversation
type ChatSession struct {
	UserID       common.UserID `json:"user_id" validate:"required"`
	ChatID       common.ChatID `json:"chat_id" validate:"required"`
	State        SessionState  `json:"state"`
	Context      string        `json:"context"`
	LastActivity time.Time     `json:"last_activity"`
}

// SessionState represents the current state of a chat session
type SessionState string

const (
	SessionStateIdle           SessionState = "idle"
	SessionStateAwaitingTask   SessionState = "awaiting_task"
	SessionStateConfirmingTask SessionState = "confirming_task"
	SessionStateManagingTasks  SessionState = "managing_tasks"
)

// Command represents supported bot commands
type Command string

const (
	CommandStart  Command = "/start"
	CommandHelp   Command = "/help"
	CommandList   Command = "/list"
	CommandDone   Command = "/done"
	CommandDelete Command = "/delete"
)

// CallbackData represents data from inline keyboard callbacks
type CallbackData struct {
	Action string            `json:"action"`
	Data   map[string]string `json:"data"`
}

// IsValid checks if the message type is valid
func (mt MessageType) IsValid() bool {
	switch mt {
	case MessageTypeCommand, MessageTypeText, MessageTypeCallback:
		return true
	default:
		return false
	}
}

// IsValid checks if the session state is valid
func (ss SessionState) IsValid() bool {
	switch ss {
	case SessionStateIdle, SessionStateAwaitingTask, SessionStateConfirmingTask, SessionStateManagingTasks:
		return true
	default:
		return false
	}
}

// IsValid checks if the command is valid
func (c Command) IsValid() bool {
	switch c {
	case CommandStart, CommandHelp, CommandList, CommandDone, CommandDelete:
		return true
	default:
		return false
	}
}
