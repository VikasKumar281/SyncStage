# SyncStage

**SyncStage** is a real-time multi-display media sequencing platform built with **React and Go**.

It allows multiple displays to run their own independent playlists while sharing a common time-based playback system. A central control interface can update playlists in real time or temporarily take over every connected display with the same media.

The project focuses on deterministic playback, reliable synchronization, real-time state updates, and simple production deployment.

## Live Demo

**Live Application:** https://syncstage.onrender.com/

**Repository:** https://github.com/VikasKumar281/SyncStage

---

## What SyncStage Does

SyncStage is designed around a simple idea:

> Every display should know what it is supposed to be playing based on the current time, its playlist, and the shared playback cycle.

For example, different displays can have completely different playlists:

```text
Lobby
M1 → M2 → M4 → repeat

Reception
M3 → Blank → M2 → M6 → repeat

Cafeteria
M5 → M1 → repeat

Corridor
M6 → M3 → M2 → M4 → Blank → repeat
```

At any moment, each display independently resolves its current media.

The operator can also trigger a temporary global synchronization:

```text
             Global Sync
                  │
        ┌─────────┼─────────┐
        ▼         ▼         ▼
      Lobby   Reception  Cafeteria
        │         │         │
        └─────────┼─────────┘
                  ▼
            Same Media
                  │
             Duration
                  │
                  ▼
          Normal playback
             resumes
```

---

## Features

### Multi-Display Playback

- Multiple display windows
- Independent playlist for every display
- Continuous playback
- Shared five-hour playback cycle
- Image, video, and blank media
- Configurable item durations
- Playback progress and remaining time
- Individual display URLs

### Playlist Management

- Create media
- Create display windows
- Add media to playlists
- Remove playlist items
- Update playlists while displays are running
- Persist changes
- Broadcast changes to connected displays

### Global Synchronization

- Temporarily override all displays
- Select the media to synchronize
- Configure synchronization duration
- Schedule synchronization using a future server timestamp
- Synchronize displays using server-adjusted time
- Automatically return to each display's normal timeline

### Real-Time Updates

- Server-Sent Events (SSE)
- Live state updates
- Playlist updates without page refresh
- Synchronization events
- Cycle reset events
- Automatic browser SSE reconnection

### Persistence

- PostgreSQL support
- Neon PostgreSQL in production
- JSONB state persistence
- Automatic database initialization
- Seed data for a ready-to-use demo
- Local JSON storage fallback

### Production Ready Setup

- React production build
- Go backend
- Multi-stage Docker build
- Single-container deployment
- Render deployment configuration
- Health check endpoint
- Bundled demo video files

---

# Technology Stack

| Area | Technology |
|---|---|
| Frontend | React |
| Build Tool | Vite |
| Language | JavaScript |
| Backend | Go |
| API | REST |
| Real-Time Communication | Server-Sent Events |
| Database | PostgreSQL |
| Production Database | Neon |
| Persistence Format | JSONB |
| Containerization | Docker |
| Deployment | Render |
| Source Control | Git / GitHub |

---

# Architecture

SyncStage uses a client/server architecture.

```text
                         ┌──────────────────┐
                         │   Control UI      │
                         │      React        │
                         └────────┬─────────┘
                                  │
                                  │ REST
                                  │
                         ┌────────▼─────────┐
                         │    Go Backend    │
                         │                  │
                         │ REST API         │
                         │ Scheduler        │
                         │ Timeline         │
                         │ SSE Hub          │
                         │ Storage Layer    │
                         └────────┬─────────┘
                                  │
                                  │
                         ┌────────▼─────────┐
                         │ PostgreSQL /     │
                         │ Neon             │
                         └──────────────────┘
```

Display windows connect to the same backend:

```text
Display 1 ─┐
Display 2 ─┤
Display 3 ─┼──→ Go Backend ──→ PostgreSQL
Display 4 ─┘
```

The backend is responsible for the authoritative application state.

The frontend is responsible for rendering that state and calculating the current playback position locally.

---

# Production Architecture

The production application runs as a single Docker service.

```text
GitHub
   │
   ▼
Render
   │
   ▼
Docker Container
   ├── Go Backend
   ├── React Production Build
   └── Bundled Media
   │
   ▼
Neon PostgreSQL
```

This keeps the production setup simple:

