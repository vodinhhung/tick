# Tick — iOS Habit Tracker

Tick is a local-first habit tracker for iOS. Build streaks, log completions, and review your progress — all offline by default, with optional cloud sync via a Go backend.

## Features

- **Track habits** — daily or weekly habits with custom categories
- **One-tap logging** — mark a habit done, add optional notes
- **Stats & streaks** — current streak, longest streak, completion rate, calendar heatmap
- **Reminders** — daily push notifications
- **Local-first** — all data lives on-device via SwiftData; works fully offline
- **Cloud sync** — sign in with Google to sync across devices via the REST API

## Requirements

| Component | Requirement |
|-----------|------------|
| iOS App   | Xcode 15+, iOS 17+, Swift 5.9+ |
| Backend   | Go 1.21+, MySQL 8 |

## Installation & Running Locally

### 1. Backend

```bash
cd tick_be
```

Create the MySQL database:

```sql
CREATE DATABASE tick_db;
```

Edit `config/cfg/local.yaml` with your credentials:

```yaml
database:
  host: localhost
  port: 3306
  user: root
  password: your_password
  name: tick_db

jwt_secret: "your-secret-key"
google_client_id: "your-client-id.apps.googleusercontent.com"

server:
  port: 8080
```

Run the server:

```bash
go run ./cmd/main.go
```

The API will be available at `http://localhost:8080`.

### 2. iOS App

Open `tick_fe/tick.xcodeproj` in Xcode, select a simulator or device, and press **Run** (⌘R).

The app works fully offline without the backend. To enable sync, sign in with Google from the Settings screen — the app will connect to the backend URL configured in `APIClient.swift`.

## Project Structure

```
tick_be/    # Go REST API (net/http, GORM, MySQL)
tick_fe/    # SwiftUI iOS app (SwiftData, local-first)
```

See [CLAUDE.md](CLAUDE.md) for a detailed breakdown of the architecture, API routes, and sync protocol.

## License

MIT License

Copyright (c) 2026 Vo Hung

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
