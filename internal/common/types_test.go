package common

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestID_Generation(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{
			name: "NewID generates unique IDs",
			test: func(t *testing.T) {
				id1 := NewID()
				id2 := NewID()

				assert.NotEqual(t, id1, id2)
				assert.NotEmpty(t, id1)
				assert.NotEmpty(t, id2)
			},
		},
		{
			name: "NewID generates valid UUIDs",
			test: func(t *testing.T) {
				id := NewID()
				assert.True(t, id.IsValid())

				// Verify it's a valid UUID
				_, err := uuid.Parse(string(id))
				assert.NoError(t, err)
			},
		},
		{
			name: "IsValid returns true for valid UUIDs",
			test: func(t *testing.T) {
				validUUID := "550e8400-e29b-41d4-a716-446655440000"
				id := ID(validUUID)
				assert.True(t, id.IsValid())
			},
		},
		{
			name: "IsValid returns false for invalid UUIDs",
			test: func(t *testing.T) {
				invalidIDs := []string{
					"invalid-uuid",
					"",
					"550e8400-e29b-41d4-a716",
					"not-a-uuid-at-all",
				}

				for _, invalidID := range invalidIDs {
					id := ID(invalidID)
					assert.False(t, id.IsValid(), "Expected %s to be invalid", invalidID)
				}
			},
		},
		{
			name: "String returns string representation",
			test: func(t *testing.T) {
				testString := "test-id-string"
				id := ID(testString)
				assert.Equal(t, testString, id.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestTypedIDs(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{
			name: "UserID type safety",
			test: func(t *testing.T) {
				baseID := NewID()
				userID := UserID(baseID)

				assert.Equal(t, string(baseID), string(userID))
				assert.IsType(t, UserID(""), userID)
			},
		},
		{
			name: "ChatID type safety",
			test: func(t *testing.T) {
				baseID := NewID()
				chatID := ChatID(baseID)

				assert.Equal(t, string(baseID), string(chatID))
				assert.IsType(t, ChatID(""), chatID)
			},
		},
		{
			name: "TaskID type safety",
			test: func(t *testing.T) {
				baseID := NewID()
				taskID := TaskID(baseID)

				assert.Equal(t, string(baseID), string(taskID))
				assert.IsType(t, TaskID(""), taskID)
			},
		},
		{
			name: "Different typed IDs are distinct types",
			test: func(t *testing.T) {
				baseID := NewID()
				userID := UserID(baseID)
				chatID := ChatID(baseID)
				taskID := TaskID(baseID)

				// These should have the same underlying value but different types
				assert.Equal(t, string(userID), string(chatID))
				assert.Equal(t, string(chatID), string(taskID))

				// Type assertions should work
				assert.IsType(t, UserID(""), userID)
				assert.IsType(t, ChatID(""), chatID)
				assert.IsType(t, TaskID(""), taskID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name     string
		error    error
		expected string
	}{
		{
			name: "ValidationError",
			error: ValidationError{
				Field:   "email",
				Message: "invalid email format",
			},
			expected: "validation error for field 'email': invalid email format",
		},
		{
			name: "NotFoundError",
			error: NotFoundError{
				Resource: "User",
				ID:       "123",
			},
			expected: "User with ID '123' not found",
		},
		{
			name: "InternalError without cause",
			error: InternalError{
				Message: "something went wrong",
			},
			expected: "internal error: something went wrong",
		},
		{
			name: "InternalError with cause",
			error: InternalError{
				Message: "database operation failed",
				Cause:   ValidationError{Field: "id", Message: "required"},
			},
			expected: "internal error: database operation failed (caused by: validation error for field 'id': required)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.error.Error())
		})
	}
}

func TestTaskStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   TaskStatus
		isValid  bool
		expected string
	}{
		{"TaskStatusActive", TaskStatusActive, true, "active"},
		{"TaskStatusCompleted", TaskStatusCompleted, true, "completed"},
		{"TaskStatusSnoozed", TaskStatusSnoozed, true, "snoozed"},
		{"TaskStatusDeleted", TaskStatusDeleted, true, "deleted"},
		{"Invalid status", TaskStatus("invalid"), false, "invalid"},
		{"Empty status", TaskStatus(""), false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.status.IsValid())
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority Priority
		isValid  bool
		expected string
	}{
		{"PriorityLow", PriorityLow, true, "low"},
		{"PriorityMedium", PriorityMedium, true, "medium"},
		{"PriorityHigh", PriorityHigh, true, "high"},
		{"PriorityUrgent", PriorityUrgent, true, "urgent"},
		{"Invalid priority", Priority("invalid"), false, "invalid"},
		{"Empty priority", Priority(""), false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.priority.IsValid())
			assert.Equal(t, tt.expected, tt.priority.String())
		})
	}
}

func TestUUIDHelpers(t *testing.T) {
	t.Run("Generated IDs are valid UUIDs", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			id := NewID()
			assert.True(t, id.IsValid(), "Generated ID should be valid UUID")

			// Parse as UUID to ensure it's valid
			parsedUUID, err := uuid.Parse(string(id))
			assert.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, parsedUUID)
		}
	})

	t.Run("ID uniqueness", func(t *testing.T) {
		const numIDs = 1000
		ids := make(map[string]bool, numIDs)

		for i := 0; i < numIDs; i++ {
			id := NewID()
			idStr := string(id)

			assert.False(t, ids[idStr], "ID %s should be unique", idStr)
			ids[idStr] = true
		}

		assert.Len(t, ids, numIDs)
	})
}