```text
Frontend
   +
API
   +
SSE
   +
Static Media
```

are served from the same application.

---

# Project Structure

```text
SyncStage/
│
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   │
│   ├── internal/
│   │   ├── api/
│   │   ├── config/
│   │   ├── models/
│   │   ├── scheduler/
│   │   ├── seed/
│   │   └── storage/
│   │       ├── jsonstore/
│   │       └── pgstore/
│   │
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
│
├── frontend/
│   ├── public/
│   │   ├── Logo.png
│   │   └── videos/
│   │       ├── m4.mp4
│   │       └── m5.mp4
│   │
│   ├── src/
│   │   ├── components/
│   │   ├── lib/
│   │   └── App.jsx
│   │
│   ├── scripts/
│   │   └── verify-parity.mjs
│   ├── package.json
│   └── vite.config.js
│
├── Dockerfile.allinone
├── docker-compose.yml
├── render.yaml
├── README.md
└── docs/
    └── ARCHITECTURE.md
```

The main playback implementations are:

```text
backend/internal/scheduler/scheduler.go
frontend/src/lib/timeline.js
```

---

# Core Playback Model

The most important part of SyncStage is the time-based playback model.

Playback is not driven by a sequence of commands such as:

```text
Play M1
Wait
Play M2
Wait
Play M4
```

Instead, the current item is calculated from time:

```text
Shared Cycle Anchor
        +
Server-Adjusted Current Time
        +
Window Playlist
        ↓
Timeline Position
        ↓
Current Media
```

Conceptually:

```text
resolveSequence(window, currentTime)
```

returns the media that should currently be visible.

This makes playback deterministic.

If a display refreshes or reconnects, it does not need to start from the beginning of the playlist. It can calculate where it should currently be.

---

# Five-Hour Playback Cycle

The normal playback cycle is five hours.

```text
5 hours
= 5 × 60 × 60 seconds
= 18,000 seconds
= 18,000,000 milliseconds
```

The production configuration is:

```text
CYCLE_SECONDS=18000
```

All displays use the same cycle reference.

However, their playlists remain independent.

```text
                  Shared Cycle
                       │
        ┌──────────────┼──────────────┐
        ▼              ▼              ▼
     Window 1       Window 2       Window 3
     M1 M2 M4       M3 B M2        M5 M1
```

---

# Playlist Looping

A playlist can be much shorter than five hours.

For example:

```text
M1 = 8 seconds
M2 = 8 seconds
M4 = 15 seconds
```

Total:

```text
31 seconds
```

The playlist therefore loops:

```text
M1 → M2 → M4
     ↓
M1 → M2 → M4
     ↓
M1 → M2 → M4
     ↓
...
```

The same logic continues throughout the five-hour cycle.

---

# Cycle Boundary

The five-hour boundary is treated as a hard boundary.

If the current media extends beyond the end of the cycle, only the remaining portion of that media is played.

Example:

```text
Current media
     │
     │ 10 seconds normally
     │
     ├─────────────┐
     │             │
     ▼             ▼
  3 seconds    cycle boundary
                  │
                  ▼
              new cycle
                  │
                  ▼
              item 0
```

No additional blank time is automatically inserted.

At the beginning of the next cycle, playback starts again from the first playlist item.

---

# Blank Media

Blank is a first-class media type.

Example:

```text
M1 → M2 → Blank → M4
```

The blank item is shown for its configured duration.

An empty playlist is different from a playlist containing an explicit blank item.

This distinction allows a display to intentionally show blank content without confusing it with a missing playlist.

---

# Global Sync Takeover

The synchronization feature temporarily overrides normal playback on every display.

Suppose the operator selects:

```text
Media: M2
Duration: 10 seconds
```

All connected displays switch to:

```text
M2
```

for the configured duration.

```text
Window 1 ──┐
Window 2 ──┤
Window 3 ──┼──→ M2
Window 4 ──┘
```

When synchronization expires, each display returns to its own normal timeline.

The normal playlists are never replaced by the sync media.

---

# Future-Timestamp Synchronization

An immediate synchronization command is not enough for multiple browsers.

If the backend simply said:

```text
"Start now"
```

each browser could receive the message at a different time.

For example:

```text
Display 1 → receives at +80ms
Display 2 → receives at +130ms
Display 3 → receives at +210ms
Display 4 → receives at +270ms
```

