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

#### 6.2. Task Creation (Modular NLU Flow)
1.  **User Input:** User sends a message (e.g., "submit the Q4 report by friday at noon").
2.  **Chatbot Integration Module:** Receives the webhook from Telegram and forwards the message to the LLM API Integration module.
3.  **LLM API Integration Module:** Processes the natural language request.
    *   Sends a request to the Gemma API with a structured prompt.
    *   **Example Prompt:**
        ```json
        {
          "prompt": "You are a task parsing assistant. Analyze the following user text and extract the task_description and a precise ISO 8601 timestamp for the due_date. The current time is [Current ISO 8601 Time]. Text: 'submit the Q4 report by friday at noon'",
          "response_format": "json"
        }
        ```
4.  **Gemma API Response:** Returns parsed task data to the LLM API Integration module.
    *   **Expected JSON Structure:**
        ```json
        {
          "task_description": "Submit the Q4 report",
          "due_date": "2023-11-10T12:00:00Z"
        }
        ```
5.  **LLM API Integration Module:** Validates and forwards the parsed data to the Nudge Logic module.
6.  **Nudge Logic Module:** Processes the task creation request.
    *   Saves the `task_description`, `due_date`, `telegram_user_id`, and `chat_id` to the PostgreSQL database with a status of `active`.
    *   Returns confirmation data to the Chatbot Integration module.
7.  **Chatbot Integration Module:** Sends confirmation message to the user.
    *   **Message Copy:** "✅ Got it! I'll remind you to: **Submit the Q4 report** on Friday, Nov 10 at 12:00 PM."

#### 6.3. Reminder & Snooze Flow
1.  **Trigger:** The Nudge Logic module's background scheduler finds a task whose `due_date` is now.
2.  **Nudge Logic Module:** Requests the Chatbot Integration module to send a reminder.
3.  **Chatbot Integration Module:** Sends a message to the user's chat.
    *   **Message Copy:** "🔔 Time for: **Submit the Q4 report**"
    *   **Inline Keyboard Buttons:**
        *   `✅ Done`
        *   `⏰ Snooze`
4.  **User Clicks `✅ Done`:** 
    *   Chatbot Integration module forwards the callback to Nudge Logic module.
    *   Nudge Logic module updates the task status to `completed` in the database.
    *   Chatbot Integration module edits the original message to "👍 Great work! Task completed."
5.  **User Clicks `⏰ Snooze`:** 
    *   Chatbot Integration module replaces the buttons with new snooze options.
    *   **Snooze Buttons:** `15 Minutes`, `1 Hour`, `Tomorrow Morning`
    *   When a snooze option is clicked, the Chatbot Integration module forwards the request to Nudge Logic module.
    *   Nudge Logic module updates the `due_date` for the task in the database.
    *   Chatbot Integration module replies: "Okay, I'll remind you again in [snooze duration]."
    *   `Tomorrow Morning` will be defined as 9:00 AM in the bot's configured timezone (initially UTC for MVP).

#### 6.4. The Nudge System
*   **Trigger:** The Nudge Logic module's background scheduler identifies an `active` task whose `due_date` is more than **2 hours** in the past and has not been snoozed or completed.
*   **Action:** Nudge Logic module requests the Chatbot Integration module to send a new, distinct message. This is a one-time nudge per task.
*   **Nudge Message Copy:** "Hey, just checking in. Were you able to get to this task? **Submit the Q4 report**"
*   **Inline Keyboard Buttons:**
    *   `✅ Yes, it's Done!` (Chatbot Integration forwards to Nudge Logic to mark task as `completed`)
    *   `⏰ Remind Me Later` (Chatbot Integration forwards to Nudge Logic which triggers the snooze options again)

#### 6.5. Task Management Commands
*   **Command:** `/list`
    *   **Action:** Chatbot Integration module forwards request to Nudge Logic module, which retrieves all `active` tasks for the user, sorted by `due_date`.
    *   **Output Format:**
        > "Here are your upcoming tasks:
        >
        > 1. Call Sarah - Due: Nov 4, 3:00 PM
        > 2. Submit the Q4 report - Due: Nov 10, 12:00 PM
        >
        > To manage a task, use `/done [number]` or `/delete [number]`."
