## Product Requirements Document: NudgeBot (Telegram MVP)

**Version:** 1.2 (Development-Ready)
**Date:** November 3, 2023
**Author:** Product Team
**Status:** Approved for Development

### 1. Introduction

**NudgeBot** is a conversational task assistant built for the Telegram platform. Its core purpose is to combat the common failure point of traditional to-do apps: the "swipe and forget" phenomenon. NudgeBot acts as a caring, persistent assistant that not only reminds users of their tasks on time but also follows up later if a task remains incomplete. This proactive "nudge" is designed to provide gentle accountability, helping users ensure their important tasks are actually completed.

This document outlines the requirements for the Minimum Viable Product (MVP), which will be developed as a Telegram bot to validate the core value proposition before investing in a standalone application.

### 2. Problem Statement

Busy individuals often rely on digital reminders, but a single notification is easily dismissed and forgotten amidst a flood of other alerts. This leads to missed deadlines, forgotten errands, and increased stress. Existing tools are passive; they inform but do not assist in the follow-through. There is a need for a tool that closes the loop between reminder and action, acting as an accountability partner rather than a simple alarm clock.

### 3. Target Audience & Persona

*   **Target Audience:** Digitally-savvy students, freelancers, and professionals who use messenger apps for daily communication and feel overwhelmed by their task lists.
*   **Primary Persona: "Alex," the Freelance Designer**
    *   **Age:** 28
    *   **Behavior:** Juggles multiple client projects with tight deadlines. Lives in Telegram and Slack. Creates to-do lists but often gets sidetracked by urgent client requests, causing non-urgent but important tasks to slip.
    *   **Pain Point:** "I get the reminder to send an invoice, but then a client calls with an 'emergency.' I swipe the notification away to deal with it and completely forget about the invoice until two days later."

### 4. Goals & Success Metrics (KPIs)

| Goal Type | Goal | Key Performance Indicator (KPI) | Success Target (First 3 Months) |
| :--- | :--- | :--- | :--- |
| **Product** | Validate the "nudge" system's effectiveness. | **Nudge Effectiveness Rate:** (# of tasks completed *after* a nudge) / (# of total nudges sent) | > 25% |
| **Product** | Achieve user adoption and engagement. | **Weekly Active Users (WAU):** Unique users interacting with the bot in a 7-day period. | > 200 WAU |
| **Product** | Ensure the bot is useful and not annoying. | **Retention Rate (Week 4):** % of new users who are still active 4 weeks after starting. | > 20% |
| **Business** | Prove Product-Market Fit for a future app. | **Qualitative Feedback:** Gather feedback via a `/feedback` command to inform future development. | > 50 pieces of actionable feedback |

### 5. User Stories

| ID | As a... | I want to... | So that I can... |
| :-- | :--- | :--- | :--- |
| US-01 | New User | Be greeted with a simple welcome message and instructions when I first start the bot. | Understand its purpose and how to use it immediately. |
| US-02 | User | Add a task using natural language, including a date and time. | Quickly capture tasks without a complicated interface. |
| US-03 | User | Receive an immediate confirmation that the bot understood my task correctly. | Have confidence that my reminder is set properly. |
| US-04 | User | Be reminded via a Telegram message when my task is due. | Act on my task at the appropriate time. |
| US-05 | User | Have inline buttons on the reminder to mark it "Done" or "Snooze". | Quickly manage a task without typing commands. |
| US-06 | User | Receive a friendly, non-intrusive follow-up message if I don't complete a task. | Be held accountable and not let things slip through the cracks. |
| US-07 | User | View a clean list of all my upcoming tasks. | Get an overview of my commitments. |
| US-08 | User | Mark a task as complete from the list view. | Easily manage my to-do list. |
| US-09 | User | Delete tasks that are no longer needed. | Keep my list relevant and uncluttered. |
| US-10 | User | Be told gracefully if the bot doesn't understand my message. | Know what went wrong and how to try again. |

### 6. Detailed Feature Specifications

#### 6.1. Onboarding & Help
*   **Command:** `/start`
*   **Action:** Triggers a one-time welcome message.
*   **Message Copy:**
    > "👋 Welcome to NudgeBot!
    >
    > I'm your friendly task assistant. Just tell me what to do and when, and I'll not only remind you, but I'll also check in later to make sure it gets done.
    >
    > **Try adding a task like:** `remind me to call Sarah tomorrow at 3pm`
    >
    > **Available commands:**
    > `/list` - View your active tasks
    > `/help` - See this message again"
*   **Command:** `/help`
    *   **Action:** Displays the same message as `/start`.