Instead, SyncStage schedules the sync for a future timestamp.

The backend normally uses:

```text
SYNC_LEAD_MS=1200
```

This gives connected displays time to receive the event.

Each display receives the same target timestamp and waits until its server-adjusted clock reaches that point.

---

# Clock Synchronization

Client machines may have different system clocks.

For example:

```text
Computer A → 12:00:00.000
Computer B → 11:59:59.700
Computer C → 12:00:00.250
```

Using the local clock directly could therefore produce synchronization errors.

SyncStage uses:

```http
GET /api/time
```

to estimate the difference between the browser clock and the server clock.

The frontend uses multiple time probes and selects a low-latency sample.

The basic idea is:

```text
offset =
serverTime -
(clientSendTime + roundTripTime / 2)
```

The browser can then estimate:

```text
serverNow =
Date.now() + offset
```

This gives all connected displays a common time reference.

---

# Sync Lifecycle

The complete sync lifecycle is:

```text
Operator selects media
        │
        ▼
Operator selects duration
        │
        ▼
POST /api/sync
        │
        ▼
Backend calculates future start time
        │
        ▼
SSE event is broadcast
        │
        ▼
All displays receive the event
        │
        ▼
Displays wait for target timestamp
        │
        ▼
Global sync becomes active
        │
        ▼
Selected media is shown
        │
        ▼
Duration expires
        │
        ▼
Each display resolves its normal timeline
```

---

# Returning From Sync

SyncStage does not store and restore an old playback position.

Instead, the normal timeline continues logically underneath the sync state.

For example:

```text
Normal timeline:

M1 → M2 → M4 → M1 → M2 → ...

                ↓
             Sync starts

                ↓

SYNC MEDIA

                ↓
            Sync ends

                ↓

Resolve normal timeline
at the current time
```

This means each display resumes whatever it should be showing at that moment.

This approach avoids manually tracking a separate playback position for every display.

---

# Server-Sent Events

SyncStage uses Server-Sent Events for real-time server-to-browser communication.

Endpoint:

```http
GET /api/events
```

The browser connects using:

```javascript
new EventSource("/api/events")
```

The backend can then push state changes to all connected clients.

Typical events include:

```text
snapshot
playlist.updated
sync.scheduled
sync.cancelled
media.created
window.created
cycle.reset
```

---

# Why SSE?

Most real-time communication in SyncStage is:

```text
Server → Browser
```

Operator actions are handled through normal REST requests:

```text
POST
DELETE
```

SSE is a natural fit because it provides:

- native browser support
- simple HTTP-based communication
- automatic browser reconnection
- straightforward server-to-client events
- less protocol complexity than a full WebSocket layer

---

# Dynamic Playlist Updates

Playlists can be changed while displays are already running.

Example:

```text
Before:

M1 → M2 → M4
```

The operator adds M6:

```text
After:

M1 → M2 → M4 → M6
```

The backend:

```text
1. Validates the request
2. Loads the current state
3. Applies the playlist change
4. Persists the state
5. Broadcasts the updated state
```

Connected displays receive the change through SSE.

There is no need to reload every display manually.

---

# Video Playback

Video playback needs special handling because a display may connect after a video has already started.

The video component therefore calculates the expected elapsed time from the timeline.

The player:

- loads the selected video
- waits for metadata when necessary
- calculates the expected playback position
- seeks toward that position
- starts playback
- uses muted inline playback for browser autoplay compatibility
- reports loading errors to the console

This helps keep video playback aligned with the shared timeline.

---

# Bundled Demo Videos

The demo videos are stored locally:

```text
frontend/public/videos/m4.mp4
frontend/public/videos/m5.mp4
```

They are included in the production Docker image.

The backend serves them as:

```text
/videos/m4.mp4
/videos/m5.mp4
```

This makes the core video demonstration independent of third-party video hosting.

User-created media can still reference external URLs.

---

# Detached Displays

Individual displays can be opened separately using the `window` query parameter.

Example:

```text
https://syncstage.onrender.com/?window=w1
```

This makes it possible to simulate multiple physical screens using separate browser windows.

Example:

```text
Browser Window 1 → Window 1
Browser Window 2 → Window 2
Browser Window 3 → Window 3
Browser Window 4 → Window 4
```

A global synchronization event can then be observed across all connected displays.

---

# Persistence

Production uses PostgreSQL through Neon.

