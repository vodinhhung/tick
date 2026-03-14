# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Tick** is a local-first iOS habit tracker with optional cloud sync. It consists of:
- `tick_fe/` — SwiftUI/SwiftData iOS app (iOS 17+)
- `tick_be/` — Go REST API backend

## Backend Commands

All commands run from `tick_be/`:

```bash
# Run the server
go run ./cmd/main.go

# Build
go build ./...

# Run tests
go test ./...

# Run a single test
go test ./internal/handler/... -run TestFunctionName
```

The server starts on port 8080 by default, configured via `config/cfg/local.yaml`.

## Frontend Commands

The iOS app is built with Xcode. Open `tick_fe/tick.xcodeproj` in Xcode. Tests are in `tickTests/` (unit) and `tickUITests/` (UI).

## Backend Configuration

`tick_be/config/cfg/local.yaml` must be configured before running:

```yaml
database:
  host: localhost
  port: 3306
  user: root
  password: password
  name: tick_db

jwt_secret: "change-me-in-production-use-a-long-random-string"
google_client_id: "your-google-client-id.apps.googleusercontent.com"

server:
  port: 8080
```

## Architecture

### Data Flow

```
iPhone App (SwiftUI + SwiftData) ←→ Go REST API (net/http) ←→ MySQL 8
                                          ↑
                                  Google OAuth2 validation
```

### Backend (`tick_be/`)

```
cmd/main.go              # Entry point
config/                  # Viper-based config loader
api/router.go            # HTTP route registration (net/http, no Gin)
internal/
  handler/               # HTTP handlers: auth, habit, habit_log, category, stats, sync
  middleware/jwt.go      # JWT auth middleware
  auth/google_auth.go    # Google ID token verification
  model/models.go        # GORM models for all DB tables
  database/database.go   # DB initialization
  dep/user_tab.go        # User repository (data access layer)
```

All API routes are under `/api/v1`. Authentication endpoints are public; all others require JWT.

### Frontend (`tick_fe/tick/`)

```
tickApp.swift            # App entry point, SwiftData ModelContainer setup
ContentView.swift        # Root navigation
Models/                  # SwiftData models: Habit, Category, HabitLog
Views/                   # Screens: TodayView, HabitDetailView, AddEditHabitView, etc.
Services/
  AuthManager.swift      # Google OAuth + JWT token management
  APIClient.swift        # HTTP client for backend API
  SyncManager.swift      # Local↔Server sync orchestration
  KeychainManager.swift  # Secure JWT storage
  NotificationManager.swift # Daily reminder notifications
  StatsCalculator.swift  # Streak and completion rate computations
```

### Key Design Decisions

**Local-first sync:** SwiftData is the source of truth on-device. The app works fully offline. Sync is triggered on foreground, after writes (debounced 5s), and on pull-to-refresh.

**Dirty tracking:** A record is "dirty" (needs sync) when `syncedAt == nil || updatedAt > syncedAt`. The sync endpoint (`POST /api/v1/sync`) pushes dirty records and pulls server changes in one round-trip.

**Soft deletes:** All tables use an `is_deleted` flag instead of hard deletes to preserve sync history.

**Idempotent upserts:** The iOS app generates UUIDs (`client_id`) for every record. The server upserts on `client_id`, making retries safe.

**Conflict resolution:** Last-write-wins based on `updated_at` timestamp.

**Auth flow:** Google ID token → `POST /api/v1/auth/google` → JWT (HS256, 24h expiry) stored in iOS Keychain. Refresh via `POST /api/v1/auth/refresh`.

**Guest mode:** The app runs fully without authentication; Google sign-in enables cloud sync.

## API Routes

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/google` | Exchange Google ID token for JWT |
| POST | `/api/v1/auth/refresh` | Refresh JWT |
| GET/POST | `/api/v1/categories` | List or create categories |
| PUT/DELETE | `/api/v1/categories/{id}` | Update or soft-delete category |
| GET/POST | `/api/v1/habits` | List (with `updated_after` delta sync) or create habit |
| PUT/DELETE | `/api/v1/habits/{id}` | Update or soft-delete habit |
| GET/POST | `/api/v1/habits/{habit_id}/logs` | List or create completion logs |
| DELETE | `/api/v1/habits/{habit_id}/logs/{log_id}` | Delete log |
| GET | `/api/v1/habits/{id}/stats` | Streak and completion rate |
| POST | `/api/v1/sync` | Bulk sync (push + pull) |

## Reference Docs

Detailed product requirements and API specifications are in `.claude/`:
- `.claude/requirement_detail.md` — Full product requirements
- `.claude/technical_design.md` — Technical architecture, DB schema, API request/response examples
