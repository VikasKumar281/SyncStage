# SyncStage

A multi-window media sequencer where every display runs its own playlist while an operator can update playlists and temporarily synchronize the same media across all displays.

SyncStage is built with a React frontend and a Go backend. The application uses a shared time-based playback model so every display can calculate what it should be showing at any moment.

The main feature is the **Sync Takeover**. An operator can select any media item and temporarily show it on all display windows at the same scheduled time. When the sync period ends, every window automatically returns to its own playlist without losing its normal position.

The deployed application uses PostgreSQL for persistent storage and is packaged as a single Docker container containing both the React production build and the Go server.

---

## Live Demo

**Live Application**

https://syncstage.onrender.com

**GitHub Repository**

https://github.com/VikasKumar281/SyncStage

---

# Table of Contents

1. [Project Overview](#project-overview)
2. [Main Features](#main-features)
3. [Technology Stack](#technology-stack)
4. [Application Architecture](#application-architecture)
5. [Project Structure](#project-structure)
6. [How the Application Works](#how-the-application-works)
7. [Time-Based Playback](#time-based-playback)
8. [Five-Hour Cycle](#five-hour-cycle)
9. [Playlist Looping](#playlist-looping)
10. [Blank Media](#blank-media)
11. [Sync Takeover](#sync-takeover)
12. [Clock Synchronization](#clock-synchronization)
13. [Server-Sent Events](#server-sent-events)
14. [Returning From Sync](#returning-from-sync)
15. [Dynamic Playlist Updates](#dynamic-playlist-updates)
16. [Detached Display Windows](#detached-display-windows)
17. [Video Playback](#video-playback)
18. [Persistent Storage](#persistent-storage)
19. [Seed Data](#seed-data)
20. [API Reference](#api-reference)
21. [Local Development](#local-development)
22. [Environment Variables](#environment-variables)
23. [Running With Docker](#running-with-docker)
24. [Running With Docker Compose](#running-with-docker-compose)
25. [Production Deployment](#production-deployment)
26. [Testing](#testing)
27. [Go and JavaScript Timeline Parity](#go-and-javascript-timeline-parity)
28. [Manual Verification](#manual-verification)
29. [Design Decisions](#design-decisions)
30. [Assumptions](#assumptions)
31. [Known Limitations](#known-limitations)
32. [Future Improvements](#future-improvements)

---

# Project Overview

SyncStage simulates a small digital-signage/media-display system.

There are multiple display windows and every window has its own playlist.

For example:

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

Every window continues playing its own sequence independently.

The operator can modify a playlist while the application is running.

For example:

```text
Before:

Window 1
M1 → M2 → M4
```

The operator can add another media item:

```text
After:

Window 1
M1 → M2 → M4 → M6
```

The change is stored in the backend and pushed to connected displays.

The second major feature is synchronized playback.

For example, if the operator selects `M2` and starts a 10-second sync:

```text
Window 1 → M2
Window 2 → M2
Window 3 → M2
Window 4 → M2
```

All windows switch to the selected media at the same scheduled instant.

After 10 seconds, the sync ends and each window returns to its own timeline.

---

# Main Features

## Display and Playback

- Multiple display windows
- Independent playlist for every window
- Continuous playlist playback
- Five-hour repeating cycle
- Image media
- Video media
- Blank media
- Configurable media duration
- Current media indicator
- Remaining-time display
- Playback progress indicator
- Next-media information

## Playlist Management

- Add media to a playlist
- Remove media from a playlist
- Change playlists while playback is running
- Updates are propagated to connected displays

## Synchronization

- Global sync takeover
- Selectable media
- Configurable sync duration
- Future timestamp based synchronization
- Client/server clock calibration
- Temporary sync override
- Automatic return to each window's normal sequence

## Real-Time Communication

- Server-Sent Events
- Automatic client reconnection through `EventSource`
- Complete state snapshots
- Live playlist updates
- Live sync events
- Cycle reset events

## Persistence

- PostgreSQL storage
- Neon PostgreSQL for the deployed version
- Seed data on first startup
- Storage abstraction
- Local JSON storage fallback for development

## Deployment

- Docker
- Multi-stage Docker build
- React and Go in one container
- Render deployment
- PostgreSQL through Neon
- Health check endpoint

## Testing

- Go unit tests
- API tests
- Scheduler/timeline tests
- Persistence tests
- CORS tests
- Go ↔ JavaScript timeline parity verification

---

# Technology Stack

## Frontend

- React
- Vite
- JavaScript
- CSS
- Browser EventSource API

## Backend

- Go
- Standard HTTP server
- REST APIs
- Server-Sent Events
- Time-based scheduler
- Storage abstraction

## Database

- PostgreSQL
- JSONB

The deployed application uses Neon PostgreSQL.

## Infrastructure

- Docker
- Render

---

# Application Architecture

The application follows a simple client/server architecture.

```text
                        ┌─────────────────────┐
                        │      Browser 1      │
                        │    React Display    │
                        └──────────┬──────────┘
                                   │
                                   │ REST + SSE
                                   │
                        ┌──────────▼──────────┐
                        │                     │
                        │     Go Backend      │
                        │                     │
                        │  REST API           │
                        │  Scheduler          │
                        │  SSE Hub            │
                        │  Storage Layer      │
                        │                     │
                        └──────────┬──────────┘
                                   │
                                   │
                        ┌──────────▼──────────┐
                        │    PostgreSQL       │
                        │      / Neon         │
                        └─────────────────────┘
```

Multiple browsers can connect to the same backend:

```text
Browser 1 ─┐
Browser 2 ─┤
Browser 3 ─┼──→ Go Backend ──→ PostgreSQL
Browser 4 ─┘
```

The backend maintains the application state.

The browser receives the current state and live updates, while the actual playback position is calculated from time.

---

# Production Architecture

For production, the React application is built into static files and served by the Go backend.

```text
                    Render
                      │
                      ▼
              ┌─────────────────┐
              │  Docker Image   │
              │                 │
              │  Go Server      │
              │       +         │
              │  React dist/    │
              │                 │
              └────────┬────────┘
                       │
                       │ DATABASE_URL
                       ▼
              ┌─────────────────┐
              │ Neon PostgreSQL │
              └─────────────────┘
```

This gives the production application a single origin:

```text
https://syncstage.onrender.com
```

The frontend and backend therefore do not need separate public domains.

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
│   │   │   ├── server.go
│   │   │   ├── middleware.go
│   │   │   ├── hub.go
│   │   │   └── server_test.go
│   │   │
│   │   ├── scheduler/
│   │   │   ├── scheduler.go
│   │   │   └── scheduler_test.go
│   │   │
│   │   ├── storage/
│   │   │   ├── store.go
│   │   │   ├── jsonstore/
│   │   │   └── pgstore/
│   │   │
│   │   ├── models/
│   │   │   └── models.go
│   │   │
│   │   ├── seed/
│   │   │   └── seed.go
│   │   │
│   │   └── config/
│   │       └── config.go
│   │
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
│
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   ├── WindowPlayer.jsx
│   │   │   ├── MediaSurface.jsx
│   │   │   ├── ControlPanel.jsx
│   │   │   └── StatusBar.jsx
│   │   │
│   │   ├── lib/
│   │   │   ├── timeline.js
│   │   │   ├── api.js
│   │   │   └── useSequencer.js
│   │   │
│   │   └── App.jsx
│   │
│   ├── scripts/
│   │   └── verify-parity.mjs
│   │
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

The most important playback files are:

```text
backend/internal/scheduler/scheduler.go
frontend/src/lib/timeline.js
```

The Go backend and React frontend both use the same timeline rules.

---

# How the Application Works

The overall flow is:

```text
1. Backend starts
       ↓
2. Configuration is loaded
       ↓
3. PostgreSQL is opened
       ↓
4. Existing state is loaded
       ↓
5. If no state exists, seed data is created
       ↓
6. HTTP server starts
       ↓
7. React frontend connects
       ↓
8. Browser gets initial state
       ↓
9. Browser calibrates its clock
       ↓
10. Browser calculates playback from time
       ↓
11. SSE keeps the state updated
```

The server does not need to send a "play next media" command every time an item ends.

Instead, both the server and browser can determine the current playback from the shared timeline.

---

# Time-Based Playback

The main playback design is based on time.

Conceptually:

```text
resolve(window, currentTime)
```

returns information such as:

```text
current media
start time
end time
remaining time
playlist index
cycle index
```

For example, suppose a window has:

```text
M1 = 8 seconds
M2 = 8 seconds
M4 = 15 seconds
```

The playlist timeline is:

```text
0s        8s        16s                  31s
│---------│----------│--------------------│
    M1         M2             M4
```

At:

```text
5 seconds
```

the current media is:

```text
M1
```

At:

```text
12 seconds
```

the current media is:

```text
M2
```

At:

```text
20 seconds
```

the current media is:

```text
M4
```

The browser can calculate this without asking the backend which media comes next.

---

# Why Playback Is Time-Based

A simpler design could have been:

```text
Browser
   ↓
"What should I play?"

Server
   ↓
"Play M2"

Browser
   ↓
"What next?"

Server
   ↓
"Play M4"
```

That approach creates unnecessary dependency on network requests.

If a display temporarily loses its connection, its playback could become incorrect.

In SyncStage, playback is derived from:

```text
cycle anchor
+
current time
+
window playlist
```

Therefore a display can determine its position again after:

- a page refresh
- reconnecting to the server
- opening a display late
- a temporary SSE disconnect

The backend is still responsible for the authoritative state, but it does not have to act as a real-time remote control for every media transition.

---

# Continuous Playback

Each window's playlist loops continuously.

Example:

```text
M1 → M2 → M4 → M1 → M2 → M4 → ...
```

The frontend calculates when the current item ends and schedules the next timeline calculation around that point.

The player does not continuously poll the server for the next item.

The current playback calculation uses:

```text
startedAtMs
endsAtMs
```

and the frontend schedules its next update using the remaining duration.

A small timing cushion is used around the transition so the recalculation lands inside the next timeline slot instead of repeatedly hitting the exact boundary.

The progress bar uses a separate, slower refresh interval.

This keeps:

```text
media switching
```

separate from:

```text
visual progress updates
```

---

# Five-Hour Cycle

The required cycle length is five hours.

The application represents five hours as:

```text
5 × 60 × 60 × 1000
```

which is:

```text
18,000,000 milliseconds
```

The default configuration is:

```text
CYCLE_SECONDS=18000
```

The cycle is shared across all windows.

However, every window still has its own playlist.

For example:

```text
                    Shared 5-hour cycle
                            │
          ┌─────────────────┼─────────────────┐
          ▼                 ▼                 ▼
       Window 1          Window 2          Window 3
       M1 M2 M4          M3 B M2 M6        M5 M1
```

The cycle provides a common time reference while each playlist remains independent.

---

# Playlist Looping

The playlist does not have to be exactly five hours long.

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

The playlist repeats:

```text
M1 → M2 → M4 → M1 → M2 → M4 → ...
```

The current position inside the playlist is determined using the position inside the current five-hour cycle.

---

# Five-Hour Boundary

A media item may cross the five-hour boundary.

For example:

```text
Five-hour boundary is 3 seconds away.

Current media normally has:
10 seconds remaining.
```

The application does not extend the current cycle beyond five hours.

Instead:

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

The current item is truncated at the cycle boundary.

The next cycle starts from the first playlist item.

No automatic blank media is inserted to fill the remaining time.

---

# Blank Media

Blank is treated as a real media type.

For example:

```text
M3 → Blank → M2
```

means that the window intentionally shows a blank state for the duration configured for that playlist item.

Blank is not used automatically to pad the five-hour cycle.

For example:

```text
M1 → M2 → M4
```

does not become:

```text
M1 → M2 → M4 → Blank → Blank → ...
```

just because the playlist does not fill the entire five-hour period.

Blank appears only when:

```text
type = blank
```

is explicitly configured.

An empty playlist is a separate state.

If a window has no playlist items, the UI shows that nothing is scheduled rather than treating the empty playlist as a configured blank item.

---

# Sync Takeover

The sync feature allows the operator to temporarily override all normal playlists.

For example:

```text
Selected media:
M2

Duration:
10 seconds
```

The operator clicks:

```text
Sync all windows
```

The desired result is:

```text
Window 1 → M2
Window 2 → M2
Window 3 → M2
Window 4 → M2
```

and all windows should start the sync as close to the same instant as possible.

---

# Why Immediate Sync Is Not Enough

A basic implementation might send:

```text
"Play M2 now"
```

to every browser.

The problem is that the message will not arrive at exactly the same time on every machine.

For example:

```text
Window 1 → receives at T + 80ms
Window 2 → receives at T + 130ms
Window 3 → receives at T + 210ms
Window 4 → receives at T + 270ms
```

Even though all browsers received the same message, they would not switch at the same instant.

Network latency, rendering time and browser scheduling can all introduce small differences.

---

# Future-Timestamp Sync

SyncStage treats synchronization as a scheduled event instead of an immediate command.

When the operator starts a sync, the backend calculates:

```text
current server time
+
sync lead time
=
scheduled start time
```

The default lead time is:

```text
SYNC_LEAD_MS=1200
```

For example:

```text
Current server time:
1789466864364

Scheduled start:
1789466865564
```

The difference is:

```text
1200 milliseconds
```

The server sends the scheduled timestamp to connected clients.

Each browser receives the event and waits for that same timestamp according to its calibrated server clock.

The event therefore does not need to arrive at exactly the moment the sync starts.

It only needs to arrive before the scheduled start.

---

# Sync Timeline

The process looks like this:

```text
Operator
   │
   │ Click "Sync all windows"
   ▼
Go Backend
   │
   │ Calculate future startAtMs
   ▼
SSE Broadcast
   │
   ├─────────────┬─────────────┬─────────────┐
   ▼             ▼             ▼             ▼
Window 1      Window 2      Window 3      Window 4
   │             │             │             │
   │             │             │             │
   └─────────────┴─────────────┴─────────────┘
                         │
                         ▼
                 Scheduled timestamp
                         │
                         ▼
                 All show selected media
```

---

# Clock Synchronization

The future-timestamp approach requires the browsers to have a reasonably accurate idea of server time.

Different machines can have different local clocks.

For example:

```text
Machine A → 12:00:00.000
Machine B → 11:59:59.700
Machine C → 12:00:00.250
```

If the browsers used their local clocks directly, they could disagree about when the scheduled timestamp occurs.

To reduce this difference, the frontend uses:

```text
GET /api/time
```

to measure the server clock.

---

# Clock Calibration Process

The browser sends multiple time requests.

For each request:

```text
1. Record local send time
2. Send request to /api/time
3. Receive server timestamp
4. Record local receive time
5. Calculate round-trip time
6. Estimate server clock offset
```

The basic calculation is:

```text
offsetMs =
serverTimeMs -
(sentAt + roundTripMs / 2)
```

The browser then calculates:

```text
serverNow() =
Date.now() + offsetMs
```

The frontend periodically recalibrates the offset.

The current clock offset and round-trip information are displayed in the application UI.

---

# Why Multiple Time Probes Are Used

Network latency is not always consistent.

For example:

```text
Probe 1 → 180ms
Probe 2 → 95ms
Probe 3 → 72ms
Probe 4 → 140ms
Probe 5 → 88ms
```

The lowest round-trip sample is usually the best approximation for estimating the server clock because it contains less network delay.

The frontend therefore uses the best available sample rather than blindly trusting one request.

---

# Returning From Sync

One important requirement is that the normal playlists must not be lost when a sync starts.

SyncStage does not solve this by doing:

```text
save current position
        ↓
show sync
        ↓
restore saved position
```

Instead, the normal timeline continues logically underneath the sync.

For example:

```text
Normal Window 1:

M1 → M2 → M4 → M1 → ...
```

A sync starts:

```text
SYNC → M6
```

While M6 is being displayed, the normal Window 1 timeline is still based on the current time.

When the sync ends:

```text
SYNC ends
    ↓
resolve normal sequence
    ↓
show whatever Window 1 should be showing now
```

This means every window automatically returns to its own sequence without needing a special restore operation.

---

# Example

Suppose:

```text
Window 1:
M1 → M2 → M4

Window 2:
M3 → Blank → M2 → M6
```

A sync starts:

```text
M5 for 10 seconds
```

During sync:

```text
Window 1 → M5
Window 2 → M5
Window 3 → M5
Window 4 → M5
```

After sync:

```text
Window 1 → its normal sequence
Window 2 → its normal sequence
Window 3 → its normal sequence
Window 4 → its normal sequence
```

The playlists themselves were never replaced.

---

# Server-Sent Events

SyncStage uses Server-Sent Events for live state updates.

The frontend opens:

```text
GET /api/events
```

using the browser's:

```javascript
EventSource
```

API.

The server sends complete snapshots when relevant changes happen.

Events include:

```text
snapshot
playlist.updated
sync.scheduled
sync.cancelled
media.created
window.created
cycle.reset
```

This allows connected display windows to receive updates without repeatedly polling the backend.

---

# Why SSE Instead of WebSockets?

The application's real-time communication is mostly:

```text
Server → Browser
```

Client-side operations such as:

```text
add playlist item
remove playlist item
create media
trigger sync
```

are normal HTTP REST requests.

Because the live communication is primarily one-way, Server-Sent Events are a good fit.

Advantages:

- Simple browser API
- Uses normal HTTP
- Easy to reconnect
- Simple event model
- Good fit for server-to-client state updates
- No need to implement a full duplex socket protocol

---

# Frontend State Management

The main frontend sequencer logic is handled through:

```text
frontend/src/lib/useSequencer.js
```

The hook manages:

```text
snapshot
connection state
errors
clock offset
round-trip time
actions
```

The browser connects to the SSE endpoint:

```javascript
const source = new EventSource(api.eventsUrl());
```

It listens for the application events and updates the local snapshot.

The hook also periodically recalibrates the clock.

When the browser becomes visible again after being in the background, it refreshes the state and recalibrates the clock.

---

# WindowPlayer

Each display is rendered by:

```text
frontend/src/components/WindowPlayer.jsx
```

The component receives:

```text
snapshot
window
serverNow()
```

and resolves the current playback.

Conceptually:

```text
snapshot
   +
window
   +
serverNow()
       │
       ▼
    resolve()
       │
       ▼
current playback
       │
       ▼
MediaSurface
```

The component also shows:

- Window name
- Number of playlist items
- Current media
- Remaining time
- Progress
- Next media
- Sync status
- Playlist items
- Detach button

The component defensively handles missing or empty playlists so the UI does not fail while state is loading or if a window has no configured items.

---

# Video Playback

Images are displayed directly.

Videos require additional handling because a video can be joined in the middle of its timeline.

For example, if the timeline says:

```text
Video should currently be at 7.2 seconds
```

the player can seek toward that position instead of always starting from:

```text
0 seconds
```

This is important when:

- A browser opens in the middle of a cycle.
- A display is refreshed.
- A new display joins.
- A sync takeover ends.
- The timeline moves to a video item after another media item.

Videos are played muted and inline because browsers commonly restrict autoplay when audio is enabled.

---

# Dynamic Playlist Updates

Playlist changes can be made while the application is running.

Example:

```text
Initial:

Window 1
M1 → M2 → M4
```

Add:

```text
M6
```

Result:

```text
M1 → M2 → M4 → M6
```

The frontend sends a REST request:

```text
POST /api/windows/{id}/playlist
```

The backend:

```text
1. Validates the request
2. Loads current state
3. Updates the playlist
4. Persists the new state
5. Broadcasts the updated snapshot
```

Connected displays receive the update through SSE.

---

# Removing Playlist Items

Playlist items can also be removed while the application is running.

Endpoint:

```text
DELETE /api/windows/{id}/playlist/{itemId}
```

After successful deletion:

```text
PostgreSQL
     ↓
updated state
     ↓
SSE
     ↓
connected clients
```

The displays update without requiring a full page reload.

---

# Detached Display Windows

Each window has a:

```text
Detach
```

button.

The purpose is to open a single display independently from the control interface.

For example:

```text
https://syncstage.onrender.com/?window=w1
```

can be used to open one display.

This is useful for testing the actual multi-window behavior.

A test setup can look like:

```text
Browser Window 1 → Display 1
Browser Window 2 → Display 2
Browser Window 3 → Display 3
Browser Window 4 → Display 4
```

Then the operator can trigger a sync and visually verify that the displays switch together.

---

# Persistent Storage

The deployed version uses PostgreSQL.

The application has a storage abstraction so the rest of the backend does not need to know whether the current store is PostgreSQL or the local JSON fallback.

The PostgreSQL implementation stores the application state as JSONB.

The main table is:

```sql
CREATE TABLE IF NOT EXISTS sequencer_state (
    id INTEGER PRIMARY KEY,
    state JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

The stored state contains the application aggregate, including:

```text
media
windows
playlists
cycle information
active sync
```

---

# Why JSONB Storage?

The application's state is relatively small and is naturally represented as one aggregate.

The project does not need a large reporting system or complex relational queries.

Therefore, storing the complete state as PostgreSQL JSONB keeps the persistence layer simple.

The flow is:

```text
Application State
       ↓
JSON Marshal
       ↓
PostgreSQL JSONB
```

When loading:

```text
PostgreSQL JSONB
       ↓
JSON Unmarshal
       ↓
Application State
```

---

# Storage Updates

State updates are serialized inside the storage layer.

The basic update flow is:

```text
Request
   ↓
Lock
   ↓
Load current state
   ↓
Apply mutation
   ↓
Serialize state
   ↓
Write to PostgreSQL
   ↓
Unlock
```

This prevents simple concurrent read-modify-write operations from overwriting each other's changes.

---

# PostgreSQL Startup

When the backend starts with a configured:

```text
DATABASE_URL
```

it uses PostgreSQL storage.

If the database table does not exist, it is created automatically.

If the state row does not exist, the application creates the initial state from the seed data.

Therefore the deployed application can start with working example data without manually inserting records into the database.

---

# Local JSON Fallback

For local development, if:

```text
DATABASE_URL
```

is not configured, the application can use the JSON storage implementation.

This makes it possible to run the backend locally without requiring PostgreSQL.

For production, PostgreSQL is used because the deployed application requires persistent storage.

---

# Seed Data

The project includes seed data for immediate demonstration.

The seed contains:

```text
7 media items
4 display windows
```

The media set includes:

```text
M1 → image
M2 → image
M3 → image
M4 → video
M5 → video
M6 → image
Blank → blank
```

The seeded windows have different playlists.

Example:

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

The different playlists make it easy to verify that the windows are actually independent.

---

# API Reference

All APIs use JSON for request and response bodies.

Time values are represented as Unix epoch milliseconds.

Durations are represented internally in milliseconds.

Some mutation APIs accept duration values in seconds because that is easier for the UI and API consumers.

---

## Health

```http
GET /api/health
```

Returns service information including:

```text
status
uptime
cycle duration
connected SSE clients
server time
```

Example:

```json
{
  "status": "ok",
  "cycleMs": 18000000,
  "serverTimeMs": 1789466864364,
  "sseClients": 0,
  "uptimeSeconds": 31
}
```

---

## Server Time

```http
GET /api/time
```

Returns the current backend time.

The frontend uses this endpoint for clock calibration.

Example:

```json
{
  "serverTimeMs": 1789466864364
}
```

---

## Complete State

```http
GET /api/state
```

This is the main state endpoint.

It provides the information required by the frontend to render the application.

The response contains:

```text
serverTimeMs
cycleMs
cycleAnchorMs
media
windows
activeSync
playback
```

A simplified response looks like:

```json
{
  "serverTimeMs": 1789466864364,
  "cycleMs": 18000000,
  "cycleAnchorMs": 1789466829807,
  "media": [],
  "windows": [],
  "activeSync": null,
  "playback": []
}
```

---

## Media

```http
GET /api/media
```

Returns the media library.

---

## Windows

```http
GET /api/windows
```

Returns all display windows and their playlists.

---

## Current Window Playback

```http
GET /api/windows/{id}/now
```

Returns what the server currently calculates for a specific window.

The response can also provide the window's normal sequence separately from an active sync so the client can understand what the display will return to after the takeover.

---

# Create Media

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

A URL is required for image and video media.

Blank media does not require a URL.

---

# Create Window

```http
POST /api/windows
```

Example:

```json
{
  "name": "Window 5"
}
```

A newly created window starts with an empty playlist.

---

# Add Playlist Item

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

The duration can use the media's default duration when not explicitly supplied.

A position can also be supplied when the caller wants to insert an item at a particular position.

---

# Remove Playlist Item

```http
DELETE /api/windows/{id}/playlist/{itemId}
```

Returns:

```text
204 No Content
```

when the item is successfully removed.

---

# Start Sync

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

The backend calculates a future start time using the configured sync lead time.

An exact future `startAtMs` can also be supplied when required.

The response contains the scheduled sync information.

---

# Cancel Sync

```http
DELETE /api/sync
```

Cancels an armed or currently active sync takeover.

---

# Reset Cycle

```http
POST /api/cycle/reset
```

Re-anchors the five-hour cycle to the current server time.

After a cycle reset, windows start their timeline from the beginning of their playlists.

---

# Event Stream

```http
GET /api/events
```

This endpoint uses Server-Sent Events.

Events include:

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

# API Error Format

Validation and lookup errors use JSON.

Example:

```json
{
  "error": "media not found"
}
```

Typical status codes include:

```text
200 → successful request
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

Docker is optional if you want to run the complete application in a container.

---

# Run the Backend

Open a terminal:

```bash
cd backend
go run ./cmd/server
```

The backend normally listens on:

```text
http://localhost:8080
```

The backend reads its configuration from environment variables.

---

# Run the Frontend

Open a second terminal:

```bash
cd frontend
npm install
npm run dev
```

The frontend normally runs on:

```text
http://localhost:5173
```

Vite proxies API requests to the backend during development.

---

# Local Development With PostgreSQL

If you want to use PostgreSQL locally, configure:

```text
DATABASE_URL=<your PostgreSQL connection string>
```

Then run:

```bash
cd backend
go run ./cmd/server
```

The backend will use PostgreSQL instead of the local JSON fallback.

The database table is created automatically.

---

# Local Development Without PostgreSQL

For a simple local run, `DATABASE_URL` can be omitted.

The backend then uses the JSON storage implementation.

This is useful when working only on the frontend or scheduler logic without setting up a database.

---

# Environment Variables

## Backend Variables

| Variable | Default | Description |
|---|---:|---|
| `PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | empty | PostgreSQL connection string |
| `DATA_PATH` | `data/state.json` | Path used by local JSON storage |
| `CYCLE_SECONDS` | `18000` | Five-hour cycle |
| `SYNC_LEAD_MS` | `1200` | Time given to clients before sync starts |
| `SYNC_DEFAULT_SECONDS` | `10` | Default sync duration |
| `ALLOWED_ORIGINS` | `*` | Allowed browser origins |
| `STATIC_DIR` | empty | Directory containing React production files |

---

# Five-Hour Cycle Configuration

The normal production value is:

```text
CYCLE_SECONDS=18000
```

because:

```text
18000 seconds
=
5 hours
```

For local testing, the cycle can be reduced.

For example:

```text
CYCLE_SECONDS=120
```

creates a two-minute cycle.

This is useful for testing cycle boundaries without waiting five hours.

The actual timeline behavior remains the same.

---

# Sync Configuration

The default sync lead time is:

```text
SYNC_LEAD_MS=1200
```

This means the server normally schedules a sync approximately 1.2 seconds in the future.

The default sync duration is:

```text
SYNC_DEFAULT_SECONDS=10
```

The UI allows the operator to configure the sync duration.

---

# Frontend Configuration

The main frontend configuration is:

```text
VITE_API_BASE_URL
```

When the frontend and backend are deployed separately, this can point to the backend URL.

For the current single-container deployment, the value is left empty so the browser uses the same origin.

---

# Running With Docker

The project contains:

```text
Dockerfile.allinone
```

This Dockerfile builds both the frontend and backend.

The build has three main stages.

---

## Stage 1 — Frontend Build

The Docker build uses Node to:

```text
install frontend dependencies
       ↓
build React application
       ↓
create frontend/dist
```

The result is the production React build.

---

## Stage 2 — Backend Build

The Docker build uses Go to:

```text
download Go dependencies
       ↓
copy backend source
       ↓
go vet ./...
       ↓
go test ./...
       ↓
build Go server
```

This means the Docker build also performs backend validation.

---

## Stage 3 — Final Runtime Image

The final image contains:

```text
Go server
React production build
```

The final runtime image is kept separate from the build environments so the final container does not need Node or the Go compiler.

---

# Build Docker Image

From the repository root:

```bash
docker build -f Dockerfile.allinone -t syncstage .
```

---

# Run Docker Container

```bash
docker run -p 8080:8080 syncstage
```

Then open:

```text
http://localhost:8080
```

The Go server serves both:

```text
/api/*
```

and:

```text
React frontend
```

from the same container.

---

# Docker Compose

The repository also includes:

```text
docker-compose.yml
```

Run:

```bash
docker compose up --build
```

Then open:

```text
http://localhost:8080
```

To stop:

```bash
docker compose down
```

---

# Production Deployment

The current production deployment uses:

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

The deployed service is:

```text
https://syncstage.onrender.com
```

---

# Render Configuration

The repository contains:

```text
render.yaml
```

The important configuration is:

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

The Render service uses:

```text
Dockerfile.allinone
```

as the production Dockerfile.

---

# Render Health Check

Render uses:

```text
/api/health
```

as the health check endpoint.

The backend returns a successful response when the service is running correctly.

Example:

```json
{
  "status": "ok",
  "cycleMs": 18000000,
  "serverTimeMs": 1789466864364,
  "sseClients": 0,
  "uptimeSeconds": 31
}
```

---

# Neon PostgreSQL

The production database is hosted using Neon PostgreSQL.

The connection string is configured through:

```text
DATABASE_URL
```

The database connection string should be stored as an environment variable and should not be committed to the repository.

The backend creates the required database table automatically.

---

# Production Deployment Flow

The production deployment works like this:

```text
Developer pushes code
        │
        ▼
GitHub repository
        │
        ▼
Render detects commit
        │
        ▼
Docker build
        │
        ├── React build
        ├── Go tests
        ├── go vet
        └── Go production build
        │
        ▼
Final Docker image
        │
        ▼
Render Web Service
        │
        ├── React frontend
        └── Go backend
                │
                ▼
        Neon PostgreSQL
```

---

# Testing

Testing focuses on the timeline and synchronization logic because these are the most important parts of the application.

Run the backend tests with:

```bash
cd backend
go vet ./...
go test ./... -count=1
```

---

# What the Tests Cover

The backend test suite covers cases such as:

- Playlist walking
- Playlist looping
- Five-hour cycle boundary
- Boundary truncation
- Clock-before-anchor handling
- Sync override
- Sync release
- Expired sync
- Playlist item addition
- Playlist item deletion
- Unknown media IDs
- Unknown window IDs
- Persistence
- CORS preflight behavior

The goal is to test both normal playback and the edge cases around time boundaries and sync.

---

# Go and JavaScript Timeline Parity

The playback algorithm exists in two places:

```text
Go:
backend/internal/scheduler/scheduler.go
```

and:

```text
JavaScript:
frontend/src/lib/timeline.js
```

This is necessary because the browser needs to calculate its own playback locally.

However, having two implementations creates a possible source of bugs.

For example:

```text
Go says:
M2

JavaScript says:
M1
```

would cause the backend and display to disagree.

To catch this, the project contains:

```text
frontend/scripts/verify-parity.mjs
```

---

# Run Parity Verification

Start the backend:

```bash
cd backend
go run ./cmd/server
```

Then in another terminal:

```bash
cd frontend
node scripts/verify-parity.mjs http://localhost:8080
```

The script compares the backend's current playback result with the frontend timeline calculation.

The expected result is that both implementations agree for every window.

---

# Manual Verification

The live application can be tested manually.

Open:

```text
https://syncstage.onrender.com
```

The initial screen should show the seeded display windows.

---

## Test 1 — Independent Playback

Observe different windows.

Each window should show its own media according to its own playlist.

For example:

```text
Window 1 → M1
Window 2 → M3
Window 3 → M5
Window 4 → M6
```

The exact media depends on the current cycle position.

The important point is that the windows do not all follow the same playlist.

---

# Test 2 — Playlist Update

Choose a window.

Add a media item.

The playlist should update without requiring a page refresh.

Open another display window and verify that the change is also reflected there.

---

# Test 3 — Playlist Removal

Remove an existing playlist item.

The item should disappear from the playlist.

Connected display windows should receive the updated state.

---

# Test 4 — Sync Takeover

Select a media item from:

```text
Sync takeover
```

Choose a duration.

Click:

```text
Sync all windows
```

All open windows should switch to the selected media.

The display cards should also indicate that they are currently under sync.

---

# Test 5 — Sync Release

Wait for the configured sync duration.

After the sync ends, each display should return to its own normal playlist.

The windows should not all continue playing the synchronized media.

---

# Test 6 — Detached Windows

Click:

```text
Detach
```

on a window.

A separate display should open.

Repeat this for multiple windows.

Then trigger a sync from the main control interface.

This provides a more realistic synchronization test than displaying everything inside one page.

---

# Test 7 — Persistence

Make a playlist change.

Refresh the page.

The changed playlist should still exist because the deployed version stores the state in PostgreSQL.

---

# Test 8 — Health Endpoint

Open:

```text
https://syncstage.onrender.com/api/health
```

A successful deployment should return a JSON response containing:

```text
status = ok
```

---

# Test 9 — Server Time

Open:

```text
https://syncstage.onrender.com/api/time
```

The response should contain the current server timestamp.

The frontend uses this endpoint for clock calibration.

---

# Test 10 — Live Event Stream

The SSE endpoint is:

```text
https://syncstage.onrender.com/api/events
```

A connected client receives live application snapshots and events.

---

# Design Decisions

## 1. Time-Based Playback

The main design decision was to make playback a function of time.

Instead of:

```text
server sends "play next"
```

the application calculates:

```text
current time
+
cycle anchor
+
playlist
=
current media
```

### Benefits

- A display does not depend on a server command for every media transition.
- A refreshed browser can calculate its current position.
- A newly opened display can calculate its current position.
- Temporary network problems do not require the server to track every media transition.
- The same timeline can be used for synchronization.

### Tradeoff

The timeline logic has to exist in both Go and JavaScript.

That is why a parity test is included.

---

# 2. Future-Timestamp Synchronization

Sync is represented as a scheduled event rather than an immediate command.

### Benefits

- Clients have time to receive the sync event.
- Every client targets the same server timestamp.
- Network latency has less impact on the actual switching moment.

### Tradeoff

Synchronization can never be mathematically perfect on arbitrary networks.

A client with a very slow or interrupted connection may not receive the event before the scheduled start.

The sync lead time provides a reasonable buffer for normal connections.

---

# 3. Server Clock Calibration

Client system clocks are not guaranteed to match each other.

The application therefore estimates the difference between:

```text
client clock
```

and:

```text
server clock
```

and uses the adjusted time for timeline calculations.

---

# 4. SSE Instead of WebSockets

The application's real-time communication is mostly:

```text
server → browser
```

while browser actions use REST.

SSE is therefore simpler than introducing a full WebSocket protocol.

---

# 5. PostgreSQL JSONB

The application state is a relatively small aggregate.

There is no requirement for complex database queries.

PostgreSQL JSONB provides persistence while keeping the storage implementation straightforward.

The storage interface also keeps the application independent from the specific persistence implementation.

---

# 6. Single Container Deployment

The React frontend and Go backend are deployed together.

The production container contains:

```text
React build
+
Go server
```

This keeps deployment simple and means the browser can use one origin for:

```text
frontend
API
SSE
```

---

# Assumptions

## Five Hours Is a Repeating Cycle

The five-hour period is treated as a cycle, not as a total lifetime for a playlist.

After five hours:

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

---

## Cycle Boundary Truncates the Current Item

If an item crosses the five-hour boundary, it is truncated at the boundary.

The next cycle begins from the first playlist item.

---

## Blank Is Explicit

Blank media appears only when a blank item is explicitly included in a playlist.

Unused time is not automatically converted to blank playback.

---

## Sync Media Does Not Need to Be in Every Playlist

A sync takeover uses a media item from the media library.

The media does not have to exist in every window's normal playlist.

The sync is a temporary global override.

---

## Normal Playback Continues Under Sync

The normal sequence is not replaced by the sync state.

The sync only changes what is currently displayed.

After sync expiration, the normal timeline is resolved again.

---

## Media Is Referenced by URL

The application stores media metadata and URLs.

It does not implement a complete media upload/transcoding pipeline.

External image and video URLs can therefore be used for the seeded data.

---

# Known Limitations

## Single Backend Instance

The current architecture is intended for a single backend instance.

The in-memory SSE connection hub is local to that backend process.

If the application were scaled horizontally to multiple backend instances, a shared event broker or distributed messaging system would be needed so that a state update from one instance reaches clients connected to another instance.

---

## External Media Availability

Images and videos are loaded from their configured URLs.

If an external media host is unavailable, the browser cannot display that media.

The application does not currently download and cache all external media.

---

## Browser Autoplay Rules

Videos are muted and played inline to work with browser autoplay restrictions.

Browsers can still apply their own playback policies.

---

## Network Conditions

The timestamp-based sync design improves synchronization but does not eliminate network limitations.

A display with a severe network delay may not receive the sync event before the scheduled start.

---

## Media Upload

The current implementation does not provide a full media upload pipeline.

Media is represented by metadata and a URL.

---

# Future Improvements

Some improvements that could be added in a larger production version include:

- Authentication and role-based operator access
- Media upload and object storage
- Media validation before adding URLs
- Distributed SSE/event broker for multiple backend instances
- Redis or another shared event system
- More detailed playback diagnostics
- Display health monitoring
- Offline media caching
- Better handling of unavailable media
- Playlist drag-and-drop ordering
- Scheduling sync events for future dates
- Audit logs for operator actions
- Persistent per-display configuration
- More detailed monitoring and metrics
- Automated end-to-end browser tests
- Better media management UI

These are outside the core scope of the current implementation.

---

# Useful Commands

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

Check:

```bash
go vet ./...
```

Test:

```bash
go test ./... -count=1
```

---

## Frontend

Install:

```bash
cd frontend
npm install
```

Development server:

```bash
npm run dev
```

Production build:

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
docker run -p 8080:8080 syncstage
```

Compose:

```bash
docker compose up --build
```

Stop Compose:

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

SSE stream:

```bash
curl -N http://localhost:8080/api/events
```

---

# Example Sync Request

Start an 8-second sync:

```bash
curl -X POST http://localhost:8080/api/sync \
  -H "Content-Type: application/json" \
  -d "{\"mediaId\":\"m2\",\"durationSeconds\":8}"
```

Then inspect:

```bash
curl http://localhost:8080/api/state
```

During the sync, the playback source should be:

```text
sync
```

for the affected windows.

After the sync expires, the source returns to:

```text
sequence
```

---

# Example Playlist Update

Add M2 to Window 1:

```bash
curl -X POST http://localhost:8080/api/windows/w1/playlist \
  -H "Content-Type: application/json" \
  -d "{\"mediaId\":\"m2\",\"durationSeconds\":8}"
```

The backend persists the change and connected clients receive the updated state through SSE.

---

# Project Goals

The main goals of SyncStage are:

```text
Independent display playback
            +
Accurate time-based sequencing
            +
Reliable synchronized takeover
            +
Live playlist updates
            +
Persistent state
            +
Simple deployment
```

The project intentionally keeps the architecture relatively small while handling the important edge cases around:

- five-hour cycle boundaries
- playlist looping
- temporary synchronization
- client clock differences
- live state updates
- persistence
- browser refreshes
- video timeline positioning

---

# Summary

SyncStage is a multi-window media sequencing application built with:

```text
React
   +
Go
   +
PostgreSQL
   +
Server-Sent Events
   +
Docker
```

The most important design choice is that playback is **time-derived** rather than controlled entirely through sequential server commands.

The synchronization feature builds on the same idea by scheduling a future timestamp and allowing every client to target that timestamp using a calibrated server clock.

This allows:

```text
Window 1 ─┐
Window 2 ─┤
Window 3 ─┼──→ synchronized media at one scheduled instant
Window 4 ─┘
```

while still allowing every window to maintain its own independent playlist.

The application is available here:

**Live Demo:**  
https://syncstage.onrender.com

**GitHub:**  
https://github.com/VikasKumar281/SyncStage
