package mocks

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"nudgebot-api/internal/chatbot"
	"nudgebot-api/internal/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//go:generate mockgen -source=../chatbot/provider.go -destination=chatbot_provider_mocks.go -package=mocks

// MockTelegramProvider implements the TelegramProvider interface for testing
type MockTelegramProvider struct {
	mutex              sync.RWMutex
	sentMessages       []MockMessage
	sentKeyboards      []MockKeyboardMessage
	webhookURL         string
	botInfo            *tgbotapi.User
	sendMessageError   error
	sendKeyboardError  error
	setWebhookError    error
	deleteWebhookError error
	getMeError         error
	rateLimitDelay     time.Duration
	callCounts         map[string]int
}

// MockMessage represents a sent message for testing verification
type MockMessage struct {
	ChatID      int64
	Text        string
	Timestamp   time.Time
	MessageID   int
	ParseMode   string
	ReplyMarkup interface{}
}

// MockKeyboardMessage represents a sent message with keyboard
type MockKeyboardMessage struct {
	ChatID    int64
	Text      string
	Keyboard  tgbotapi.InlineKeyboardMarkup
	Timestamp time.Time
	MessageID int
}

// NewMockTelegramProvider creates a new mock Telegram provider
func NewMockTelegramProvider() *MockTelegramProvider {
	return &MockTelegramProvider{
		sentMessages:  make([]MockMessage, 0),
		sentKeyboards: make([]MockKeyboardMessage, 0),
		botInfo: &tgbotapi.User{
			ID:        123456789,
			UserName:  "mock_bot",
			FirstName: "Mock Bot",
			IsBot:     true,
		},
		callCounts: make(map[string]int),
	}
}

// SendMessage implements the TelegramProvider interface
func (m *MockTelegramProvider) SendMessage(chatID int64, text string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.callCounts["SendMessage"]++

	if m.sendMessageError != nil {
		return m.sendMessageError
	}

	if m.rateLimitDelay > 0 {
		time.Sleep(m.rateLimitDelay)
	}

	message := MockMessage{
		ChatID:    chatID,
		Text:      text,
		Timestamp: time.Now(),
		MessageID: len(m.sentMessages) + 1,
		ParseMode: "HTML",
	}

	m.sentMessages = append(m.sentMessages, message)
	return nil
}

// SendMessageWithKeyboard implements the TelegramProvider interface
func (m *MockTelegramProvider) SendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.callCounts["SendMessageWithKeyboard"]++

	if m.sendKeyboardError != nil {
		return m.sendKeyboardError
	}

	if m.rateLimitDelay > 0 {
		time.Sleep(m.rateLimitDelay)
	}

	keyboardMessage := MockKeyboardMessage{
		ChatID:    chatID,
		Text:      text,
		Keyboard:  keyboard,
		Timestamp: time.Now(),
		MessageID: len(m.sentKeyboards) + 1,
	}

	m.sentKeyboards = append(m.sentKeyboards, keyboardMessage)

	// Also add to regular messages for unified tracking
	message := MockMessage{
		ChatID:      chatID,
		Text:        text,
		Timestamp:   time.Now(),
		MessageID:   len(m.sentMessages) + 1,
		ParseMode:   "HTML",
		ReplyMarkup: keyboard,
	}
	m.sentMessages = append(m.sentMessages, message)

	return nil
}

// SetWebhook implements the TelegramProvider interface
func (m *MockTelegramProvider) SetWebhook(webhookURL string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.callCounts["SetWebhook"]++

	if m.setWebhookError != nil {
		return m.setWebhookError
	}

	m.webhookURL = webhookURL
	return nil
}

// DeleteWebhook implements the TelegramProvider interface
func (m *MockTelegramProvider) DeleteWebhook() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.callCounts["DeleteWebhook"]++

	if m.deleteWebhookError != nil {
		return m.deleteWebhookError
	}

	m.webhookURL = ""
	return nil
}

// GetMe implements the TelegramProvider interface
func (m *MockTelegramProvider) GetMe() (*tgbotapi.User, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	m.callCounts["GetMe"]++

	if m.getMeError != nil {
		return nil, m.getMeError
	}

	return m.botInfo, nil
}

// Test helper methods

// GetSentMessages returns all sent messages
func (m *MockTelegramProvider) GetSentMessages() []MockMessage {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy to prevent race conditions
	messages := make([]MockMessage, len(m.sentMessages))
	copy(messages, m.sentMessages)
	return messages
}

// GetSentKeyboards returns all sent keyboard messages
func (m *MockTelegramProvider) GetSentKeyboards() []MockKeyboardMessage {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	keyboards := make([]MockKeyboardMessage, len(m.sentKeyboards))
	copy(keyboards, m.sentKeyboards)
	return keyboards
}