The storage layer is abstracted from the application logic.

The PostgreSQL implementation stores the application state as JSONB.

The main table is:

```sql
CREATE TABLE IF NOT EXISTS sequencer_state (
    id INTEGER PRIMARY KEY,
    state JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

The persisted state contains the application's:

- media
- display windows
- playlists
- cycle information
- synchronization state

---

# PostgreSQL Startup

When `DATABASE_URL` is configured, the backend:

```text
1. Opens the PostgreSQL connection pool
2. Verifies the database connection
3. Creates the state table if necessary
4. Loads existing state
5. Seeds initial data if no state exists
6. Applies bundled video URL migration
7. Starts the HTTP server
```

This allows the production service to initialize itself without a separate database migration command for the initial state.

---

# Local Storage Fallback

For simple local development, PostgreSQL is optional.

If:

```text
DATABASE_URL
```

is not configured, the application falls back to local JSON storage.

The default local path is:

```text
data/state.json
```

This makes it possible to run the project locally without setting up a database.

---

# Seed Data

SyncStage starts with sample media and display windows so the application can be explored immediately.

The seeded data includes:

```text
Images
Videos
Blank media
```

and multiple display playlists.

Example:

```text
Lobby
M1 → M2 → M4

Reception
M3 → Blank → M2 → M6

Cafeteria
M5 → M1

Corridor
M6 → M3 → M2 → M4 → Blank
```

This provides enough variety to demonstrate:

- independent playlists
- playlist looping
- video playback
- blank slots
- dynamic updates
- global synchronization

---

# API

All API endpoints are exposed by the Go backend.

Time values are represented as Unix epoch milliseconds.

## Health

```http
GET /api/health
```

Used by the deployment platform and for service monitoring.

---

## Server Time

```http
GET /api/time
```

Returns the current server timestamp.

Used for frontend clock calibration.

---

## State

```http
GET /api/state
```

Returns the complete application state.

---

## Media

```http
GET /api/media
```

Returns the media library.

---

## Create Media

```http
POST /api/media
```

Example:

```json
{
  "name": "Product Image",
  "type": "image",
  "url": "https://example.com/image.jpg",
  "defaultDurationSeconds": 10
}
```

Supported types:

```text
image
video
blank
```

---

## Windows

```http
GET /api/windows
```

Returns all display windows and their playlists.

---

## Create Window

```http
POST /api/windows
```

Example:

```json
{
  "name": "Window 5"
}
```

---

## Current Playback

```http
GET /api/windows/{id}/now
```

Returns the media currently resolved for a display.

---

## Add Playlist Item

```http
POST /api/windows/{id}/playlist
```

Example:

```json
{
  "mediaId": "m2",
  "durationSeconds": 8
}
```

---

## Remove Playlist Item

```http
DELETE /api/windows/{id}/playlist/{itemId}
```

Successful deletion returns:

```text
204 No Content
```

---

## Start Sync

```http
POST /api/sync
```

Example:

```json
{
  "mediaId": "m2",
  "durationSeconds": 10
}
```

The backend schedules the synchronization using a future timestamp.

---

## Cancel Sync

```http
DELETE /api/sync
```

Cancels the active or scheduled synchronization.

---

## Reset Cycle

```http
POST /api/cycle/reset
```

Resets the shared cycle anchor.

---

## Events

```http
GET /api/events
```

Opens the SSE stream for real-time updates.

---

# Local Development

## Requirements

Install:

- Go 1.25+
- Node.js
- npm

Optional:

- Docker
- PostgreSQL

---

## 1. Clone the Repository

```bash
git clone https://github.com/VikasKumar281/SyncStage.git
cd SyncStage
```

---

## 2. Start the Backend

Open a terminal:

```bash
cd backend
go run ./cmd/server
```

The backend runs on:

```text
http://localhost:8080
```

---

## 3. Start the Frontend

Open another terminal:

```bash
cd frontend
npm install
npm run dev
```

The development frontend runs on:

```text
http://localhost:5173
```

---

## 4. Open the Application

Go to:

```text
http://localhost:5173
```

The Vite development server proxies API requests to the Go backend.

---

# Local PostgreSQL

PostgreSQL is optional during local development.

If you want to use PostgreSQL, configure:

```text
DATABASE_URL=<your-postgresql-connection-string>
```

Then start the backend normally:

```bash
cd backend
go run ./cmd/server
```

The backend automatically initializes the required state table.

Do not commit database credentials to Git.

---

# Environment Variables

| Variable | Default | Purpose |
|---|---:|---|
| `PORT` | `8080` | Backend HTTP port |
| `DATABASE_URL` | empty | PostgreSQL connection string |
| `DATA_PATH` | `data/state.json` | Local JSON storage path |
| `CYCLE_SECONDS` | `18000` | Playback cycle duration |
| `SYNC_LEAD_MS` | `1200` | Lead time before sync |
| `SYNC_DEFAULT_SECONDS` | `10` | Default sync duration |
| `ALLOWED_ORIGINS` | `*` | Allowed browser origins |
| `STATIC_DIR` | empty | Production frontend directory |

---

# Testing With a Short Cycle

The production cycle is:

```text
18000 seconds
```

which is five hours.

For faster local testing, it can be reduced.

Example:

```text
CYCLE_SECONDS=120
```

This creates a two-minute cycle.

The playback logic remains the same while making cycle-boundary behavior much easier to verify locally.

---

# Docker

SyncStage includes a production-oriented multi-stage Docker build.

The main file is:

```text
Dockerfile.allinone
```

The build process is approximately:

```text
Node build
    ↓