func TestStringMethods(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{
			name: "ID JSON marshaling",
			test: func(t *testing.T) {
				id := NewID()

				jsonData, err := json.Marshal(id)
				require.NoError(t, err)

				var unmarshaled ID
				err = json.Unmarshal(jsonData, &unmarshaled)
				require.NoError(t, err)

				assert.Equal(t, id, unmarshaled)
			},
		},
		{
			name: "TaskStatus string representation",
			test: func(t *testing.T) {
				statuses := map[TaskStatus]string{
					TaskStatusActive:    "active",
					TaskStatusCompleted: "completed",
					TaskStatusSnoozed:   "snoozed",
					TaskStatusDeleted:   "deleted",
				}

				for status, expected := range statuses {
					assert.Equal(t, expected, status.String())
				}
			},
		},
		{
			name: "Priority string representation",
			test: func(t *testing.T) {
				priorities := map[Priority]string{
					PriorityLow:    "low",
					PriorityMedium: "medium",
					PriorityHigh:   "high",
					PriorityUrgent: "urgent",
				}

				for priority, expected := range priorities {
					assert.Equal(t, expected, priority.String())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestErrorUnwrapping(t *testing.T) {
	originalErr := ValidationError{Field: "test", Message: "test error"}
	wrappedErr := InternalError{
		Message: "wrapped error",
		Cause:   originalErr,
	}

	unwrapped := wrappedErr.Unwrap()
	assert.Equal(t, originalErr, unwrapped)

	// Test with no cause
	noCauseErr := InternalError{Message: "no cause"}
	assert.Nil(t, noCauseErr.Unwrap())
}

func TestTaskStatusTransitions(t *testing.T) {
	// Test valid status values
	validStatuses := []TaskStatus{
		TaskStatusActive,
		TaskStatusCompleted,
		TaskStatusSnoozed,
		TaskStatusDeleted,
	}

	for _, status := range validStatuses {
		t.Run("Valid status: "+string(status), func(t *testing.T) {
			assert.True(t, status.IsValid())
			assert.NotEmpty(t, status.String())
		})
	}

	// Test status comparison
	t.Run("Status equality", func(t *testing.T) {
		status1 := TaskStatusActive
		status2 := TaskStatusActive
		status3 := TaskStatusCompleted

		assert.Equal(t, status1, status2)
		assert.NotEqual(t, status1, status3)
	})
}

func TestPriorityComparison(t *testing.T) {
	// Test valid priority values
	validPriorities := []Priority{
		PriorityLow,
		PriorityMedium,
		PriorityHigh,
		PriorityUrgent,
	}

	for _, priority := range validPriorities {
		t.Run("Valid priority: "+string(priority), func(t *testing.T) {
			assert.True(t, priority.IsValid())
			assert.NotEmpty(t, priority.String())
		})
	}

	// Test priority comparison
	t.Run("Priority equality", func(t *testing.T) {
		priority1 := PriorityHigh
		priority2 := PriorityHigh
		priority3 := PriorityLow

		assert.Equal(t, priority1, priority2)
		assert.NotEqual(t, priority1, priority3)
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("Empty ID validation", func(t *testing.T) {
		emptyID := ID("")
		assert.False(t, emptyID.IsValid())
		assert.Equal(t, "", emptyID.String())
	})

	t.Run("Malformed UUID validation", func(t *testing.T) {
		malformedIDs := []string{
			"550e8400-e29b-41d4-a716-44665544000",   // Too short
			"550e8400-e29b-41d4-a716-446655440000x", // Extra character
			"550e8400xe29bx41d4xa716x446655440000",  // Wrong separators
			"not-a-uuid",
			"12345",
		}

		for _, malformed := range malformedIDs {
			id := ID(malformed)
			assert.False(t, id.IsValid(), "Expected %s to be invalid", malformed)
		}
	})

	t.Run("JSON marshaling edge cases", func(t *testing.T) {
		// Test with empty ID
		emptyID := ID("")
		jsonData, err := json.Marshal(emptyID)
		require.NoError(t, err)
		assert.Equal(t, `""`, string(jsonData))

		// Test unmarshaling empty JSON
		var id ID
		err = json.Unmarshal([]byte(`""`), &id)
		require.NoError(t, err)
		assert.Equal(t, ID(""), id)

		// Test unmarshaling invalid JSON
		err = json.Unmarshal([]byte(`invalid`), &id)
		assert.Error(t, err)
	})
}