// GetLastMessage returns the last sent message
func (m *MockTelegramProvider) GetLastMessage() *MockMessage {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if len(m.sentMessages) == 0 {
		return nil
	}
	return &m.sentMessages[len(m.sentMessages)-1]
}

// GetLastKeyboard returns the last sent keyboard message
func (m *MockTelegramProvider) GetLastKeyboard() *MockKeyboardMessage {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if len(m.sentKeyboards) == 0 {
		return nil
	}
	return &m.sentKeyboards[len(m.sentKeyboards)-1]
}

// GetMessagesForChat returns all messages sent to a specific chat
func (m *MockTelegramProvider) GetMessagesForChat(chatID int64) []MockMessage {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var chatMessages []MockMessage
	for _, msg := range m.sentMessages {
		if msg.ChatID == chatID {
			chatMessages = append(chatMessages, msg)
		}
	}
	return chatMessages
}

// GetWebhookURL returns the currently set webhook URL
func (m *MockTelegramProvider) GetWebhookURL() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.webhookURL
}

// GetCallCount returns the number of times a method was called
func (m *MockTelegramProvider) GetCallCount(method string) int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.callCounts[method]
}

// Configuration methods for testing different scenarios

// SetSendMessageError configures the provider to return an error on SendMessage
func (m *MockTelegramProvider) SetSendMessageError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.sendMessageError = err
}

// SetSendKeyboardError configures the provider to return an error on SendMessageWithKeyboard
func (m *MockTelegramProvider) SetSendKeyboardError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.sendKeyboardError = err
}

// SetWebhookError configures the provider to return an error on SetWebhook
func (m *MockTelegramProvider) SetWebhookError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.setWebhookError = err
}

// SetDeleteWebhookError configures the provider to return an error on DeleteWebhook
func (m *MockTelegramProvider) SetDeleteWebhookError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.deleteWebhookError = err
}

// SetGetMeError configures the provider to return an error on GetMe
func (m *MockTelegramProvider) SetGetMeError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.getMeError = err
}

// SetBotInfo configures the bot information returned by GetMe
func (m *MockTelegramProvider) SetBotInfo(botInfo *tgbotapi.User) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.botInfo = botInfo
}

// SetRateLimitDelay simulates rate limiting by adding delays
func (m *MockTelegramProvider) SetRateLimitDelay(delay time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.rateLimitDelay = delay
}

// ClearHistory clears all sent messages and call counts
func (m *MockTelegramProvider) ClearHistory() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.sentMessages = make([]MockMessage, 0)
	m.sentKeyboards = make([]MockKeyboardMessage, 0)
	m.callCounts = make(map[string]int)
}

// Factory methods for creating test scenarios

// CreateTestMessage creates a test message for verification
func CreateTestMessage(userID, chatID int64, text string) MockMessage {
	return MockMessage{
		ChatID:    chatID,
		Text:      text,
		Timestamp: time.Now(),
		MessageID: 1,
		ParseMode: "HTML",
	}
}

// CreateTestCallback creates test callback query data
func CreateTestCallback(userID, chatID int64, action string, data map[string]string) string {
	callbackData := chatbot.CallbackData{
		Action: action,
		Data:   data,
	}

	jsonData, _ := json.Marshal(callbackData)
	return string(jsonData)
}

// CreateTestCommand creates a test command for processing
func CreateTestCommand(command, userID, chatID string) (chatbot.Command, common.UserID, common.ChatID) {
	return chatbot.Command(command), common.UserID(userID), common.ChatID(chatID)
}

// MockWebhookParser for testing webhook processing without actual Telegram updates
type MockWebhookParser struct {
	parseUpdateError     error
	extractMessageError  error
	extractCallbackError error
	mockUpdate           *tgbotapi.Update
	mockMessage          *chatbot.Message
	mockCallbackData     *chatbot.CallbackData
	mockCorrelationID    string
	mockUserID           common.UserID
	mockChatID           common.ChatID
	mockMessageType      chatbot.MessageType
}

// NewMockWebhookParser creates a new mock webhook parser
func NewMockWebhookParser() *MockWebhookParser {
	return &MockWebhookParser{
		mockCorrelationID: "test_correlation_id",
		mockUserID:        "test_user_123",
		mockChatID:        "test_chat_456",
		mockMessageType:   chatbot.MessageTypeText,
	}
}

// ParseUpdate simulates parsing webhook data
func (m *MockWebhookParser) ParseUpdate(updateData []byte) (*tgbotapi.Update, error) {
	if m.parseUpdateError != nil {
		return nil, m.parseUpdateError
	}

	if m.mockUpdate != nil {
		return m.mockUpdate, nil
	}

	// Return a default test update
	return &tgbotapi.Update{
		UpdateID: 123456,
		Message: &tgbotapi.Message{
			MessageID: 1,
			From: &tgbotapi.User{
				ID:        123,
				UserName:  "testuser",
				FirstName: "Test",
			},
			Chat: &tgbotapi.Chat{
				ID:   456,
				Type: "private",
			},
			Text: "test message",
			Date: int(time.Now().Unix()),
		},
	}, nil
}

