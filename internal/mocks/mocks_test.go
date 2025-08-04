package mocks

import (
	"testing"

	"nudgebot-api/internal/chatbot"
	"nudgebot-api/internal/common"
	"nudgebot-api/internal/events"
	"nudgebot-api/internal/nudge"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestChatbotMocks_Basic(t *testing.T) {
	mockChatbot := &MockChatbotService{}

	// Test SendMessage mock
	mockChatbot.On("SendMessage", mock.Anything, mock.Anything).Return(nil)

	err := mockChatbot.SendMessage(common.ChatID("123"), "test message")
	assert.NoError(t, err)

	mockChatbot.AssertExpectations(t)
}

func TestChatbotMocks_SendMessageWithKeyboard(t *testing.T) {
	mockChatbot := &MockChatbotService{}

	keyboard := chatbot.InlineKeyboard{
		Buttons: [][]chatbot.InlineButton{
			{{Text: "Button 1", CallbackData: "btn1"}},
		},
	}

	mockChatbot.On("SendMessageWithKeyboard", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := mockChatbot.SendMessageWithKeyboard(common.ChatID("123"), "test message", keyboard)
	assert.NoError(t, err)

	mockChatbot.AssertExpectations(t)
}

func TestChatbotMocks_HandleWebhook(t *testing.T) {
	mockChatbot := &MockChatbotService{}

	webhookData := []byte(`{"message": {"text": "test"}}`)

	mockChatbot.On("HandleWebhook", mock.Anything).Return(nil)

	err := mockChatbot.HandleWebhook(webhookData)
	assert.NoError(t, err)

	mockChatbot.AssertExpectations(t)
}

func TestChatbotMocks_ProcessCommand(t *testing.T) {
	mockChatbot := &MockChatbotService{}

	command := chatbot.Command{
		Type: chatbot.CommandStart,
		Args: []string{},
	}

	mockChatbot.On("ProcessCommand", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := mockChatbot.ProcessCommand(command, common.UserID("user1"), common.ChatID("chat1"))
	assert.NoError(t, err)

	mockChatbot.AssertExpectations(t)
}

func TestEventBusMocks_PublishSubscribe(t *testing.T) {
	mockBus := &MockEventBus{}

	// Test Publish
	event := events.Event{
		Type:    events.TaskCreatedEvent,
		Payload: map[string]interface{}{"task_id": "123"},
	}

	mockBus.On("Publish", mock.Anything, mock.Anything).Return(nil)

	err := mockBus.Publish("task.created", event)
	assert.NoError(t, err)

	// Test Subscribe
	handler := func(event events.Event) error {
		return nil
	}

	mockBus.On("Subscribe", mock.Anything, mock.Anything).Return(nil)

	err = mockBus.Subscribe("task.created", handler)
	assert.NoError(t, err)

	mockBus.AssertExpectations(t)
}

func TestEventBusMocks_Unsubscribe(t *testing.T) {
	mockBus := &MockEventBus{}

	mockBus.On("Unsubscribe", mock.Anything, mock.Anything).Return(nil)

	err := mockBus.Unsubscribe("task.created", nil)
	assert.NoError(t, err)

	mockBus.AssertExpectations(t)
}

func TestEventBusMocks_Close(t *testing.T) {
	mockBus := &MockEventBus{}

	mockBus.On("Close").Return(nil)

	err := mockBus.Close()
	assert.NoError(t, err)

	mockBus.AssertExpectations(t)
}

func TestNudgeRepositoryMocks_CRUD(t *testing.T) {
	mockRepo := &MockNudgeRepository{}

	task := &nudge.Task{
		ID:          common.TaskID("task1"),
		UserID:      common.UserID("user1"),
		Description: "Test task",
		Status:      nudge.StatusPending,
	}

	// Test Create
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := mockRepo.Create(nil, task)
	assert.NoError(t, err)

	// Test GetByID
	mockRepo.On("GetByID", mock.Anything, mock.Anything).Return(task, nil)

	result, err := mockRepo.GetByID(nil, task.ID)
	assert.NoError(t, err)
	assert.Equal(t, task.ID, result.ID)

	// Test Update
	mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	err = mockRepo.Update(nil, task)
	assert.NoError(t, err)

	// Test Delete
	mockRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)

	err = mockRepo.Delete(nil, task.ID)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestNudgeRepositoryMocks_GetByUserID(t *testing.T) {
	mockRepo := &MockNudgeRepository{}

	userID := common.UserID("user1")
	tasks := []*nudge.Task{
		{
			ID:          common.TaskID("task1"),
			UserID:      userID,
			Description: "Task 1",
			Status:      nudge.StatusPending,
		},
		{
			ID:          common.TaskID("task2"),
			UserID:      userID,
			Description: "Task 2",
			Status:      nudge.StatusCompleted,
		},
	}

	mockRepo.On("GetByUserID", mock.Anything, userID).Return(tasks, nil)

	result, err := mockRepo.GetByUserID(nil, userID)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "Task 1", result[0].Description)
	assert.Equal(t, "Task 2", result[1].Description)

	mockRepo.AssertExpectations(t)
}

func TestNudgeRepositoryMocks_GetPendingReminders(t *testing.T) {
	mockRepo := &MockNudgeRepository{}

	reminders := []*nudge.Reminder{
		{
			ID:     common.ReminderID("rem1"),
			TaskID: common.TaskID("task1"),
			UserID: common.UserID("user1"),
		},
		{
			ID:     common.ReminderID("rem2"),
			TaskID: common.TaskID("task2"),
			UserID: common.UserID("user2"),
		},
	}

	mockRepo.On("GetPendingReminders", mock.Anything).Return(reminders, nil)

	result, err := mockRepo.GetPendingReminders(nil)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	mockRepo.AssertExpectations(t)
}

func TestSchedulerMocks_StartStop(t *testing.T) {
	mockScheduler := &MockScheduler{}

	// Test Start
	mockScheduler.On("Start", mock.Anything).Return(nil)

	err := mockScheduler.Start(nil)
	assert.NoError(t, err)

	// Test Stop
	mockScheduler.On("Stop", mock.Anything).Return(nil)

	err = mockScheduler.Stop(nil)
	assert.NoError(t, err)

	mockScheduler.AssertExpectations(t)
}

func TestSchedulerMocks_IsRunning(t *testing.T) {
	mockScheduler := &MockScheduler{}

	// Test IsRunning
	mockScheduler.On("IsRunning").Return(true)

	running := mockScheduler.IsRunning()
	assert.True(t, running)

	mockScheduler.AssertExpectations(t)
}

func TestSchedulerMocks_ProcessReminders(t *testing.T) {
	mockScheduler := &MockScheduler{}

	reminders := []*nudge.Reminder{
		{
			ID:     common.ReminderID("rem1"),
			TaskID: common.TaskID("task1"),
			UserID: common.UserID("user1"),
		},
	}

	mockScheduler.On("ProcessReminders", mock.Anything, reminders).Return(nil)

	err := mockScheduler.ProcessReminders(nil, reminders)
	assert.NoError(t, err)

	mockScheduler.AssertExpectations(t)
}

func TestMockErrors_Handling(t *testing.T) {
	mockChatbot := &MockChatbotService{}

	// Test error scenarios
	expectedError := assert.AnError
	mockChatbot.On("SendMessage", mock.Anything, mock.Anything).Return(expectedError)

	err := mockChatbot.SendMessage(common.ChatID("123"), "test")
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)

	mockChatbot.AssertExpectations(t)
}

func TestMockArguments_Matching(t *testing.T) {
	mockChatbot := &MockChatbotService{}

	// Test specific argument matching
	specificChatID := common.ChatID("specific-chat")
	specificMessage := "specific message"

	mockChatbot.On("SendMessage", specificChatID, specificMessage).Return(nil)

	// This should work
	err := mockChatbot.SendMessage(specificChatID, specificMessage)
	assert.NoError(t, err)

	mockChatbot.AssertExpectations(t)
}

func TestMockArguments_AnyMatch(t *testing.T) {
	mockChatbot := &MockChatbotService{}

	// Test any argument matching
	mockChatbot.On("SendMessage", mock.Anything, mock.Anything).Return(nil)

	// These should all work
	err1 := mockChatbot.SendMessage(common.ChatID("chat1"), "message1")
	err2 := mockChatbot.SendMessage(common.ChatID("chat2"), "message2")
	err3 := mockChatbot.SendMessage(common.ChatID("chat3"), "message3")

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)

	mockChatbot.AssertExpectations(t)
}

