# Task Action Validation Implementation

## Summary

Added comprehensive input validation to the `handleTaskActionRequested()` method in `internal/nudge/service.go`. The validation ensures that task action requests are properly validated before processing.

## Validation Features Implemented

### 1. Event Structure Validation
- **UserID**: Must be present and a valid UUID format
- **ChatID**: Must be present
- **TaskID**: Must be present and a valid UUID format  
- **Action**: Must be present and one of the allowed actions

### 2. Action Validation
- **Allowed actions**: "done", "complete", "delete", "snooze"
- **Invalid actions**: Returns `NewInvalidTaskActionError` for unsupported actions

### 3. Task Ownership Validation
- **Task existence**: Verifies the task exists in the repository
- **User ownership**: Ensures the task belongs to the requesting user
- **Prevents unauthorized access**: Users cannot perform actions on tasks they don't own

### 4. Status-based Action Validation
- **Complete/Done actions**: Only allowed on active or snoozed tasks
- **Delete actions**: Allowed on all tasks except already deleted ones
- **Snooze actions**: Only allowed on active tasks

### 5. Mock Mode Support
- **Graceful degradation**: When repository is nil (mock mode), skips database validation
- **Testing friendly**: Allows testing without database dependencies

## Methods Added

### `validateTaskActionRequest(event events.TaskActionRequested) error`
Main validation function that performs comprehensive validation of task action requests.

### `validateActionForTaskStatus(action string, currentStatus common.TaskStatus) error`
Helper function that validates whether a specific action is allowed for the current task status.

### `publishTaskActionResponse(event events.TaskActionRequested, success bool, message string)`
Helper function that publishes TaskActionResponse events, extracted from the main handler for reusability.

## Error Handling

The validation provides detailed error messages for different failure scenarios:
- Missing required fields
- Invalid UUID formats
- Task not found
- Unauthorized access attempts
- Invalid action for current task status

## Integration

The validation is seamlessly integrated into the existing event handler:
1. Validation runs before any business logic
2. Failed validation immediately returns error response
3. Successful validation allows normal processing flow
4. All responses are properly published via the event bus

## Benefits

1. **Security**: Prevents unauthorized task modifications
2. **Data Integrity**: Ensures only valid operations are performed
3. **User Experience**: Provides clear error messages for invalid requests
4. **Maintainability**: Centralized validation logic that's easy to extend
5. **Testing**: Support for both database and mock mode testing
