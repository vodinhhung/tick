# Tick — Technical Design

## Table of Contents
1. [Architecture Overview](#1-architecture-overview)
2. [Backend — Database Design](#2-backend--database-design)
3. [Backend — API Design](#3-backend--api-design)
4. [Frontend — Local Data (SwiftData)](#4-frontend--local-data-swiftdata)
5. [Frontend — Authentication Flow](#5-frontend--authentication-flow)
6. [Frontend — Non-Login vs Login User Workflow](#6-frontend--non-login-vs-login-user-workflow)
7. [Frontend — Sync Protocol](#7-frontend--sync-protocol)

---

## 1. Architecture Overview

```
iPhone App (SwiftUI + SwiftData)
        │
        │  HTTPS / JWT
        ▼
Go REST API (Gin / net-http)
        │
        │  GORM
        ▼
MySQL 8 Database
        │
(Google OAuth2 — ID Token validation)
```

- The iOS app is **local-first**: all reads/writes go to SwiftData first.
- When the user is authenticated and online, a **SyncManager** pushes local changes and pulls remote changes.
- The server validates the user's **Google ID Token** directly (no redirect flow needed for mobile) and issues a short-lived **JWT** the app stores in the iOS Keychain.

---

## 2. Backend — Database Design

### 2.1 `user_tab` (existing)

| Column         | Type                | Constraints                  |
|----------------|---------------------|------------------------------|
| id             | BIGINT UNSIGNED     | PK, AUTO_INCREMENT           |
| google_id      | VARCHAR(64)         | NOT NULL, UNIQUE             |
| email          | VARCHAR(255)        | NOT NULL, UNIQUE             |
| verified_email | TINYINT(1)          | DEFAULT 0                    |
| name           | VARCHAR(255)        |                              |
| given_name     | VARCHAR(128)        |                              |
| family_name    | VARCHAR(128)        |                              |
| picture        | VARCHAR(512)        |                              |
| locale         | VARCHAR(16)         |                              |
| created_at     | DATETIME(3)         | NOT NULL                     |
| updated_at     | DATETIME(3)         | NOT NULL                     |

**Indexes:** `UNIQUE(google_id)`, `UNIQUE(email)`, `INDEX(name)`

---

### 2.2 `category_tab`

Stores both preset categories (shared, `user_id IS NULL`) and user-created custom categories.

| Column     | Type            | Constraints                         |
|------------|-----------------|-------------------------------------|
| id         | BIGINT UNSIGNED | PK, AUTO_INCREMENT                  |
| user_id    | BIGINT UNSIGNED | NULL (presets) / FK → user_tab(id)  |
| name       | VARCHAR(100)    | NOT NULL                            |
| is_preset  | TINYINT(1)      | NOT NULL, DEFAULT 0                 |
| is_deleted | TINYINT(1)      | NOT NULL, DEFAULT 0                 |
| created_at | DATETIME(3)     | NOT NULL                            |
| updated_at | DATETIME(3)     | NOT NULL                            |

**Indexes:**
- `INDEX idx_category_user (user_id, is_deleted)` — list categories per user
- `INDEX idx_category_updated (user_id, updated_at)` — delta sync

**Notes:**
- Preset rows are seeded once (`user_id = NULL, is_preset = 1`). They are never modified via API.
- Custom categories are owned by a user. Soft-deleted with `is_deleted = 1`.

---

### 2.3 `habit_tab`

| Column          | Type              | Constraints                          |
|-----------------|-------------------|--------------------------------------|
| id              | BIGINT UNSIGNED   | PK, AUTO_INCREMENT                   |
| user_id         | BIGINT UNSIGNED   | NOT NULL, FK → user_tab(id)          |
| client_id       | VARCHAR(36)       | NOT NULL, UNIQUE — UUID from device  |
| name            | VARCHAR(255)      | NOT NULL                             |
| category_id     | BIGINT UNSIGNED   | NULL, FK → category_tab(id)          |
| frequency_type  | VARCHAR(32)       | NOT NULL — `'daily'` or `'weekly'`   |
| frequency_value | TINYINT UNSIGNED  | NOT NULL, DEFAULT 1                  |
| is_deleted      | TINYINT(1)        | NOT NULL, DEFAULT 0                  |
| created_at      | DATETIME(3)       | NOT NULL                             |
| updated_at      | DATETIME(3)       | NOT NULL                             |

**Indexes:**
- `INDEX idx_habit_user (user_id, is_deleted)` — list habits per user
- `UNIQUE INDEX idx_habit_client (client_id)` — idempotent upsert from client
- `INDEX idx_habit_sync (user_id, updated_at)` — delta sync

**Frequency extensibility:** `frequency_type` is a VARCHAR, not an ENUM, so new types can be added without schema migration. `frequency_value` meaning depends on type (for `'weekly'` it is the N count; for future `'monthly'` it could be day-of-month, etc.).

---

### 2.4 `habit_log_tab`

| Column       | Type            | Constraints                          |
|--------------|-----------------|--------------------------------------|
| id           | BIGINT UNSIGNED | PK, AUTO_INCREMENT                   |
| user_id      | BIGINT UNSIGNED | NOT NULL, FK → user_tab(id)          |
| habit_id     | BIGINT UNSIGNED | NOT NULL, FK → habit_tab(id)         |
| client_id    | VARCHAR(36)     | NOT NULL, UNIQUE — UUID from device  |
| completed_at | DATETIME(3)     | NOT NULL — stored as UTC             |
| note         | VARCHAR(280)    | NULL                                 |
| is_extra     | TINYINT(1)      | NOT NULL, DEFAULT 0                  |
| is_deleted   | TINYINT(1)      | NOT NULL, DEFAULT 0                  |
| created_at   | DATETIME(3)     | NOT NULL                             |
| updated_at   | DATETIME(3)     | NOT NULL                             |

**Indexes:**
- `INDEX idx_log_habit_date (habit_id, completed_at, is_deleted)` — heatmap/streak queries
- `INDEX idx_log_user_sync (user_id, updated_at)` — delta sync
- `UNIQUE INDEX idx_log_client (client_id)` — idempotent upsert

---

## 3. Backend — API Design

### Conventions

- Base path: `/api/v1`
- All requests/responses: `Content-Type: application/json`
- Authentication: `Authorization: Bearer <jwt>` (except auth endpoints)
- Timestamps: ISO 8601 UTC strings, e.g. `"2026-03-08T14:00:00Z"`
- Error body:
  ```json
  { "code": "ERROR_CODE", "message": "human readable message" }
  ```
- HTTP status codes: `200 OK`, `201 Created`, `204 No Content`, `400 Bad Request`, `401 Unauthorized`, `403 Forbidden`, `404 Not Found`, `409 Conflict`, `422 Unprocessable Entity`, `500 Internal Server Error`

---

### 3.1 Auth

#### `POST /api/v1/auth/google`

Exchange a Google ID Token (obtained natively on the device via Google Sign-In SDK) for a server-issued JWT.

**Request:**
```json
{
  "id_token": "eyJhbGci..."
}
```

**Response `200 OK`:**
```json
{
  "token": "eyJhbGci...",
  "expires_at": "2026-03-09T14:00:00Z",
  "user": {
    "id": 42,
    "email": "user@gmail.com",
    "name": "Jane Doe",
    "picture": "https://..."
  }
}
```

**Errors:**

| HTTP | Code | Reason |
|------|------|--------|
| 400  | `MISSING_ID_TOKEN` | `id_token` field absent |
| 401  | `INVALID_ID_TOKEN` | Token failed Google verification |
| 500  | `SERVER_ERROR` | DB or internal failure |

**Server logic:**
1. Verify `id_token` with Google's public keys (using `google.golang.org/api/idtoken`).
2. Extract `sub` (GoogleID), `email`, `name`, `picture`.
3. Upsert user in `user_tab` (existing `AddOrUpdateUser`).
4. Issue a signed JWT (`sub = user.id`, expiry = 24h).
5. Return token + user.

---

#### `POST /api/v1/auth/refresh`

Refresh a near-expiry JWT without re-authenticating with Google.

**Request:** _(empty body, valid JWT in Authorization header)_

**Response `200 OK`:**
```json
{
  "token": "eyJhbGci...",
  "expires_at": "2026-03-10T14:00:00Z"
}
```

**Errors:**

| HTTP | Code | Reason |
|------|------|--------|
| 401  | `TOKEN_EXPIRED` | JWT is already expired |
| 401  | `INVALID_TOKEN` | JWT signature invalid |

---

### 3.2 Categories

#### `GET /api/v1/categories`

Returns preset categories + user's custom categories.

**Query params:** none

**Response `200 OK`:**
```json
{
  "categories": [
    {
      "id": 1,
      "name": "Health",
      "is_preset": true,
      "is_deleted": false,
      "updated_at": "2026-01-01T00:00:00Z"
    },
    {
      "id": 101,
      "name": "Guitar Practice",
      "is_preset": false,
      "is_deleted": false,
      "updated_at": "2026-03-01T09:00:00Z"
    }
  ]
}
```

**Errors:** `401 UNAUTHORIZED`

---

#### `POST /api/v1/categories`

Create a custom category.

**Request:**
```json
{
  "name": "Guitar Practice"
}
```

**Response `201 Created`:**
```json
{
  "id": 101,
  "name": "Guitar Practice",
  "is_preset": false,
  "is_deleted": false,
  "created_at": "2026-03-08T10:00:00Z",
  "updated_at": "2026-03-08T10:00:00Z"
}
```

**Errors:**

| HTTP | Code | Reason |
|------|------|--------|
| 400  | `MISSING_NAME` | `name` is empty |
| 422  | `NAME_TOO_LONG` | `name` > 100 chars |
| 409  | `DUPLICATE_NAME` | User already has a category with this name |

---

#### `PUT /api/v1/categories/:id`

Rename a custom category. Cannot rename preset categories.

**Request:**
```json
{
  "name": "Music"
}
```

**Response `200 OK`:** _(same schema as POST response)_

**Errors:**

| HTTP | Code | Reason |
|------|------|--------|
| 403  | `PRESET_IMMUTABLE` | Attempt to rename a preset |
| 404  | `NOT_FOUND` | Category not found or not owned by user |
| 409  | `DUPLICATE_NAME` | Name collision |

---

#### `DELETE /api/v1/categories/:id`

Soft-delete a custom category. Habits in this category will have `category_id` set to NULL.

**Response `204 No Content`**

**Errors:**

| HTTP | Code | Reason |
|------|------|--------|
| 403  | `PRESET_IMMUTABLE` | Cannot delete preset |
| 404  | `NOT_FOUND` | Not found or not owned by user |

---

### 3.3 Habits

#### `GET /api/v1/habits`

List the user's habits. Supports delta sync via `updated_after`.

**Query params:**

| Param | Type | Description |
|-------|------|-------------|
| `updated_after` | ISO 8601 string | Return only records with `updated_at > updated_after`. Omit for full fetch. |
| `include_deleted` | bool | Default `false`. Set `true` during sync to receive soft-deleted records. |

**Response `200 OK`:**
```json
{
  "habits": [
    {
      "id": 10,
      "client_id": "uuid-from-device",
      "name": "Morning Run",
      "category_id": 1,
      "frequency_type": "daily",
      "frequency_value": 1,
      "is_deleted": false,
      "created_at": "2026-02-01T07:00:00Z",
      "updated_at": "2026-03-01T07:00:00Z"
    }
  ],
  "server_time": "2026-03-08T14:00:00Z"
}
```

`server_time` is returned so the client can use it as its next `updated_after` cursor.

---

#### `POST /api/v1/habits`

Create a habit. Idempotent via `client_id`.

**Request:**
```json
{
  "client_id": "uuid-generated-on-device",
  "name": "Morning Run",
  "category_id": 1,
  "frequency_type": "daily",
  "frequency_value": 1
}
```

**Response `201 Created`:** _(single habit object as above)_

If `client_id` already exists on the server, returns `200 OK` with the existing record (idempotent upsert).

**Errors:**

| HTTP | Code | Reason |
|------|------|--------|
| 400  | `MISSING_NAME` | `name` empty |
| 400  | `MISSING_CLIENT_ID` | `client_id` absent |
| 400  | `INVALID_FREQUENCY` | Unknown `frequency_type` or `frequency_value` out of range |
| 422  | `NAME_TOO_LONG` | `name` > 255 chars |

---

#### `PUT /api/v1/habits/:id`

Update a habit. `id` is the **server ID** (use `client_id` to find it on first sync).

**Request:** _(any subset of mutable fields + required `updated_at` for optimistic locking)_
```json
{
  "name": "Evening Run",
  "category_id": 2,
  "frequency_type": "weekly",
  "frequency_value": 3,
  "updated_at": "2026-03-08T06:00:00Z"
}
```

**Response `200 OK`:** _(full habit object)_

**Errors:**

| HTTP | Code | Reason |
|------|------|--------|
| 404  | `NOT_FOUND` | Habit not found or not owned by user |
| 409  | `CONFLICT` | Server `updated_at` is newer than request `updated_at` — client must re-fetch |

---

#### `DELETE /api/v1/habits/:id`

Soft-delete a habit (sets `is_deleted = 1`, cascades to logs).

**Response `204 No Content`**

**Errors:** `404 NOT_FOUND`

---

### 3.4 Habit Logs

#### `GET /api/v1/habits/:habit_id/logs`

Get completion logs for a habit. Supports date-range and delta sync.

**Query params:**

| Param | Type | Description |
|-------|------|-------------|
| `from` | ISO 8601 date | `completed_at >= from` (UTC date) |
| `to` | ISO 8601 date | `completed_at <= to` (UTC date) |
| `updated_after` | ISO 8601 | Delta sync cursor |
| `include_deleted` | bool | Default `false` |

**Response `200 OK`:**
```json
{
  "logs": [
    {
      "id": 500,
      "client_id": "uuid-from-device",
      "habit_id": 10,
      "completed_at": "2026-03-08T06:30:00Z",
      "note": "Felt great!",
      "is_extra": false,
      "is_deleted": false,
      "created_at": "2026-03-08T06:30:00Z",
      "updated_at": "2026-03-08T06:30:00Z"
    }
  ],
  "server_time": "2026-03-08T14:00:00Z"
}
```

---

#### `POST /api/v1/habits/:habit_id/logs`

Create a completion log. Idempotent via `client_id`.

**Request:**
```json
{
  "client_id": "uuid-from-device",
  "completed_at": "2026-03-08T06:30:00Z",
  "note": "Felt great!"
}
```

**Response `201 Created`:** _(single log object)_

Server sets `is_extra` based on existing log count for the same habit/day (or week for weekly habits).

**Errors:**

| HTTP | Code | Reason |
|------|------|--------|
| 400  | `MISSING_CLIENT_ID` | `client_id` absent |
| 400  | `MISSING_COMPLETED_AT` | `completed_at` absent |
| 422  | `NOTE_TOO_LONG` | `note` > 280 chars |
| 404  | `HABIT_NOT_FOUND` | `habit_id` not found or not owned |

---

#### `DELETE /api/v1/habits/:habit_id/logs/:log_id`

Soft-delete a single log entry.

**Response `204 No Content`**

**Errors:**

| HTTP | Code | Reason |
|------|------|--------|
| 404  | `NOT_FOUND` | Log not found or not owned by user |

---

### 3.5 Sync (Bulk)

#### `POST /api/v1/sync`

Single round-trip sync. The client pushes dirty local records and receives all server-side changes since its last sync cursor.

**Request:**
```json
{
  "last_synced_at": "2026-03-07T10:00:00Z",
  "categories": [
    {
      "client_id": "uuid",
      "name": "Music",
      "is_deleted": false,
      "updated_at": "2026-03-08T09:00:00Z"
    }
  ],
  "habits": [
    {
      "client_id": "uuid",
      "name": "Morning Run",
      "category_client_id": "uuid",
      "frequency_type": "daily",
      "frequency_value": 1,
      "is_deleted": false,
      "updated_at": "2026-03-08T07:00:00Z"
    }
  ],
  "logs": [
    {
      "client_id": "uuid",
      "habit_client_id": "uuid",
      "completed_at": "2026-03-08T06:30:00Z",
      "note": "Good run",
      "is_deleted": false,
      "updated_at": "2026-03-08T06:30:00Z"
    }
  ]
}
```

**Response `200 OK`:**
```json
{
  "server_time": "2026-03-08T14:05:00Z",
  "categories": [ /* server-side changes since last_synced_at */ ],
  "habits": [ /* server-side changes since last_synced_at */ ],
  "logs": [ /* server-side changes since last_synced_at */ ],
  "id_map": {
    "categories": { "client-uuid": 101 },
    "habits": { "client-uuid": 10 },
    "logs": { "client-uuid": 500 }
  }
}
```

`id_map` maps client UUIDs → server integer IDs so the client can update its local records with the authoritative server ID.

**Conflict resolution:** For records that exist on both sides, **server wins** (server `updated_at` takes precedence over the client's version).

**Errors:** `401 UNAUTHORIZED`, `500 SERVER_ERROR`

---

### 3.6 Stats (computed server-side, optional — for cross-device consistency)

#### `GET /api/v1/habits/:id/stats`

**Response `200 OK`:**
```json
{
  "habit_id": 10,
  "current_streak": 7,
  "longest_streak": 21,
  "completion_rate": 85.3,
  "total_completions": 64,
  "computed_at": "2026-03-08T14:00:00Z"
}
```

Stats are computed on-demand. In v1, streak/rate are also calculated locally on the device from SwiftData.

---

## 4. Frontend — Local Data (SwiftData)

### 4.1 Model Definitions

```swift
// Category.swift
@Model
class Category {
    @Attribute(.unique) var id: UUID          // client-generated
    var serverId: Int?                        // set after first sync
    var name: String
    var isPreset: Bool
    var isDeleted: Bool
    var createdAt: Date
    var updatedAt: Date
    var syncedAt: Date?                       // nil = never synced (dirty)

    @Relationship(deleteRule: .nullify, inverse: \Habit.category)
    var habits: [Habit] = []
}

// Habit.swift
@Model
class Habit {
    @Attribute(.unique) var id: UUID
    var serverId: Int?
    var name: String
    var category: Category?
    var frequencyType: String                 // "daily" | "weekly"
    var frequencyValue: Int
    var isDeleted: Bool
    var createdAt: Date
    var updatedAt: Date
    var syncedAt: Date?

    @Relationship(deleteRule: .cascade, inverse: \HabitLog.habit)
    var logs: [HabitLog] = []
}

// HabitLog.swift
@Model
class HabitLog {
    @Attribute(.unique) var id: UUID
    var serverId: Int?
    var habit: Habit
    var completedAt: Date
    var note: String?
    var isExtra: Bool
    var isDeleted: Bool
    var createdAt: Date
    var updatedAt: Date
    var syncedAt: Date?
}

// AppSettings.swift  (singleton — stored in UserDefaults, not SwiftData)
struct AppSettings {
    var reminderTime: DateComponents          // hour + minute
    var lastSyncedAt: Date?
    var jwtToken: String?                     // reference only; actual value in Keychain
}
```

### 4.2 Dirty Record Detection

A record is **dirty** (needs push to server) when `syncedAt == nil || updatedAt > syncedAt`.

All local writes must:
1. Set `updatedAt = Date.now`
2. Leave `syncedAt` unchanged (SyncManager sets it after a successful push)

### 4.3 Streak & Stats Computation (Local)

Stats are computed in-memory from SwiftData queries — no extra stored columns needed.

```
currentStreak(habit):
  if daily:
    walk backward from today; count consecutive days with ≥1 non-deleted log
  if weekly:
    walk backward from current week; count consecutive weeks with ≥N non-deleted logs

longestStreak(habit):
  compute over all logs, track max run

completionRate(habit):
  count distinct fulfilled periods / total periods since habit.createdAt * 100
```

---

## 5. Frontend — Authentication Flow

### 5.1 Libraries
- **Google Sign-In SDK for iOS** (`GoogleSignIn-iOS` via SPM) — handles the native OAuth flow, returns a `GIDGoogleUser` with an `idToken`.
- **iOS Keychain** (via `Security` framework or a thin wrapper like `KeychainAccess`) — stores the server JWT securely.

### 5.2 Sign-In Flow

```
App Launch
    │
    ▼
KeychainManager.loadToken()
    │
    ├─ Token found & not expired ──► AppState = .authenticated
    │                                SyncManager.syncIfNeeded()
    │
    └─ No token / expired ─────────► AppState = .guest
                                     (local-only mode)

User taps "Sign in with Google" (in Settings screen)
    │
    ▼
GIDSignIn.sharedInstance.signIn(withPresenting: rootVC)
    │
    ├─ Error ──► Show alert, remain in .guest
    │
    └─ Success (GIDGoogleUser)
            │
            ▼
        idToken = user.idToken.tokenString
            │
            ▼
        POST /api/v1/auth/google { id_token }
            │
            ├─ Error ──► Show alert
            │
            └─ 200 OK { token, expires_at, user }
                    │
                    ▼
                KeychainManager.saveToken(token, expiresAt)
                AppState = .authenticated
                    │
                    ▼
                SyncManager.initialSync()
                  - Pull all server data
                  - Merge with local (local dirty records pushed, server records pulled)
                  - Update AppState with merged data
```

### 5.3 Token Lifecycle

| Event | Action |
|-------|--------|
| JWT expires within 1 hour | Automatically call `POST /api/v1/auth/refresh` on next foreground |
| 401 received from any API call | Clear Keychain token, set `AppState = .guest`, show sign-in prompt |
| User taps "Sign Out" | Clear Keychain token, clear `GIDSignIn` session, set `AppState = .guest`, local data is **retained** |

### 5.4 Keychain Storage Schema

| Key | Value |
|-----|-------|
| `tick.jwt` | JWT string |
| `tick.jwt.expiry` | ISO 8601 expiry string |

---

## 6. Frontend — Non-Login vs Login User Workflow

### 6.1 AppState

```swift
enum AppState {
    case guest       // No JWT; local SwiftData only
    case authenticated(user: UserProfile)
}
```

The entire app is accessible in both states. The difference is data persistence scope and sync availability.

### 6.2 Non-Login (Guest) User Workflow

```
App Launch → AppState = .guest
    │
    ▼
Today View loads habits from local SwiftData
    │
    ├─ Create Habit → saved to SwiftData (serverId = nil, syncedAt = nil)
    ├─ Tick Habit  → HabitLog saved to SwiftData
    ├─ View Stats  → computed from local SwiftData
    └─ Settings    → shows "Sign in with Google" banner

No network calls are made in guest mode.
Local data persists indefinitely on device.
```

### 6.3 Login User Workflow

```
App Launch → token found → AppState = .authenticated
    │
    ▼
SyncManager.syncIfNeeded()        ← foreground trigger
    │
    ▼
Today View loads from local SwiftData (instant, no network wait)
    │
    ├─ Create Habit
    │       │
    │       ├─ Write to SwiftData (updatedAt = now, syncedAt = nil)
    │       └─ Notify SyncManager (debounced 5s) → POST /api/v1/sync
    │
    ├─ Tick Habit
    │       │
    │       ├─ Write HabitLog to SwiftData
    │       └─ Notify SyncManager (debounced 5s) → POST /api/v1/sync
    │
    ├─ Pull-to-Refresh → SyncManager.syncNow()
    │
    └─ App goes to background → cancel debounce, flush pending sync immediately
```

### 6.4 Guest → Login Migration

When a guest user signs in for the first time:

```
1. POST /api/v1/auth/google → receive JWT
2. SyncManager.initialSync():
   a. Collect ALL local records (categories, habits, logs)
      — all have serverId = nil
   b. Push them to server via POST /api/v1/sync
   c. Server creates them, returns id_map
   d. Update local records with serverId and syncedAt
   e. Pull any server records (none for a new account)
3. From this point, normal sync flow applies.
```

If the user signs into an **existing account** that already has server data:

```
1. Push local records (new ones the server doesn't have)
2. Pull server records
3. Merge:
   - If client_id collision (same UUID on both sides): server wins
   - If only on server: add to SwiftData
   - If only on client (serverId = nil): already pushed in step 1
```

---

## 7. Frontend — Sync Protocol

### 7.1 SyncManager (Singleton)

```swift
actor SyncManager {
    var isSyncing: Bool = false
    var lastSyncedAt: Date? { AppSettings.lastSyncedAt }

    func syncIfNeeded()     // called on foreground; skips if sync ran < 60s ago
    func syncNow()          // called on pull-to-refresh
    func scheduleDebouncedSync()  // called after any local write; fires after 5s idle
}
```

### 7.2 Sync Steps

```
1. Collect dirty records:
   categories = localDB.query(Category, where: syncedAt == nil || updatedAt > syncedAt)
   habits     = localDB.query(Habit,    where: syncedAt == nil || updatedAt > syncedAt)
   logs       = localDB.query(HabitLog, where: syncedAt == nil || updatedAt > syncedAt)

2. POST /api/v1/sync with { last_synced_at, categories, habits, logs }

3. On success response:
   a. Apply id_map: update serverId on local records
   b. Mark pushed records: syncedAt = serverTime
   c. For each returned server record:
      - Find by client_id or serverId
      - If server.updatedAt > local.updatedAt → overwrite local fields
      - If not found locally → insert into SwiftData
   d. AppSettings.lastSyncedAt = serverTime

4. Notify UI via Combine/AsyncStream to refresh if any records changed.
```

### 7.3 Offline Handling

- If the network request fails, dirty records remain (`syncedAt` unchanged).
- SyncManager retries on the next foreground trigger.
- No data is lost; local SwiftData is always the source of truth for the UI.

---

## 8. Push Notifications

- Use `UNUserNotificationCenter` with a `UNCalendarNotificationTrigger`.
- On each habit completion / app foreground, recompute pending count and reschedule the notification:
  ```swift
  // Schedule daily reminder at user-configured time
  let pendingCount = localDB.countPendingHabitsForToday()
  if pendingCount > 0 {
      scheduleNotification(at: reminderTime, body: "You have \(pendingCount) habit(s) still pending today.")
  } else {
      cancelPendingNotification()
  }
  ```
- Reminder time is stored in `UserDefaults` (no sync needed — it is device-local).