#### 6.2. Task Creation (NLU Flow)
1.  **User Input:** User sends a message (e.g., "submit the Q4 report by friday at noon").
2.  **Go Server:** Receives the webhook from Telegram.
3.  **API Call to Gemma:** The server sends a request to the Gemma API with a structured prompt.
    *   **Example Prompt:**
        ```json
        {
          "prompt": "You are a task parsing assistant. Analyze the following user text and extract the task_description and a precise ISO 8601 timestamp for the due_date. The current time is [Current ISO 8601 Time]. Text: 'submit the Q4 report by friday at noon'",
          "response_format": "json"
        }
        ```
4.  **Gemma Response:** Gemma returns a JSON object.
    *   **Expected JSON Structure:**
        ```json
        {
          "task_description": "Submit the Q4 report",
          "due_date": "2023-11-10T12:00:00Z"
        }
        ```
5.  **Database:** The Go server saves the `task_description`, `due_date`, `telegram_user_id`, and `chat_id` to the PostgreSQL database with a status of `active`.
6.  **Confirmation to User:** The bot sends a confirmation message.
    *   **Message Copy:** "✅ Got it! I'll remind you to: **Submit the Q4 report** on Friday, Nov 10 at 12:00 PM."

#### 6.3. Reminder & Snooze Flow
1.  **Trigger:** The backend scheduler finds a task whose `due_date` is now.
2.  **Action:** The bot sends a message to the user's chat.
    *   **Message Copy:** "🔔 Time for: **Submit the Q4 report**"
    *   **Inline Keyboard Buttons:**
        *   `✅ Done`
        *   `⏰ Snooze`
3.  **User Clicks `✅ Done`:** The task status is updated to `completed` in the database. The bot edits the original message to "👍 Great work! Task completed."
4.  **User Clicks `⏰ Snooze`:** The buttons are replaced with new snooze options.
    *   **Snooze Buttons:** `15 Minutes`, `1 Hour`, `Tomorrow Morning`
    *   When a snooze option is clicked, the `due_date` for the task is updated in the database, and the bot replies: "Okay, I'll remind you again in [snooze duration]."
    *   `Tomorrow Morning` will be defined as 9:00 AM in the bot's configured timezone (initially UTC for MVP).

#### 6.4. The Nudge System
*   **Trigger:** The backend scheduler identifies an `active` task whose `due_date` is more than **2 hours** in the past and has not been snoozed or completed.
*   **Action:** The bot sends a new, distinct message. This is a one-time nudge per task.
*   **Nudge Message Copy:** "Hey, just checking in. Were you able to get to this task? **Submit the Q4 report**"
*   **Inline Keyboard Buttons:**
    *   `✅ Yes, it's Done!` (marks task as `completed`)
    *   `⏰ Remind Me Later` (triggers the snooze options again)

#### 6.5. Task Management Commands
*   **Command:** `/list`
    *   **Action:** Bot retrieves all `active` tasks for the user, sorted by `due_date`.
    *   **Output Format:**
        > "Here are your upcoming tasks:
        >
        > 1. Call Sarah - Due: Nov 4, 3:00 PM
        > 2. Submit the Q4 report - Due: Nov 10, 12:00 PM
        >
        > To manage a task, use `/done [number]` or `/delete [number]`."
*   **Command:** `/done [number]` (e.g., `/done 1`)
    *   **Action:** Marks the corresponding task as `completed`.
    *   **Reply:** "✅ Task 'Call Sarah' marked as done!"
*   **Command:** `/delete [number]`
    *   **Action:** Deletes the task from the database.
    *   **Reply:** "🗑️ Task 'Call Sarah' has been deleted."

#### 6.6. Error Handling
*   **Trigger:** The Gemma API returns an error or a JSON that doesn't contain a `task_description` or `due_date`.
*   **Action:** The bot sends a helpful error message.
*   **Message Copy:** "Sorry, I had trouble understanding that. Could you try phrasing it more simply, like `remind me to [task] on [date] at [time]`?"

### 7. Technical Architecture & Stack

*   **Platform:** Telegram Bot API
*   **Backend Language:** **Go (Golang)** - Chosen for performance and concurrency, essential for the scheduler.
*   **Web Framework:** **Gin** - A lightweight, high-performance web framework for Go.
*   **NLU Service:** **Google Gemma API** - All natural language parsing will be handled via API calls.
*   **Database:** **PostgreSQL** - To store user and task data.
*   **Scheduler:** A custom scheduler built with native **Go goroutines and channels**. A main goroutine will "tick" every minute, querying the DB and spawning new goroutines for due tasks.
*   **Deployment:** The application will be packaged in a **Docker container** and hosted on a cloud service like **Google Cloud Run**.

### 8. Scope: Out of MVP

The following features will NOT be included in this MVP but will be considered for future versions:
*   Recurring tasks (e.g., "every Tuesday").
*   Tasks without a specific due date ("someday" list).
*   Location-based reminders.
*   Editing existing tasks.
*   Timezone personalization for users.
*   File attachments or detailed notes.
*   Any form of monetization.