// SetParseUpdateError configures the parser to return an error
func (m *MockWebhookParser) SetParseUpdateError(err error) {
	m.parseUpdateError = err
}

// SetMockUpdate configures the parser to return a specific update
func (m *MockWebhookParser) SetMockUpdate(update *tgbotapi.Update) {
	m.mockUpdate = update
}

// MockCommandProcessor for testing command logic in isolation
type MockCommandProcessor struct {
	processStartResponse  string
	processHelpResponse   string
	processDoneResponse   string
	processDeleteResponse string
	callbackResponse      string
	processStartError     error
	processHelpError      error
	processListError      error
	processDoneError      error
	processDeleteError    error
	callbackError         error
	commandCalls          []MockCommandCall
}

// MockCommandCall represents a command processing call for verification
type MockCommandCall struct {
	Command   string
	UserID    string
	ChatID    string
	Args      []string
	Timestamp time.Time
}

// NewMockCommandProcessor creates a new mock command processor
func NewMockCommandProcessor() *MockCommandProcessor {
	return &MockCommandProcessor{
		processStartResponse:  "Welcome to the test bot!",
		processHelpResponse:   "Test help message",
		processDoneResponse:   "Task marked as done",
		processDeleteResponse: "Task deleted",
		callbackResponse:      "Callback processed",
		commandCalls:          make([]MockCommandCall, 0),
	}
}

// ProcessStartCommand simulates start command processing
func (m *MockCommandProcessor) ProcessStartCommand(userID, chatID string) (string, error) {
	m.commandCalls = append(m.commandCalls, MockCommandCall{
		Command:   "/start",
		UserID:    userID,
		ChatID:    chatID,
		Timestamp: time.Now(),
	})

	if m.processStartError != nil {
		return "", m.processStartError
	}

	return m.processStartResponse, nil
}

// ProcessHelpCommand simulates help command processing
func (m *MockCommandProcessor) ProcessHelpCommand(userID, chatID string) (string, error) {
	m.commandCalls = append(m.commandCalls, MockCommandCall{
		Command:   "/help",
		UserID:    userID,
		ChatID:    chatID,
		Timestamp: time.Now(),
	})

	if m.processHelpError != nil {
		return "", m.processHelpError
	}

	return m.processHelpResponse, nil
}

// GetCommandCalls returns all recorded command calls
func (m *MockCommandProcessor) GetCommandCalls() []MockCommandCall {
	return m.commandCalls
}

// GetLastCommandCall returns the last recorded command call
func (m *MockCommandProcessor) GetLastCommandCall() *MockCommandCall {
	if len(m.commandCalls) == 0 {
		return nil
	}
	return &m.commandCalls[len(m.commandCalls)-1]
}

// SetProcessStartResponse configures the response for start command
func (m *MockCommandProcessor) SetProcessStartResponse(response string) {
	m.processStartResponse = response
}

// SetProcessStartError configures the processor to return an error for start command
func (m *MockCommandProcessor) SetProcessStartError(err error) {
	m.processStartError = err
}

// ClearCommandHistory clears all recorded command calls
func (m *MockCommandProcessor) ClearCommandHistory() {
	m.commandCalls = make([]MockCommandCall, 0)
}

// SimulateWebhookUpdate creates test webhook data for testing
func SimulateWebhookUpdate(updateType string, userID, chatID int64, text string) []byte {
	var update tgbotapi.Update

	switch updateType {
	case "message":
		update = tgbotapi.Update{
			UpdateID: 123456,
			Message: &tgbotapi.Message{
				MessageID: 1,
				From: &tgbotapi.User{
					ID:        userID,
					UserName:  fmt.Sprintf("user_%d", userID),
					FirstName: "Test User",
				},
				Chat: &tgbotapi.Chat{
					ID:   chatID,
					Type: "private",
				},
				Text: text,
				Date: int(time.Now().Unix()),
			},
		}
	case "callback":
		update = tgbotapi.Update{
			UpdateID: 123456,
			CallbackQuery: &tgbotapi.CallbackQuery{
				ID: "callback_123",
				From: &tgbotapi.User{
					ID:        userID,
					UserName:  fmt.Sprintf("user_%d", userID),
					FirstName: "Test User",
				},
				Message: &tgbotapi.Message{
					MessageID: 1,
					Chat: &tgbotapi.Chat{
						ID:   chatID,
						Type: "private",
					},
				},
				Data: text, // callback data
			},
		}
	}

	jsonData, _ := json.Marshal(update)
	return jsonData
}

// ExpectCommand configures expected command processing behavior
func (m *MockCommandProcessor) ExpectCommand(command, userID, chatID string, response string, err error) {
	// This would be used in more sophisticated mock setups
	// For now, we use the simpler setter methods above
}