React production bundle
    ↓
Go dependency/build stage
    ↓
Go tests + validation
    ↓
Minimal runtime image
    ↓
Go server + React build + media
```

---

# Build the Docker Image

Run from the repository root:

```bash
docker build -f Dockerfile.allinone -t syncstage .
```

---

# Run the Docker Container

```bash
docker run --rm -p 8080:8080 syncstage
```

Then open:

```text
http://localhost:8080
```

If port `8080` is already being used:

```bash
docker run --rm -p 8081:8080 syncstage
```

Then open:

```text
http://localhost:8081
```

---

# Verify Bundled Videos

When running on port `8080`:

```text
http://localhost:8080/videos/m4.mp4
http://localhost:8080/videos/m5.mp4
```

When running on port `8081`:

```text
http://localhost:8081/videos/m4.mp4
http://localhost:8081/videos/m5.mp4
```

Browsers may request video content using HTTP range requests, so a successful video response can appear as:

```text
206 Partial Content
```

---

# Docker Compose

The repository also includes:

```text
docker-compose.yml
```

Start the application:

```bash
docker compose up --build
```

Stop it:

```bash
docker compose down
```

---

# Production Deployment

The application is deployed using:

```text
Render
```

with:

```text
Neon PostgreSQL
```

The live application is:

```text
https://syncstage.onrender.com/
```

---

# Render Configuration

The repository contains:

```text
render.yaml
```

The service uses the all-in-one Docker image.

Important settings include:

```yaml
services:
  - type: web
    name: syncstage
    runtime: docker
    dockerfilePath: ./Dockerfile.allinone
    dockerContext: .
    plan: free
    healthCheckPath: /api/health

    envVars:
      - key: DATABASE_URL
        sync: false

      - key: CYCLE_SECONDS
        value: "18000"

      - key: ALLOWED_ORIGINS
        value: "*"
```

The PostgreSQL connection string is configured through the Render environment and points to Neon.

Secrets are not stored in the repository.

---

# Deployment Flow

```text
Git push
   │
   ▼
GitHub
   │
   ▼
Render
   │
   ▼
Docker Build
   │
   ├── React build
   ├── Go validation
   ├── Go tests
   └── Production image
   │
   ▼
SyncStage Service
   │
   ▼