func TestMockReturnValues_Multiple(t *testing.T) {
	mockRepo := &MockNudgeRepository{}

	task := &nudge.Task{
		ID:     common.TaskID("task1"),
		UserID: common.UserID("user1"),
	}

	// Test multiple return values
	mockRepo.On("GetByID", mock.Anything, mock.Anything).Return(task, nil)

	result, err := mockRepo.GetByID(nil, task.ID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, task.ID, result.ID)

	mockRepo.AssertExpectations(t)
}

func TestMockCallCounting(t *testing.T) {
	mockChatbot := &MockChatbotService{}

	// Test call counting
	mockChatbot.On("SendMessage", mock.Anything, mock.Anything).Return(nil).Times(3)

	// Call exactly 3 times
	for i := 0; i < 3; i++ {
		err := mockChatbot.SendMessage(common.ChatID("123"), "test")
		assert.NoError(t, err)
	}

	mockChatbot.AssertExpectations(t)
}

func TestMockOnce(t *testing.T) {
	mockBus := &MockEventBus{}

	event := events.Event{Type: events.TaskCreatedEvent}

	// Test Once() method
	mockBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Once()

	err := mockBus.Publish("test.topic", event)
	assert.NoError(t, err)

	mockBus.AssertExpectations(t)
}

func TestMockCallback_Functions(t *testing.T) {
	mockChatbot := &MockChatbotService{}

	// Test callback functions
	called := false
	mockChatbot.On("SendMessage", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		called = true
		chatID := args.Get(0).(common.ChatID)
		message := args.Get(1).(string)
		assert.Equal(t, common.ChatID("123"), chatID)
		assert.Equal(t, "test", message)
	})

	err := mockChatbot.SendMessage(common.ChatID("123"), "test")
	assert.NoError(t, err)
	assert.True(t, called)

	mockChatbot.AssertExpectations(t)
}
