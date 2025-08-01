package chatbot

import (
	"encoding/json"
	"fmt"
	"time"

	"nudgebot-api/internal/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// KeyboardBuilder provides utilities for creating inline keyboards
type KeyboardBuilder struct{}

// NewKeyboardBuilder creates a new KeyboardBuilder instance
func NewKeyboardBuilder() *KeyboardBuilder {
	return &KeyboardBuilder{}
}

// TaskSummary represents task information for keyboard display
type TaskSummary struct {
	ID      common.TaskID `json:"id"`
	Title   string        `json:"title"`
	DueDate *time.Time    `json:"due_date,omitempty"`
	Status  string        `json:"status"`
}

// CallbackAction constants for different button actions
const (
	CallbackActionDone     = "done"
	CallbackActionDelete   = "delete"
	CallbackActionConfirm  = "confirm"
	CallbackActionCancel   = "cancel"
	CallbackActionList     = "list"
	CallbackActionSnooze   = "snooze"
	CallbackActionPrevPage = "prev_page"
	CallbackActionNextPage = "next_page"
	CallbackActionBack     = "back"
	CallbackActionHelp     = "help"
)

// BuildTaskActionKeyboard creates Done/Delete buttons for a specific task
func (kb *KeyboardBuilder) BuildTaskActionKeyboard(taskID string) tgbotapi.InlineKeyboardMarkup {
	doneData := kb.encodeCallbackData(CallbackActionDone, map[string]string{
		"task_id": taskID,
	})

	deleteData := kb.encodeCallbackData(CallbackActionDelete, map[string]string{
		"task_id": taskID,
	})

	snoozeData := kb.encodeCallbackData(CallbackActionSnooze, map[string]string{
		"task_id": taskID,
	})

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Done", doneData),
			tgbotapi.NewInlineKeyboardButtonData("❌ Delete", deleteData),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏰ Snooze", snoozeData),
		),
	)
}

// BuildTaskListKeyboard creates a paginated task list with action buttons
func (kb *KeyboardBuilder) BuildTaskListKeyboard(tasks []TaskSummary, currentPage, totalPages int) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	// Add task buttons (max 5 per page)
	const tasksPerPage = 5
	startIdx := currentPage * tasksPerPage
	endIdx := startIdx + tasksPerPage
	if endIdx > len(tasks) {
		endIdx = len(tasks)
	}

	for i := startIdx; i < endIdx; i++ {
		task := tasks[i]
		buttonText := fmt.Sprintf("📋 %s", truncateText(task.Title, 30))

		callbackData := kb.encodeCallbackData("view_task", map[string]string{
			"task_id": string(task.ID),
		})

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData),
		))
	}

	// Add pagination row if needed
	if totalPages > 1 {
		var paginationRow []tgbotapi.InlineKeyboardButton

		if currentPage > 0 {
			prevData := kb.encodeCallbackData(CallbackActionPrevPage, map[string]string{
				"page": fmt.Sprintf("%d", currentPage-1),
			})
			paginationRow = append(paginationRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Prev", prevData))
		}

		// Page indicator
		pageText := fmt.Sprintf("%d/%d", currentPage+1, totalPages)
		paginationRow = append(paginationRow, tgbotapi.NewInlineKeyboardButtonData(pageText, "noop"))

		if currentPage < totalPages-1 {
			nextData := kb.encodeCallbackData(CallbackActionNextPage, map[string]string{
				"page": fmt.Sprintf("%d", currentPage+1),
			})
			paginationRow = append(paginationRow, tgbotapi.NewInlineKeyboardButtonData("➡️ Next", nextData))
		}

		rows = append(rows, paginationRow)
	}

	// Add back button
	backData := kb.encodeCallbackData(CallbackActionBack, map[string]string{})
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Back", backData),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// BuildConfirmationKeyboard creates a confirmation dialog with Yes/No buttons
func (kb *KeyboardBuilder) BuildConfirmationKeyboard(action string, taskID string) tgbotapi.InlineKeyboardMarkup {
	confirmData := kb.encodeCallbackData(CallbackActionConfirm, map[string]string{
		"action":  action,
		"task_id": taskID,
	})

	cancelData := kb.encodeCallbackData(CallbackActionCancel, map[string]string{
		"action":  action,
		"task_id": taskID,
	})

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Yes", confirmData),
			tgbotapi.NewInlineKeyboardButtonData("❌ No", cancelData),
		),
	)
}

// BuildMainMenuKeyboard creates the main bot menu with common actions
func (kb *KeyboardBuilder) BuildMainMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	listData := kb.encodeCallbackData(CallbackActionList, map[string]string{})
	helpData := kb.encodeCallbackData(CallbackActionHelp, map[string]string{})

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 My Tasks", listData),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❓ Help", helpData),
		),
	)
}

// BuildPaginationKeyboard creates navigation buttons for pagination
func (kb *KeyboardBuilder) BuildPaginationKeyboard(currentPage, totalPages int, baseCallback string) tgbotapi.InlineKeyboardMarkup {
	var buttons []tgbotapi.InlineKeyboardButton

	if currentPage > 0 {
		prevData := kb.encodeCallbackData(CallbackActionPrevPage, map[string]string{
			"page":     fmt.Sprintf("%d", currentPage-1),
			"callback": baseCallback,
		})
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("⬅️", prevData))
	}

	// Page indicator
	pageText := fmt.Sprintf("%d/%d", currentPage+1, totalPages)
	buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(pageText, "noop"))

	if currentPage < totalPages-1 {
		nextData := kb.encodeCallbackData(CallbackActionNextPage, map[string]string{
			"page":     fmt.Sprintf("%d", currentPage+1),
			"callback": baseCallback,
		})
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("➡️", nextData))
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(buttons...),
	)
}

// ConvertDomainKeyboard converts domain InlineKeyboard to Telegram format
func (kb *KeyboardBuilder) ConvertDomainKeyboard(domainKeyboard InlineKeyboard) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, buttonRow := range domainKeyboard.Buttons {
		var tgButtonRow []tgbotapi.InlineKeyboardButton

		for _, button := range buttonRow {
			var tgButton tgbotapi.InlineKeyboardButton

			if button.URL != "" {
				tgButton = tgbotapi.NewInlineKeyboardButtonURL(button.Text, button.URL)
			} else {
				tgButton = tgbotapi.NewInlineKeyboardButtonData(button.Text, button.CallbackData)
			}

			tgButtonRow = append(tgButtonRow, tgButton)
		}

		rows = append(rows, tgButtonRow)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// encodeCallbackData encodes callback data as JSON string
func (kb *KeyboardBuilder) encodeCallbackData(action string, data map[string]string) string {
	callbackData := CallbackData{
		Action: action,
		Data:   data,
	}

	jsonData, err := json.Marshal(callbackData)
	if err != nil {
		// Fallback to simple action if JSON encoding fails
		return action
	}

	// Telegram has a 64-byte limit for callback data
	if len(jsonData) > 64 {
		return action
	}

	return string(jsonData)
}

// DecodeCallbackData decodes JSON callback data
func (kb *KeyboardBuilder) DecodeCallbackData(callbackDataStr string) (*CallbackData, error) {
	var callbackData CallbackData

	if err := json.Unmarshal([]byte(callbackDataStr), &callbackData); err != nil {
		// Fallback to simple string format
		return &CallbackData{
			Action: callbackDataStr,
			Data:   make(map[string]string),
		}, nil
	}

	return &callbackData, nil
}

// truncateText truncates text to specified length with ellipsis
func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	if maxLength <= 3 {
		return text[:maxLength]
	}

	return text[:maxLength-3] + "..."
}