Neon PostgreSQL
```

---

# Health Check

The service exposes:

```http
GET /api/health
```

The endpoint is used to verify that the backend is running correctly.

Example:

```text
https://syncstage.onrender.com/api/health
```

---

# Timeline Parity

The playback timeline is implemented in both the backend and frontend.

Backend:

```text
backend/internal/scheduler/scheduler.go
```

Frontend:

```text
frontend/src/lib/timeline.js
```

This is necessary because:

- the backend needs to resolve playback state
- the frontend needs to render playback locally

Both implementations need to produce the same result for the same input.

The project includes a parity verification script:

```text
frontend/scripts/verify-parity.mjs
```

Run it with:

```bash
cd frontend
node scripts/verify-parity.mjs http://localhost:8080
```

---

# Testing

Backend validation:

```bash
cd backend
go vet ./...
```

Run tests:

```bash
go test ./... -count=1
```

Frontend production build:

```bash
cd frontend
npm run build
```

Docker build:

```bash
docker build -f Dockerfile.allinone -t syncstage .
```

---

# What Is Tested

The backend test suite covers important timeline and state-management behavior such as:

- playlist traversal
- playlist looping
- five-hour cycle behavior
- cycle boundary truncation
- timestamps before the cycle anchor
- active synchronization
- synchronization release
- expired synchronization
- adding playlist items
- deleting playlist items
- unknown media IDs
- unknown window IDs
- state persistence
- CORS behavior

---

# Manual Testing

## Independent Playback

Open multiple display windows.

Verify that:

```text
Window 1
```

follows its own playlist while:

```text
Window 2
```

continues following its own playlist.

The displays should not affect each other's normal playback.

---

## Playlist Update

1. Select a display.
2. Add a media item.
3. Confirm the playlist changes.
4. Check another connected display.
5. Confirm the updated state reaches connected clients.

---

## Playlist Removal

1. Select an existing playlist item.
2. Remove it.
3. Confirm it disappears from the playlist.
4. Confirm connected clients receive the new state.

---

## Global Sync

1. Open multiple displays.
2. Select sync media.
3. Select a duration.
4. Start synchronization.
5. Confirm all displays show the same media.
6. Wait for the configured duration.
7. Confirm every display returns to its normal timeline.

---

## Detached Displays

Open individual display URLs such as:

```text
/?window=w1
/?window=w2
/?window=w3
```

Place them in separate browser windows.

Trigger global synchronization and verify that all displays respond to the same sync event.

---

## Persistence

1. Change a playlist.
2. Refresh the browser.
3. Confirm the change remains.

In production, this verifies persistence through PostgreSQL/Neon.

---

## Video Playback

Verify:

```text
M4
M5
```

load successfully.

The bundled files should be available through:

```text
/videos/m4.mp4
/videos/m5.mp4
```

---

# Useful API Checks

Health:

```bash
curl http://localhost:8080/api/health
```

Server time:

```bash
curl http://localhost:8080/api/time
```

State:

```bash
curl http://localhost:8080/api/state
```

Windows:

```bash
curl http://localhost:8080/api/windows
```

Media:

```bash
curl http://localhost:8080/api/media
```

Current playback:

```bash
curl http://localhost:8080/api/windows/w1/now
```

SSE stream:

```bash
curl -N http://localhost:8080/api/events
```

---

# Example Sync Request

Start an eight-second global sync:

```bash
curl -X POST http://localhost:8080/api/sync \
  -H "Content-Type: application/json" \
  -d "{\"mediaId\":\"m2\",\"durationSeconds\":8}"
