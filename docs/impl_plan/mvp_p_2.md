# NudgeBot MVP - Implementation Plan (Phase 2)

**Status:** Planning
**Date:** August 4, 2025

This document outlines the next phase of implementation required to fully meet the NudgeBot MVP product requirements, based on a review of the existing codebase against the PRD.

## Identified Gaps & Recommendations

The foundational architecture is solid and aligns well with the PRD. The following items represent the primary gaps that need to be addressed to complete the MVP functionality.

---

### 1. Nudge Logic Implementation

*   **Gap:** The core "nudge" feature, which is central to the product's value proposition, is not fully implemented. The current scheduler is generic and does not contain the specific business rule to send a follow-up message for tasks that are past their due date and have not been completed.

*   **Recommendation:**
    1.  **Enhance Scheduler Query:** Modify the `scheduler`'s primary loop (in `internal/scheduler/worker.go`) to include a new query. This query should find tasks where:
        -   `status` is 'active'.
        -   `due_date` is more than 2 hours in the past.
        -   The task has not already had a nudge sent (this may require adding a `nudge_sent_at` timestamp or a similar flag to the `Task` model in `internal/nudge/domain.go`).
    2.  **Dispatch Nudge Event:** When such tasks are found, the scheduler should publish a new, specific event, such as `NudgeDue`, or reuse the existing `ReminderDue` event with the `ReminderType` set to `nudge`.
    3.  **Handle Nudge Event:** The `chatbot.service` needs to subscribe to this event and send the specific nudge message copy defined in the PRD, including the "Yes, it's Done!" and "Remind Me Later" inline keyboard buttons.

---

### 2. Snooze Functionality Refinement

*   **Gap:** The PRD describes a two-step snooze mechanism: a user first clicks a general "Snooze" button, which is then replaced by specific duration options ("15 Minutes", "1 Hour", "Tomorrow Morning"). The current implementation handles a generic snooze action but lacks this specific, user-friendly, two-step flow.

*   **Recommendation:**
    1.  **Update Callback Handler:** In the `chatbot.service`'s `handleCallbackQuery` function, add specific logic to detect a callback with the action `snooze_task`.
    2.  **Edit Message Keyboard:** Instead of immediately snoozing the task, this logic should call the Telegram provider to *edit* the original message's inline keyboard, replacing it with the new set of buttons for snooze durations (e.g., `snooze_15m`, `snooze_1h`, `snooze_tomorrow`).
    3.  **Handle Duration Callbacks:** Add handlers for these new duration-specific callbacks. These handlers will calculate the new `due_date` and then call the `nudge.service`'s `SnoozeTask` method.

---

### 3. Numbered Command Arguments (`/done 1`)

*   **Gap:** The PRD requires that users can manage tasks from the `/list` view by using commands with numeric arguments, like `/done 1` or `/delete 2`. The current command processing logic in `chatbot.service` is not equipped to map these numbers back to the correct `TaskID`, as it would require knowledge of the user's last viewed list.

*   **Recommendation:**
    1.  **Implement User Session Cache:** Introduce a simple, short-lived cache within the `chatbot` service. This cache will be keyed by `UserID` or `ChatID`.
    2.  **Cache Task List:** When a user requests `/list`, the `chatbot.service` should store the ordered list of `TaskID`s that it displays to the user in this cache.
    3.  **Resolve Numbered Commands:** When a `/done [number]` or `/delete [number]` command is received, the `commandProcessor` should:
        a.  Parse the number from the command.
        b.  Look up the user's cached list of `TaskID`s.
        c.  Retrieve the `TaskID` at the specified index (e.g., number `1` corresponds to index `0`).
        d.  Dispatch the `TaskActionRequested` event using the resolved `TaskID`.
    4.  **Cache Invalidation:** The cache for a user should be cleared after a short time (e.g., 5-10 minutes) or after a successful `/done` or `/delete` action to prevent accidental actions on a stale list.
