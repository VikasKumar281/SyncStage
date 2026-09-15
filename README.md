# SyncStage

> Real-time multi-display media sequencing and synchronized playback built with React and Go.

SyncStage is a digital-signage style application where multiple display windows run independent playlists while an operator can update playlists and temporarily synchronize the same media across all connected displays.

The core playback model is **time-based and deterministic**: displays calculate their expected playback position from a shared cycle anchor, the current server-adjusted time, and their own playlist. Synchronization is implemented as a temporary global override using a future server timestamp.

## Live Demo

**Application:** https://syncstage.onrender.com

**GitHub:** https://github.com/VikasKumar281/SyncStage

---

## Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Technology Stack](#technology-stack)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Playback Model](#playback-model)
- [Five-Hour Cycle](#five-hour-cycle)
- [Playlist Looping](#playlist-looping)
- [Blank Media](#blank-media)
- [Global Sync Takeover](#global-sync-takeover)
- [Clock Synchronization](#clock-synchronization)
- [Real-Time Communication](#real-time-communication)
- [Dynamic Playlist Updates](#dynamic-playlist-updates)
- [Video Playback](#video-playback)
- [Detached Displays](#detached-displays)
- [Persistence](#persistence)
- [Seed Data](#seed-data)
- [API Reference](#api-reference)
- [Local Development](#local-development)
- [Environment Variables](#environment-variables)
- [Docker](#docker)
- [Docker Compose](#docker-compose)
- [Production Deployment](#production-deployment)
- [Testing](#testing)
- [Manual Verification](#manual-verification)
- [Design Decisions](#design-decisions)
- [Assumptions](#assumptions)
- [Known Limitations](#known-limitations)
- [Future Improvements](#future-improvements)
- [Submission Checklist](#submission-checklist)

---

# Overview

SyncStage simulates a small multi-display media system.

Each display has an independent playlist:

```text
Window 1 — Lobby
M1 → M2 → M4 → repeat

Window 2 — Reception
M3 → Blank → M2 → M6 → repeat

Window 3 — Cafeteria
M5 → M1 → repeat

Window 4 — Corridor
M6 → M3 → M2 → M4 → Blank → repeat
```

The displays share the same five-hour cycle reference, but each window resolves its own playlist independently.

An operator can:

- view multiple display windows
- add media to playlists
- remove media from playlists
- create media
- create display windows
- trigger a global synchronization takeover
- select the synchronization media
- configure synchronization duration
- reset the playback cycle
- open individual displays separately

---

# Key Features

### Display & Playback

- Multiple display windows
- Independent playlist per window
- Continuous playlist playback
- Five-hour repeating cycle
- Image media
- Video media
- Explicit blank media
- Configurable media duration
- Current media indicator
- Remaining-time display
- Playback progress
- Next-media information

### Playlist Management

- Add media to a playlist
- Remove media from a playlist
- Modify playlists while playback is running
- Persist playlist changes
- Push updates to connected displays through SSE

### Synchronization

- Global sync takeover
- Selectable sync media
- Configurable sync duration
- Future-timestamp synchronization
- Server/client clock calibration
- Temporary sync override
- Automatic return to each window's normal timeline

### Real-Time Communication

- Server-Sent Events (SSE)
- Automatic browser `EventSource` reconnection
- State snapshots
- Live playlist updates
- Live synchronization events
- Cycle reset events

### Persistence

- PostgreSQL in production
- Neon PostgreSQL for deployment
- JSONB state storage
- Automatic database table creation
- Seed data on first startup
- Local JSON fallback for development

### Deployment

- Multi-stage Docker build
- React production build
- Go backend
- Single-container deployment
- Render deployment
- Neon PostgreSQL
- Health-check endpoint

---

# Technology Stack

| Layer | Technology |
|---|---|
| Frontend | React, Vite, JavaScript, CSS |
| Backend | Go |
| API | REST |
| Real-time | Server-Sent Events |
| Database | PostgreSQL / JSONB |
| Production DB | Neon PostgreSQL |
| Containerization | Docker |
| Deployment | Render |
| Source Control | GitHub |

---

# Architecture

SyncStage follows a simple client/server architecture.

```text
                         ┌─────────────────────┐
                         │      Browser 1      │
                         │    React Display    │
                         └──────────┬──────────┘
                                    │
                                    │ REST + SSE
                                    │
                         ┌──────────▼──────────┐
                         │     Go Backend      │
                         │                     │
                         │  REST API           │
                         │  Scheduler          │
                         │  SSE Hub            │
                         │  Storage Layer      │
                         └──────────┬──────────┘
                                    │
                                    │
                         ┌──────────▼──────────┐
                         │   PostgreSQL/Neon   │
                         └─────────────────────┘
```

Multiple displays connect to the same backend:

```text
Browser 1 ─┐
Browser 2 ─┤
Browser 3 ─┼──→ Go Backend ──→ PostgreSQL
Browser 4 ─┘
```

The backend owns the authoritative application state.

The browser receives that state and calculates the current playback position locally using the shared timeline and server-adjusted time.

---

# Production Architecture

The production application is packaged into a single Docker container.

```text
GitHub
   ↓
Render
   ↓
Docker Image
   ├── React production build
   ├── Go server
   └── Bundled video assets
   ↓
SyncStage
   ↓
Neon PostgreSQL
```

The single-container architecture means the browser uses one origin for:

```text
Frontend
API
SSE
Bundled videos
```

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

Important playback implementations:

```text
backend/internal/scheduler/scheduler.go
frontend/src/lib/timeline.js
```

---

# Playback Model

The most important design decision is that playback is **time-based**, not command-driven.

Conceptually:

```text
resolve(window, currentTime)
```

determines:

- current media
- playlist index
- cycle index
- item start time
- item end time
- remaining time
- playback source

Example:

```text
M1 = 8 seconds
M2 = 8 seconds
M4 = 15 seconds
```

Timeline:

```text
0s        8s        16s                  31s
│---------│----------│--------------------│
    M1         M2             M4
```

At 5 seconds:

```text
M1
```

At 12 seconds:

```text
M2
```

At 20 seconds:

```text
M4
```

The browser does not need a server command for every media transition.

Instead:

```text
shared cycle anchor
        +
server-adjusted current time
        +
window playlist
        ↓
current media
```

This allows a display to recover its expected position after refreshes or reconnections.

---

# Five-Hour Cycle

The required normal playback cycle is five hours.

```text
5 × 60 × 60 × 1000
=
18,000,000 milliseconds
```

Production configuration:

```text
CYCLE_SECONDS=18000
```

All windows share the same cycle reference:

```text
                 Shared 5-hour cycle
                         │
          ┌──────────────┼──────────────┐
          ▼              ▼              ▼
       Window 1       Window 2       Window 3
       M1 M2 M4       M3 B M2 M6     M5 M1
```

The cycle is shared, while playlists remain independent.

---

# Playlist Looping

A playlist does not need to be exactly five hours long.

Example:

```text
M1 = 8 seconds
M2 = 8 seconds
M4 = 15 seconds
```

Total playlist duration:

```text
31 seconds
```

The playlist loops:

```text
M1 → M2 → M4 → M1 → M2 → M4 → ...
```

The current position is resolved from the position within the shared five-hour cycle.

---

# Five-Hour Boundary

If an item crosses the five-hour boundary, it is truncated at the boundary.

Example:

```text
Current media has 10 seconds remaining.

Only 3 seconds remain in the five-hour cycle.
```

Playback becomes:

```text
Current media
     │
     ├── 3 seconds
     │
     ▼
5-hour boundary
     │
     ▼
New cycle
     │
     ▼
Playlist item 0
```

No automatic blank padding is inserted.

---

# Blank Media

Blank is an explicit media type.

Example:

```text
M3 → Blank → M2
```

The blank item is displayed for its configured duration.

Blank playback occurs only when a blank item is explicitly included in a playlist.

An empty playlist is treated separately and does not automatically become a blank playlist.

---

# Global Sync Takeover

Sync Takeover temporarily overrides the normal playback of all windows.

Example:

```text
Selected media: M2
Duration: 10 seconds
```

During the takeover:

```text
Window 1 → M2
Window 2 → M2
Window 3 → M2
Window 4 → M2
```

After the duration expires:

```text
Window 1 → normal sequence
Window 2 → normal sequence
Window 3 → normal sequence
Window 4 → normal sequence
```

The normal playlists are not replaced.

---

# Why Future-Timestamp Sync?

An immediate command such as:

```text
"Play M2 now"
```

would not arrive at exactly the same time on every browser.

For example:

```text
Window 1 → +80ms
Window 2 → +130ms
Window 3 → +210ms
Window 4 → +270ms
```

Instead, SyncStage schedules the takeover for a future server timestamp.

The default lead time is:

```text
SYNC_LEAD_MS=1200
```

So the backend normally schedules synchronization approximately 1.2 seconds in the future.

Each client receives the same target timestamp and switches when its server-adjusted clock reaches that timestamp.

---

# Clock Synchronization

Different machines can have different local clocks.

Example:

```text
Machine A → 12:00:00.000
Machine B → 11:59:59.700
Machine C → 12:00:00.250
```

The frontend therefore calls:

```http
GET /api/time
```

to estimate the server clock.

The approximate calculation is:

```text
offsetMs =
serverTimeMs -
(sentAt + roundTripMs / 2)
```

Then:

```text
serverNow() =
Date.now() + offsetMs
```

The frontend periodically recalibrates the clock offset.

Multiple probes are used because network latency can vary.

---

# Returning From Sync

SyncStage does not use:

```text
save current position
        ↓
show sync media
        ↓
restore old position
```

Instead, the normal timeline continues logically underneath the sync.

Example:

```text
Normal Window 1:
M1 → M2 → M4 → M1 → ...

Sync starts:
SYNC → M6

Sync ends:
    ↓
resolve normal timeline again
    ↓
show whatever Window 1 should display now
```

This means every display automatically resumes its correct sequence without manually storing a playback position.

---

# Real-Time Communication

SyncStage uses Server-Sent Events for server-to-browser updates.

Endpoint:

```http
GET /api/events
```

The frontend uses the browser's native:

```javascript
EventSource
```

API.

Important events include:

```text
snapshot
playlist.updated
sync.scheduled
sync.cancelled
media.created
window.created
cycle.reset
```

A newly connected client receives the current snapshot.

---

# Why SSE Instead of WebSockets?

The main live communication pattern is:

```text
Server → Browser
```

Operator actions use normal REST requests:

```text
POST
DELETE
```

SSE is therefore a simple fit because it provides:

- native browser support
- normal HTTP
- automatic `EventSource` reconnect behavior
- simple server-to-client events
- less protocol complexity than full WebSockets

---

# Dynamic Playlist Updates

Playlist changes can happen while playback is running.

Example:

```text
Before:

M1 → M2 → M4
```

After adding M6:

```text
M1 → M2 → M4 → M6
```

The backend:

```text
1. validates the request
2. loads current state
3. applies the mutation
4. persists the updated state
5. broadcasts the new state through SSE
```

Connected displays can update without a full page reload.

---

# Video Playback

Images can be displayed directly.

Videos require additional handling because the timeline may indicate that a video should already be part-way through its duration.

The video player therefore:

- loads the selected video
- waits for metadata when necessary
- calculates elapsed playback time
- seeks toward the expected timeline position
- starts playback
- keeps video muted and inline for browser autoplay compatibility

This helps after:

- page refresh
- late display connection
- timeline changes
- sync release

## Bundled Demo Videos

The seeded video files are stored locally:

```text
frontend/public/videos/m4.mp4
frontend/public/videos/m5.mp4
```

They are included in the Docker image and served by the Go server:

```text
/videos/m4.mp4
/videos/m5.mp4
```

Therefore the core demo does not depend on an external video host.

User-created media can still reference external URLs.

---

# Detached Displays

Individual displays can be opened separately using the `window` query parameter.

Example:

```text
https://syncstage.onrender.com/?window=w1
```

This allows a realistic multi-display setup:

```text
Browser Window 1 → Window 1
Browser Window 2 → Window 2
Browser Window 3 → Window 3
Browser Window 4 → Window 4
```

The operator can then trigger a global sync and observe the synchronized takeover across displays.

---

# Persistence

Production uses PostgreSQL through Neon.

The backend has a storage abstraction so the application logic is not tightly coupled to a specific storage implementation.

The PostgreSQL implementation stores the application state as JSONB.

The required table is:

```sql
CREATE TABLE IF NOT EXISTS sequencer_state (
    id INTEGER PRIMARY KEY,
    state JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

The persisted state includes the application's media, windows, playlists, cycle information, and active synchronization state.

---

# PostgreSQL Startup

When `DATABASE_URL` is configured:

1. PostgreSQL storage is opened.
2. The connection is checked.
3. The required table is created if necessary.
4. Existing state is loaded.
5. Seed data is written if no state exists.
6. Video URL migration is applied for the bundled demo videos.

This makes the production service self-initializing.

---

# Local JSON Fallback

When:

```text
DATABASE_URL
```

is not configured, the application can use local JSON storage.

Default path:

```text
data/state.json
```

This is useful for local development without PostgreSQL.

Production uses Neon PostgreSQL.

---

# Seed Data

The application includes demonstration data:

```text
7 media items
4 display windows
```

The seeded media demonstrates:

```text
Image
Video
Blank
```

Example playlists:

```text
Window 1 — Lobby
M1 → M2 → M4

Window 2 — Reception
M3 → Blank → M2 → M6

Window 3 — Cafeteria
M5 → M1

Window 4 — Corridor
M6 → M3 → M2 → M4 → Blank
```

This gives the application enough variation to test independent playback, videos, blank slots, playlist updates, and global synchronization.

---

# API Reference

All API endpoints are served by the Go backend.

Time values are Unix epoch milliseconds.

## Health

```http
GET /api/health
```

Returns service information such as:

```text
status
cycle duration
server time
SSE client count
uptime
```

---

## Server Time

```http
GET /api/time
```

Returns the current server timestamp.

Used by the frontend for clock calibration.

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

Supported media types:

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

Returns display windows and their playlists.

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

## Current Window Playback

```http
GET /api/windows/{id}/now
```

Returns the current playback calculation for a window.

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

The backend calculates a future start timestamp using the configured lead time.

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

## SSE

```http
GET /api/events
```

Opens the real-time event stream.

---

# API Error Format

Validation and lookup errors use JSON.

Example:

```json
{
  "error": "media not found"
}
```

Typical status codes:

```text
200 → success
201 → resource created
204 → successful deletion
400 → invalid request
404 → resource not found
```

---

# Local Development

## Requirements

Install:

- Go 1.25+
- Node.js
- npm
- Docker (optional)

---

## Step 1 — Clone

```bash
git clone https://github.com/VikasKumar281/SyncStage.git
cd SyncStage
```

---

## Step 2 — Start Backend

Open terminal 1:

```bash
cd backend
go run ./cmd/server
```

Backend:

```text
http://localhost:8080
```

---

## Step 3 — Start Frontend

Open terminal 2:

```bash
cd frontend
npm install
npm run dev
```

Frontend:

```text
http://localhost:5173
```

During development, Vite proxies API requests to the backend.

---

## Step 4 — Optional PostgreSQL

Set:

```text
DATABASE_URL=<your PostgreSQL connection string>
```

Then:

```bash
cd backend
go run ./cmd/server
```

The backend will use PostgreSQL instead of JSON storage.

---

## Step 5 — Run Without PostgreSQL

Simply leave `DATABASE_URL` unset.

The backend uses:

```text
data/state.json
```

for local persistence.

---

# Environment Variables

| Variable | Default | Description |
|---|---:|---|
| `PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | empty | PostgreSQL connection string |
| `DATA_PATH` | `data/state.json` | Local JSON storage path |
| `CYCLE_SECONDS` | `18000` | Five-hour cycle |
| `SYNC_LEAD_MS` | `1200` | Lead time before sync starts |
| `SYNC_DEFAULT_SECONDS` | `10` | Default sync duration |
| `ALLOWED_ORIGINS` | `*` | Allowed browser origins |
| `STATIC_DIR` | empty | React production directory |

## Five-Hour Cycle

Production:

```text
CYCLE_SECONDS=18000
```

For fast local testing:

```text
CYCLE_SECONDS=120
```

This creates a two-minute cycle while preserving the same timeline behavior.

## Sync Configuration

Default lead time:

```text
SYNC_LEAD_MS=1200
```

Default duration:

```text
SYNC_DEFAULT_SECONDS=10
```

The UI can configure the actual synchronization duration.

## Frontend Configuration

```text
VITE_API_BASE_URL
```

For the current single-container deployment this can remain empty so the browser uses the same origin.

---

# Docker

The main production Dockerfile is:

```text
Dockerfile.allinone
```

It uses multiple build stages.

## Stage 1 — Frontend

```text
Node
  ↓
npm install
  ↓
npm run build
  ↓
frontend/dist
```

## Stage 2 — Backend

```text
Go
  ↓
download dependencies
  ↓
go vet ./...
  ↓
go test ./...
  ↓
build server
```

## Stage 3 — Runtime

The final image contains:

```text
Go server
React production build
Bundled video files
```

Node and the Go compiler are not required in the final runtime image.

---

# Build Docker Image

From the repository root:

```bash
docker build -f Dockerfile.allinone -t syncstage .
```

---

# Run Docker Container

Standard:

```bash
docker run --rm -p 8080:8080 syncstage
```

Open:

```text
http://localhost:8080
```

If port `8080` is already occupied:

```bash
docker run --rm -p 8081:8080 syncstage
```

Open:

```text
http://localhost:8081
```

---

# Verify Bundled Videos

With the application running on port 8080:

```text
http://localhost:8080/videos/m4.mp4
http://localhost:8080/videos/m5.mp4
```

With port 8081:

```text
http://localhost:8081/videos/m4.mp4
http://localhost:8081/videos/m5.mp4
```

The browser may request video ranges and receive:

```text
206 Partial Content
```

which is expected for media range requests.

---

# Docker Compose

The repository also contains:

```text
docker-compose.yml
```

Start:

```bash
docker compose up --build
```

Open:

```text
http://localhost:8080
```

Stop:

```bash
docker compose down
```

---

# Production Deployment

Current deployment:

```text
GitHub
   ↓
Render
   ↓
Docker
   ↓
SyncStage
   ↓
Neon PostgreSQL
```

Live application:

```text
https://syncstage.onrender.com
```

---

# Render Configuration

The repository contains:

```text
render.yaml
```

Important configuration:

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

The production `DATABASE_URL` is configured in Render and points to the Neon PostgreSQL database.

Never commit the database connection string to Git.

---

# Production Deployment Flow

```text
Developer pushes code
        ↓
GitHub
        ↓
Render detects commit
        ↓
Docker build
        ↓
React build
        ↓
Go vet
        ↓
Go tests
        ↓
Go production build
        ↓
Final Docker image
        ↓
Render Web Service
        ↓
SyncStage
        ↓
Neon PostgreSQL
```

---

# Render Health Check

Render uses:

```http
GET /api/health
```

A healthy service returns a response containing:

```json
{
  "status": "ok"
}
```

along with additional service information.

---

# Testing

Run backend validation:

```bash
cd backend
go vet ./...
go test ./... -count=1
```

The tests cover important behavior including:

- playlist walking
- playlist looping
- five-hour cycle boundary
- boundary truncation
- clock-before-anchor behavior
- sync override
- sync release
- expired sync
- playlist item addition
- playlist item deletion
- unknown media IDs
- unknown window IDs
- persistence
- CORS behavior

---

# Timeline Parity

Playback logic exists in both:

```text
Go:
backend/internal/scheduler/scheduler.go
```

and:

```text
JavaScript:
frontend/src/lib/timeline.js
```

The browser needs its own implementation because it must calculate playback locally.

Having two implementations creates a potential divergence risk.

For example:

```text
Go says:
M2

JavaScript says:
M1
```

would cause the backend and display to disagree.

The project therefore includes a parity verification script:

```text
frontend/scripts/verify-parity.mjs
```

Run it against a running backend:

```bash
cd frontend
node scripts/verify-parity.mjs http://localhost:8080
```

---

# Manual Verification

## 1. Independent Playback

Open the application and verify that each window follows its own playlist.

Expected behavior:

```text
Window 1 → its playlist
Window 2 → its playlist
Window 3 → its playlist
Window 4 → its playlist
```

---

## 2. Playlist Update

1. Select a display.
2. Add a media item.
3. Confirm the playlist changes.
4. Observe another connected display.
5. Confirm the state update arrives without a full page reload.

---

## 3. Playlist Removal

1. Select an existing playlist item.
2. Remove it.
3. Confirm it disappears.
4. Confirm connected clients receive the updated state.

---

## 4. Global Sync

1. Open multiple displays.
2. Select sync media.
3. Choose a duration.
4. Start sync.
5. Confirm all displays switch to the selected media.
6. Confirm the sync state is visible.

---

## 5. Sync Release

Wait for the selected duration.

After sync expires:

```text
Window 1 → normal sequence
Window 2 → normal sequence
Window 3 → normal sequence
Window 4 → normal sequence
```

---

## 6. Detached Displays

1. Open a display using `Detach`.
2. Open multiple detached windows.
3. Trigger sync from the control interface.
4. Confirm all displays respond to the same scheduled synchronization.

---

## 7. Persistence

1. Modify a playlist.
2. Refresh the application.
3. Confirm the change remains.

In production this verifies PostgreSQL/Neon persistence.

---

## 8. Health

Open:

```text
https://syncstage.onrender.com/api/health
```

Confirm:

```text
status = ok
```

---

## 9. Server Time

Open:

```text
https://syncstage.onrender.com/api/time
```

Confirm a server timestamp is returned.

---

## 10. Bundled Videos

Open the video URLs or observe M4/M5 in the application.

Confirm that both videos load without depending on an external video host.

---

# Design Decisions

## 1. Time-Based Playback

Playback is calculated from:

```text
current time
+
cycle anchor
+
playlist
```

Benefits:

- refresh recovery
- late display joining
- reduced network dependency
- deterministic playback
- natural synchronization support

Tradeoff:

The playback algorithm exists in both Go and JavaScript, so parity testing is important.

---

## 2. Future-Timestamp Synchronization

Sync uses a scheduled future timestamp instead of an immediate command.

Benefits:

- clients have time to receive the event
- every client targets the same timestamp
- normal network latency has less impact on the switching moment

Tradeoff:

No software solution can guarantee mathematically perfect synchronization across arbitrary networks.

---

## 3. Server Clock Calibration

The browser estimates its offset from the backend server clock.

This prevents different client system clocks from directly determining sync timing.

---

## 4. SSE Instead of WebSockets

Server-to-browser communication is the primary real-time requirement.

SSE provides a simpler implementation for:

```text
state updates
playlist changes
sync events
cycle events
```

while REST handles operator actions.

---

## 5. PostgreSQL JSONB

The application state is a relatively small aggregate and does not require complex relational queries.

JSONB provides a straightforward persistent representation.

---

## 6. Single Container Deployment

React and Go are deployed together.

Benefits:

- simple deployment
- one public origin
- simple frontend/API routing
- simple SSE routing
- bundled demo videos
- fewer production services

---

# Assumptions

## Five Hours Is a Repeating Cycle

The five-hour period is a repeating cycle:

```text
Cycle 0
  ↓
Cycle 1
  ↓
Cycle 2
  ↓
...
```

Playback continues indefinitely.

## Cycle Boundary Truncates the Current Item

If an item crosses the five-hour boundary, it is truncated.

The next cycle starts from playlist item zero.

## Blank Is Explicit

Blank playback occurs only when a blank item is explicitly included.

Unused cycle time is not automatically converted into blank playback.

## Sync Media Is Independent of Normal Playlists

A sync media item does not have to be present in every window's normal playlist.

## Normal Playback Continues Under Sync

Sync is a temporary display override.

The normal timeline remains logically time-based underneath it.

## Media Uses URLs

Media is represented using metadata and a URL.

The application does not implement a full media upload/transcoding pipeline.

The seeded demo videos are bundled locally for reliable demonstration.

---

# Known Limitations

## Single Backend Instance

The current architecture is designed for a single backend instance.

The SSE connection hub is stored in memory.

Horizontal scaling would require a shared event broker or distributed messaging layer.

## External User Media

User-created media may reference external URLs.

If an external media host is unavailable, the browser cannot display that media.

The application does not currently download and cache all external user media.

## Browser Autoplay Policies

Videos are muted and played inline to support normal browser autoplay restrictions.

Browsers may still apply their own policies.

## Network Conditions

Timestamp-based synchronization reduces the effect of latency but cannot eliminate network failures.

A severely delayed client may receive a sync event after its scheduled start.

## Media Upload

There is currently no complete upload, validation, storage, or transcoding pipeline for user media.

---

# Future Improvements

Possible production-scale improvements include:

- Authentication and role-based access
- Media upload and object storage
- Media validation
- Distributed SSE/event broker
- Redis or another shared event system
- Display health monitoring
- Offline media caching
- Better unavailable-media handling
- Drag-and-drop playlist ordering
- Scheduled future sync events
- Audit logs
- Persistent per-display configuration
- Detailed monitoring and metrics
- Automated browser end-to-end tests
- Richer media management UI

These are outside the core scope of the current implementation.

---

# Useful Commands

## Backend

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

## Frontend

Install:

```bash
cd frontend
npm install
```

Development:

```bash
npm run dev
```

Production build:

```bash
npm run build
```

## Docker

Build:

```bash
docker build -f Dockerfile.allinone -t syncstage .
```

Run:

```bash
docker run --rm -p 8080:8080 syncstage
```

Alternative:

```bash
docker run --rm -p 8081:8080 syncstage
```

## Docker Compose

```bash
docker compose up --build
```

Stop:

```bash
docker compose down
```

---

# Quick API Checks

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

Current Window 1 playback:

```bash
curl http://localhost:8080/api/windows/w1/now
```

SSE:

```bash
curl -N http://localhost:8080/api/events
```

---

# Example Sync Request

Start an 8-second sync:

```bash
curl -X POST http://localhost:8080/api/sync   -H "Content-Type: application/json"   -d "{"mediaId":"m2","durationSeconds":8}"
```

Then inspect:

```bash
curl http://localhost:8080/api/state
```

During synchronization, affected playback should report the sync source.

After the sync expires, playback should return to the normal sequence.

---

# Submission Checklist

Before submitting the assignment:

- [ ] GitHub repository is public
- [ ] README is present
- [ ] Live Render URL works
- [ ] `/api/health` returns `status: ok`
- [ ] Frontend loads
- [ ] All display windows render
- [ ] Images render
- [ ] M4 video renders
- [ ] M5 video renders
- [ ] Playlist add works
- [ ] Playlist delete works
- [ ] SSE connection works
- [ ] Global sync works
- [ ] Sync release works
- [ ] Detached displays work
- [ ] Playlist changes persist after refresh
- [ ] Production uses PostgreSQL/Neon
- [ ] No database secrets are committed
- [ ] `go vet ./...` passes
- [ ] `go test ./... -count=1` passes
- [ ] Frontend production build passes
- [ ] Docker image builds successfully
- [ ] Docker container serves frontend and videos

---

# Final Architecture Summary

The core playback flow is:

```text
                  Shared Server Time
                         │
                         ▼
                Five-Hour Cycle
                         │
                         ▼
                  Window Playlist
                         │
                         ▼
                  resolve(...)
                         │
                         ▼
                  Current Media
                         │
                         ▼
                   MediaSurface
```

The synchronization flow is:

```text
Operator
   │
   │ Start Sync
   ▼
Go Backend
   │
   │ Calculate future startAtMs
   ▼
SSE Broadcast
   │
   ├──────────┬──────────┬──────────┐
   ▼          ▼          ▼          ▼
Window 1   Window 2   Window 3   Window 4
   │          │          │          │
   └──────────┴──────────┴──────────┘
                    │
                    ▼
            Same server timestamp
                    │
                    ▼
              Sync takeover
                    │
                    ▼
            Normal timelines resume
```

SyncStage therefore combines:

```text
Deterministic playback
        +
Independent playlists
        +
Real-time state updates
        +
Server-clock calibration
        +
Future-timestamp synchronization
        +
Persistent storage
        +
Single-container deployment
```

to provide a practical multi-display media sequencing system.
