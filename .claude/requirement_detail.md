# Tick — iPhone Habit Tracker: Full Product Requirements

## 1. Overview

**App Name:** Tick
**Platform:** iOS (iPhone)
**Architecture:** Local-first SwiftUI app with optional cloud sync via Go backend
**Auth:** Google OAuth2

Tick helps users build and maintain habits by allowing them to log daily completions, track streaks, and visualize progress over time.

---

## 2. Habit Management

### 2.1 Create Habit
- User can create a new habit with the following fields:
  - **Name** (required, string)
  - **Category** (optional, user-defined or from a preset list)
  - **Frequency** (required, see §2.4)
- Habits are saved locally via SwiftData immediately upon creation.

### 2.2 Edit Habit
- User can edit any field (name, category, frequency) of an existing habit.
- Changing frequency does not retroactively alter past completion logs.

### 2.3 Delete Habit
- User can delete a habit.
- Deleting a habit also removes all associated completion logs (cascade delete).
- A confirmation prompt is shown before deletion.

### 2.4 Frequency
- **Daily:** Habit must be completed once per calendar day.
- **N times per week:** Habit must be completed N times within a Monday–Sunday week (N ∈ 1..7).
- Data model is designed to be extensible for future frequency types (e.g., monthly, custom intervals) without schema breakage.

---

## 3. Habit Completion (Tick)

### 3.1 Logging a Completion
- Each habit has a "Tick" button on its card in the main list view.
- Tapping "Tick" records a completion log with:
  - `habit_id`
  - `completed_at` timestamp (device local time, stored as UTC)
  - `note` (optional, short text, e.g., max 280 characters)
- Multiple taps on the same day for a daily habit are allowed but only the first counts toward streak/completion rate (excess logs are stored but flagged as extra).
- For N-times-per-week habits, taps count up to N per week.

### 3.2 Undo / Remove Log
- User can remove the most recent completion log for a habit (undo last tick) from the habit detail view.
- Older logs can be removed from the calendar heatmap view (see §5.3).

### 3.3 Completion Note
- After tapping "Tick", an optional modal prompts for a short note.
- User can skip the note; it is never required.
- Notes are displayed in the habit detail/history view.

---

## 4. Categories

### 4.1 Category Assignment
- A habit belongs to at most one category.
- Default categories (presets): Health, Fitness, Mindfulness, Learning, Productivity, Other.
- Users can create custom categories (name only, no icon/color in v1).

### 4.2 Category Filtering
- Main list view supports filtering by category (show all / show one category).
- Habits can also be grouped by category in the list view (toggle option).

---

## 5. Stats & Streaks

### 5.1 Current Streak
- For **daily** habits: consecutive days (ending today or yesterday) with at least one completion.
- For **N times/week** habits: consecutive weeks (ending this or last week) where the habit was completed ≥ N times.
- Displayed on each habit card and in the habit detail view.

### 5.2 Longest Streak
- Historical maximum streak, computed from all completion logs.
- Displayed in the habit detail view.

### 5.3 Completion Rate
- Percentage of required periods (days or weeks) in which the habit was completed since its creation date (or a selected date range).
- Displayed in the habit detail view.
- Formula: `completed_periods / total_required_periods * 100`

### 5.4 Calendar Heatmap
- A monthly calendar view per habit showing completion intensity per day (0, partial, full).
- Color intensity reflects number of completions relative to the daily/weekly target.
- Tapping a day opens a detail popover showing completions and notes for that day.
- From this popover, individual completion logs can be deleted.

---

## 6. Reminders (Push Notifications)

### 6.1 Daily Reminder
- A single daily push notification reminds the user of habits that are not yet completed for the day.
- The notification fires at a user-configured time (default: 8:00 PM).
- Notification content: "You have X habit(s) still pending today."
- The app requests notification permission on first launch.

### 6.2 Scope (v1)
- One global reminder time applies to all habits.
- No per-habit reminder configuration in v1.
- Notification is suppressed if all habits for the day are already completed.

---

## 7. Data Storage

### 7.1 Local (SwiftData)
- All data is stored locally first; the app is fully functional offline.
- SwiftData models:
  - `Habit`: id, name, categoryId, frequencyType, frequencyValue, createdAt, updatedAt, syncedAt, isDeleted
  - `HabitLog`: id, habitId, completedAt, note, isExtra, syncedAt, isDeleted
  - `Category`: id, name, isCustom, createdAt

### 7.2 Cloud Sync (Go Backend + MySQL)
- Sync occurs when the user is authenticated (Google OAuth2) and the device is online.
- Sync strategy: **last-write-wins** with `updatedAt` timestamp comparison.
- Soft deletes (`isDeleted` flag) are used so deletions sync correctly.
- Sync is triggered:
  - On app foreground
  - After any local write (debounced, e.g., 5 seconds)
  - On explicit pull-to-refresh
- Conflict resolution: server timestamp wins for concurrent edits from multiple devices.

### 7.3 Authentication
- Google OAuth2 via the Go backend.
- Auth token stored in iOS Keychain.
- Unauthenticated users can use the app in full local-only mode.

---

## 8. Navigation & UI Structure

| Screen | Description |
|--------|-------------|
| Today View (Home) | List of all habits; shows completion status for today; Tick button per habit |
| Habit Detail | Stats (streak, longest streak, rate), heatmap, log history with notes |
| Add / Edit Habit | Form: name, category, frequency |
| Category Filter | Filter/group habits by category |
| Settings | Reminder time, Google sign-in/out, app info |

---

## 9. Out of Scope (v1)

- Social features (sharing, friends, leaderboards)
- Per-habit reminder times
- Widgets / Apple Watch support
- Multiple frequency types beyond daily and N times/week
- Habit reordering / priorities
- Dark/light theme toggle (follow system)

---

## 10. Non-Functional Requirements

- **Offline-first:** All core features work without network.
- **Performance:** Habit list and heatmap render within 300ms on iPhone 12 or newer.
- **Privacy:** No habit data is shared with third parties. Sync is only to the user's own account.
- **Extensibility:** Frequency model and sync protocol are versioned to allow future expansion without breaking existing clients.
