# Enhanced Error Handling for Task List Requests

## Summary

Enhanced the error handling in the `handleTaskListRequested()` method in `internal/nudge/service.go` and updated the event types to support comprehensive error responses. The chatbot service now provides meaningful feedback to users when task list requests fail.

## Changes Made

### 1. Enhanced TaskListResponse Event Type (`internal/events/types.go`)

Added error-related fields to the `TaskListResponse` struct:
```go
type TaskListResponse struct {
    Event
    UserID     string        `json:"user_id" validate:"required"`
    ChatID     string        `json:"chat_id" validate:"required"`
    Tasks      []TaskSummary `json:"tasks"`
    TotalCount int           `json:"total_count"`
    HasMore    bool          `json:"has_more"`
    Success    bool          `json:"success"`          // NEW: Indicates operation success
    ErrorCode  string        `json:"error_code,omitempty"`    // NEW: Specific error code
    ErrorMsg   string        `json:"error_message,omitempty"` // NEW: Human-readable error message
}
```

### 2. New Error Types (`internal/nudge/errors.go`)

#### Added Error Codes:
- `ErrCodeTaskListFailed` - General task list retrieval failures
- `ErrCodeUserNotFound` - User doesn't exist
- `ErrCodeUnauthorized` - Access denied
- `ErrCodeInvalidRequest` - Invalid request format

#### New TaskListError Type:
```go
type TaskListError struct {
    UserID     common.UserID
    Details    string
    ErrorCode  string
    Cause      error
    Retryable  bool
}
```

#### Helper Constructors:
- `NewTaskListError()` - General task list errors (retryable)
- `NewTaskListValidationError()` - Validation errors (non-retryable)
- `NewTaskListUnauthorizedError()` - Authorization errors (non-retryable)

### 3. Enhanced handleTaskListRequested Method (`internal/nudge/service.go`)

#### Input Validation:
- Validates required fields (UserID, ChatID)
- Validates UUID format for UserID
- Validates ChatID is not empty
- Early validation failure returns specific error response

#### Comprehensive Error Handling:
- Categorizes different types of errors (validation, database, authorization)
- Creates appropriate error responses with specific error codes
- Provides detailed logging for debugging
- Maintains backward compatibility for successful responses

#### New Helper Methods:
- `validateTaskListRequest()` - Validates incoming task list requests
- `publishTaskListErrorResponse()` - Publishes error responses with proper error details

### 4. Enhanced Chatbot Error Handling (`internal/chatbot/service.go`)

#### Updated handleTaskListResponse Method:
- Checks `Success` field in response
- Handles error responses before processing successful ones
- Provides specific user feedback based on error type
- Logs errors for monitoring

#### New Error Message Formatting:
- `formatTaskListErrorMessage()` - Creates user-friendly error messages
- Maps error codes to appropriate user messages
- Provides fallback for unknown error types

#### Error Code to Message Mapping:
- `VALIDATION_FAILED` → "Invalid Request" message
- `UNAUTHORIZED` → "Access Denied" message  
- `USER_NOT_FOUND` → "User Not Found" message
- `REPOSITORY_ERROR` → "System Temporarily Unavailable" message
- `TASK_LIST_FAILED` → "Unable to Retrieve Tasks" message
- Default → Generic error message with details

## Error Handling Flow

1. **Request Validation**
   - Basic field validation (required fields, formats)
   - Early return with validation error if invalid

2. **Database Operation**
   - Attempt to retrieve tasks from repository
   - Catch and categorize database errors
   - Create appropriate TaskListError with context

3. **Response Generation**
   - Success: Generate normal TaskListResponse with tasks
   - Error: Generate TaskListResponse with error details

4. **User Feedback**
   - Chatbot receives response and checks Success field
   - Error responses show user-friendly messages
   - Success responses show task list as before

## Benefits

### Security & Reliability
- **Input Validation**: Prevents invalid requests from reaching business logic
- **Error Classification**: Different error types handled appropriately
- **Graceful Degradation**: System continues operating even with partial failures

### User Experience
- **Clear Error Messages**: Users understand what went wrong
- **Actionable Feedback**: Error messages suggest what users can do
- **Consistent Interface**: All task list errors handled uniformly

### Debugging & Monitoring
- **Detailed Logging**: Errors logged with context for debugging
- **Error Codes**: Categorized errors for monitoring and metrics
- **Error Tracing**: Full error chain preserved for root cause analysis

### Maintainability
- **Centralized Error Handling**: All error logic in dedicated methods
- **Extensible Design**: Easy to add new error types and messages
- **Type Safety**: Structured error types prevent runtime issues

## Error Response Examples

### Validation Error Response:
```json
{
  "user_id": "invalid-uuid",
  "chat_id": "123456",
  "tasks": [],
  "total_count": 0,
  "has_more": false,
  "success": false,
  "error_code": "VALIDATION_FAILED",
  "error_message": "userID must be a valid UUID: invalid-uuid"
}
```

### Database Error Response:
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "chat_id": "123456", 
  "tasks": [],
  "total_count": 0,
  "has_more": false,
  "success": false,
  "error_code": "REPOSITORY_ERROR",
  "error_message": "Failed to retrieve tasks from database"
}
```

### Success Response (unchanged format, added success field):
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "chat_id": "123456",
  "tasks": [...],
  "total_count": 5,
  "has_more": false,
  "success": true,
  "error_code": "",
  "error_message": ""
}
```