*   **Command:** `/done [number]` (e.g., `/done 1`)
    *   **Action:** Chatbot Integration module forwards to Nudge Logic module, which marks the corresponding task as `completed`.
    *   **Reply:** "✅ Task 'Call Sarah' marked as done!"
*   **Command:** `/delete [number]`
    *   **Action:** Chatbot Integration module forwards to Nudge Logic module, which deletes the task from the database.
    *   **Reply:** "🗑️ Task 'Call Sarah' has been deleted."

#### 6.6. Error Handling
*   **Trigger:** The LLM API Integration module receives an error from the Gemma API or a JSON that doesn't contain a `task_description` or `due_date`.
*   **Action:** LLM API Integration module returns an error response to the Chatbot Integration module, which sends a helpful error message to the user.
*   **Message Copy:** "Sorry, I had trouble understanding that. Could you try phrasing it more simply, like `remind me to [task] on [date] at [time]`?"

### 7. Technical Architecture & Stack

#### 7.1. Modular Architecture Design

The NudgeBot system is built using a modular architecture consisting of three main modules that work together to provide a seamless task management experience:

##### **Chatbot Integration Module**
*   **Responsibility:** Handles all Telegram Bot API interactions, message routing, and user interface elements
*   **Key Functions:**
    *   Process incoming Telegram webhooks
    *   Send messages and inline keyboards to users
    *   Handle button callbacks and user interactions
    *   Route messages between other modules
*   **Interface:** Exposes REST endpoints for receiving webhooks and internal APIs for sending messages

##### **LLM API Integration Module**
*   **Responsibility:** Manages all interactions with external Language Learning Models for natural language processing
*   **Key Functions:**
    *   Parse natural language task descriptions
    *   Extract task details (description, due date) from user messages
    *   Handle API communication with Google Gemma
    *   Format and validate LLM responses
*   **Interface:** Provides task parsing endpoints that accept raw text and return structured task data

##### **Nudge Logic Module**
*   **Responsibility:** Core business logic for task management, scheduling, and nudge functionality
*   **Key Functions:**
    *   Task CRUD operations and state management
    *   Scheduling system for reminders and nudges
    *   Task completion and snooze logic
    *   Database operations and data persistence
*   **Interface:** Exposes task management APIs for creating, updating, and querying tasks

##### **Module Communication Flow**
```
User Message → Chatbot Integration → LLM API Integration → Nudge Logic → Database
                     ↓                        ↓                    ↓
User Response ← Chatbot Integration ← Parsed Task Data ← Task Created
```

#### 7.2. Technical Stack

*   **Platform:** Telegram Bot API
*   **Backend Language:** **Go (Golang)** - Chosen for performance and concurrency, essential for the scheduler and modular architecture.
*   **Web Framework:** **Gin** - A lightweight, high-performance web framework for Go, used across all modules.
*   **NLU Service:** **Google Gemma API** - All natural language parsing handled by the LLM API Integration module.
*   **Database:** **PostgreSQL** - To store user and task data, accessed through the Nudge Logic module.
*   **Scheduler:** A custom scheduler built with native **Go goroutines and channels** within the Nudge Logic module. A main goroutine will "tick" every minute, querying the DB and coordinating with the Chatbot Integration module for notifications.
*   **Inter-Module Communication:** HTTP/REST APIs for communication between modules, enabling loose coupling and independent scaling.
*   **Deployment:** Each module will be packaged in separate **Docker containers** and hosted on a cloud service like **Google Cloud Run** for independent scaling and deployment.

### 8. Scope: Out of MVP

The following features will NOT be included in this MVP but will be considered for future versions:
*   Recurring tasks (e.g., "every Tuesday").
*   Tasks without a specific due date ("someday" list).
*   Location-based reminders.
*   Editing existing tasks.
*   Timezone personalization for users.
*   File attachments or detailed notes.
*   Any form of monetization.