```

Then inspect the current state:

```bash
curl http://localhost:8080/api/state
```

The displays should switch to the selected media when the scheduled timestamp is reached.

---

# Design Decisions

## Time-Based Playback

A time-based model was chosen instead of a command-driven playback model.

This means a display can calculate:

```text
What should I show right now?
```

without needing the backend to send a command for every media transition.

Benefits:

- deterministic playback
- refresh recovery
- reconnect recovery
- late display joining
- simpler synchronization
- fewer real-time messages

---

## Future Timestamp for Sync

Synchronization is scheduled slightly in the future instead of immediately.

This gives clients time to receive the event before the target moment.

It also means every client works toward the same timestamp rather than reacting at different network arrival times.

---

## Server Clock

The backend provides the authoritative time reference.

Clients estimate their offset from the backend rather than trusting their own system clock.

This makes synchronization more consistent across different devices.

---

## SSE for Live State

SSE keeps the real-time communication simple.

REST is used for actions:

```text
Create
Update
Delete
Sync
Reset
```

SSE is used for:

```text
State changes
Playlist changes
Sync events
```

---

## PostgreSQL JSONB

The application state is relatively small and naturally represented as a single state object.

JSONB provides:

- persistence
- straightforward serialization
- simple loading
- simple updates
- PostgreSQL reliability

The storage interface also keeps the application logic separate from the database implementation.

---

## Single-Container Deployment

The React frontend and Go backend are packaged together.

This provides:

- one deployment
- one public origin
- simple API routing
- simple SSE routing
- simple static media serving
- fewer moving parts

---

# Assumptions

### Repeating Five-Hour Cycle

Playback continues through repeating five-hour cycles.

### Hard Cycle Boundary

An item crossing the cycle boundary is truncated and the next cycle starts from the beginning of the playlist.

### Explicit Blank

Blank playback only occurs when a blank item exists in a playlist.

### Independent Normal Playlists

Each display keeps its own normal playlist.

### Temporary Sync Override

Global synchronization temporarily overrides normal playback but does not modify the underlying playlists.

### Normal Timeline Continues

The normal timeline remains time-based while synchronization is active.

### Media Uses URLs

Media records reference URLs.

The application does not currently provide a complete media upload and transcoding pipeline.

---

# Known Limitations

## Single Backend Instance

The current SSE hub is in memory and is designed for a single backend instance.

Scaling horizontally would require a shared event mechanism.

Possible future solutions include:

```text
Redis
Message Broker
Pub/Sub
Distributed Event Bus
```

---

## External Media

User-created media may point to external URLs.

If an external media server is unavailable, the browser cannot load that media.

The bundled demo videos avoid this dependency for the main demonstration.

---

## Browser Autoplay

Videos are muted and configured for inline playback to improve autoplay compatibility.

Browser policies can still vary between devices and browsers.

---

## Network Conditions

Future-timestamp synchronization reduces the impact of network latency but cannot completely eliminate network failures or severely delayed clients.

---

## Media Upload

A complete upload pipeline is not currently included.

The project does not handle:

```text
Upload
Transcoding
Media CDN
Object Storage
Automatic Optimization
```

for arbitrary user media.

---

# Future Improvements

Some natural next steps for SyncStage would be:

- Authentication
- Role-based access
- Media upload
- Object storage
- Media validation
- Media caching
- Display health monitoring
- Offline playback
- Better unavailable-media handling
- Drag-and-drop playlist ordering
- Scheduled future synchronization
- Audit logs
- Display status monitoring
- Distributed SSE infrastructure
- Redis/pub-sub support
- End-to-end browser testing
- Application metrics and monitoring

The current architecture leaves room for these additions without changing the core time-based playback model.

---

# Development Commands

## Backend

Run:

```bash
cd backend
go run ./cmd/server
```

Format:

```bash
gofmt -w .
```

Validate:

```bash
go vet ./...
```

Test:

```bash
go test ./... -count=1
```

---

## Frontend

Install dependencies:

```bash
cd frontend
npm install
```

Start development server:

```bash
npm run dev
```

Create production build:

```bash
npm run build
```

---

## Docker

Build:

```bash
docker build -f Dockerfile.allinone -t syncstage .
```

Run:

```bash
docker run --rm -p 8080:8080 syncstage
```

Alternative port:

```bash
docker run --rm -p 8081:8080 syncstage
```

---

## Docker Compose

Start:

```bash
docker compose up --build
```

Stop:

```bash
docker compose down
```

---

# Quick Architecture Reference

## Normal Playback

```text
Server Clock
     │
     ▼
Cycle Anchor
     │
     ▼
Current Cycle Position
     │
     ▼
Window Playlist
     │
     ▼
Timeline Resolver
     │
     ▼
Current Media
     │
     ▼
MediaSurface
```

## Playlist Update

```text
Operator
   │
   ▼
REST API
   │
   ▼
Validate
   │
   ▼
Update State
   │
   ▼
Persist
   │
   ▼
SSE Broadcast
   │
   ▼
Connected Displays
```

## Global Sync

```text
Operator
   │
   ▼
POST /api/sync
   │
   ▼
Future startAtMs
   │
   ▼
SSE Broadcast
   │
   ▼
All Displays
   │
   ▼
Server-Adjusted Clock
   │
   ▼
Sync Media
   │
   ▼
Duration Ends
   │
   ▼
Normal Timeline
```

---

# Repository

The project source is available on GitHub:

https://github.com/VikasKumar281/SyncStage

The live application is available at:

https://syncstage.onrender.com/

---

# Closing

SyncStage is built around a simple principle:

> **Playback should be predictable, synchronization should be time-based, and display state should stay consistent in real time.**

The combination of a deterministic timeline, server-clock calibration, future-timestamp synchronization, SSE updates, and persistent state makes the system suitable as a foundation for multi-screen digital signage and synchronized media playback.

