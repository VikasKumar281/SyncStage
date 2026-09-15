# Multi-Window Media Sequencer with Sync Playback

Four display windows, each looping its own playlist inside a 5-hour cycle. An
operator can change any playlist while everything is running, and can trigger a
**sync takeover** that puts one media item on every window at the same instant —
after which each window returns to its own sequence exactly where it would have
been.

React frontend, Go backend, persistent storage, deployable as one container.

```
┌──────────── one command ────────────┐
│  docker compose up --build          │
│  → http://localhost:8080            │
└─────────────────────────────────────┘
```

---

## Contents

1. [Quick start](#quick-start)
2. [How sync works](#how-sync-works) ← the interesting part
3. [How continuous playback works](#how-continuous-playback-works)
4. [The 5-hour cycle](#the-5-hour-cycle)
5. [Verifying it actually works](#verifying-it-actually-works)
6. [API reference](#api-reference)
7. [Project layout](#project-layout)
8. [Deployment](#deployment)
9. [Configuration](#configuration)
10. [Assumptions and tradeoffs](#assumptions-and-tradeoffs)

---

## Quick start

### Option A — Docker (one command, nothing else installed)

```bash
docker compose up --build
```

Open <http://localhost:8080>. The Go process serves the API and the built React
app from one origin, so there is no CORS setup and no second terminal.

### Option B — local development

Two terminals. Requires Go 1.22+ and Node 18+.

```bash
# terminal 1 — API on :8080
cd backend
go run ./cmd/server
```

```bash
# terminal 2 — UI on :5173
cd frontend
npm install
npm run dev
```

Open <http://localhost:5173>. Vite proxies `/api` to the backend, so the browser
stays on one origin in development too.

There is nothing to configure and no database to provision. On first start the
backend writes `backend/data/state.json` with the seed data (7 media items, 4
windows) and begins playing immediately.

> **No network at build time?** The Go module has **zero external dependencies**,
> so `go run ./cmd/server` works with no `go mod download` step. Only the
> frontend needs `npm install`.

---

## How sync works

This is the part of the assignment worth thinking hardest about, so here is the
reasoning in full.

### The approach that does not work

The obvious design is a server that pushes commands: *"everyone show M2 now."*

It fails in practice. The message reaches four browsers at four different
moments — one is on wifi, one is mid-render, one is on a throttled background
tab. "At the same time" becomes a lie by tens or hundreds of milliseconds, and
it gets worse with every extra display. Any dropped message desynchronises a
window permanently, and a page refresh restarts that window from item 0.

### What this implementation does instead

A sync is **not a command. It is a reservation on a shared timeline.**

```
POST /api/sync { "mediaId": "m2", "durationSeconds": 10 }

    server time now         = 1789466864364
    scheduled start         = 1789466865564   ← now + 1200ms
    scheduled end           = 1789466873564
```

The server picks an instant **slightly in the future** and broadcasts it. Every
window has already calibrated its clock against the server, so each one knows
when `1789466865564` will occur on its own machine — and they all switch on that
tick, independently, without needing the message to arrive at the same time.
The message only has to arrive *before* the deadline.

`SYNC_LEAD_MS` (default 1200 ms) is that head start. It comfortably covers an
SSE hop plus a render on a normal connection.

### Clock calibration

Laptops are routinely seconds off NTP, which would make the whole scheme
worthless. On load, and every two minutes after, each client probes
`GET /api/time` five times and keeps the sample with the **lowest round trip**
— Cristian's algorithm, on the reasoning that the fastest probe is where the
"half the round trip each way" assumption is least wrong:

```js
offsetMs = serverTimeMs - (sentAt + roundTripMs / 2)
serverNow() = Date.now() + offsetMs
```

The measured offset and round trip are displayed in the top-right of the UI.
That number is the real error bound on how simultaneous the sync is.

### Returning to the normal sequence

The requirement is that after the sync, *"each window should continue its own
normal sequence without losing its playlist configuration."*

There is **no save-and-restore logic anywhere in this codebase**, because none
is needed. A window's own position is a function of the wall clock, so it keeps
advancing underneath the sync the whole time it is off screen. When the sync
window closes, the window simply stops asking about the sync and resumes asking
about its own timeline — and lands exactly where it would have been:

```
=== during sync ===
  w1: source=sync media=m2 window=1789466865564-1789466873564
  w2: source=sync media=m2 window=1789466865564-1789466873564
  w3: source=sync media=m2 window=1789466865564-1789466873564
  w4: source=sync media=m2 window=1789466865564-1789466873564
  → every window switches on the same millisecond

=== after the sync expires ===
  w1: sequence m2      idx=1
  w2: sequence m-blank idx=1
  w3: sequence m5      idx=0
  w4: sequence m3      idx=1
  → back on their own independent, out-of-phase sequences
```

While a takeover is live, each window's card shows what it *would* be playing
and when it resumes, so the behaviour is visible rather than something you have
to take on trust.

---

## How continuous playback works

Playback is a **pure function of time**:

```
resolve(window, t) → { media, startedAt, endsAt, remainingMs }
```

Given the shared cycle anchor and a window's playlist, any client can compute
what belongs on screen right now, without asking the server. The consequences
are what make the system robust:

| | |
|---|---|
| **No "next item" endpoint** | The assignment explicitly allows this. Nothing is streamed; nothing polls. |
| **Network outage** | A display disconnected for ten minutes keeps playing correctly and is still frame-accurate when it returns. |
| **Page refresh** | Resumes mid-item at the correct position, not from item 0. |
| **New display** | Opens already in step with every existing one. |
| **Server restart** | Displays never notice. |

The server is only needed to answer *"has anything changed?"* — which it does
over Server-Sent Events.

Switching is driven by a self-scheduling `setTimeout` that fires at the exact
instant the slot ends (plus a 20 ms cushion so the recomputation lands inside
the next slot, not on the seam). A separate, slower interval updates the
progress bar. Keeping those apart means switches stay frame-accurate no matter
how coarse the progress refresh is.

### The algorithm exists twice — and they are checked against each other

`resolve()` is implemented in Go (`backend/internal/scheduler/scheduler.go`) and
in JavaScript (`frontend/src/lib/timeline.js`). They must agree exactly, so
there is a script that proves it. See
[Verifying it actually works](#verifying-it-actually-works).

### Video joins mid-stream

A window an hour into its cycle, or one returning from a sync, must not restart
its video from zero. The player seeks to the position the timeline says it is
at, wrapping if the clip is shorter than its slot. That is what makes two
displays show the same *frame* of the same clip, not just the same file.

Videos are muted and `playsInline` — browsers block audible autoplay, and an
unattended signage display never gets the user interaction that would unblock it.

---

## The 5-hour cycle

Each window's total play size is 5 hours (`CYCLE_SECONDS=18000`). Within that
window the playlist repeats back to back; at the 5-hour boundary every window
restarts from item 0 together.

```
cycle:    |<────────────────── 5 hours ──────────────────>|<── next cycle ──>
playlist: [M1][M2][M4][M1][M2][M4][M1][M2][M4] ... [M1][M2|  restart at [M1]
                                                         ↑
                                     boundary truncates the item in progress
```

**On blank.** The assignment is specific: *"Blank is only a configured playlist
item when included; the rest of the cycle should not become blank playback by
default."*

The cycle length is not required to be a multiple of the playlist length, so the
last pass of a cycle is usually cut short. That leftover is handled by
**truncating the item in progress** at the boundary — never by padding with
blank. Blank appears only where a `blank`-type item is explicitly configured
(window 2 and window 4 each have one, to show it working).

There is one other case that renders blank: a window with an **empty playlist**.
That is reported as `source: "idle"` with an on-screen "No playlist configured"
message — a misconfiguration state, deliberately distinct from cycle padding.

`POST /api/cycle/reset` re-anchors the cycle to now, which restarts every window
at item 0 simultaneously. Set `CYCLE_SECONDS=120` to watch a rollover happen in
two minutes rather than five hours; nothing else about the behaviour changes.

---

## Verifying it actually works

### Go test suite

```bash
cd backend
go vet ./... && go test ./... -count=1
```

18 cases covering: playlist walking and looping, the 5-hour boundary restart,
boundary truncation, clock-behind-anchor handling, sync override and release,
expired syncs not leaking into responses, playlist add/delete, unknown-id
rejection, persistence across a simulated restart, and CORS preflight.

### Go ↔ JS parity

The browser decides what to display on its own, so the two implementations of
`resolve()` must agree. With the backend running:

```bash
cd frontend
node scripts/verify-parity.mjs http://localhost:8080
```

It asks the server what each window should be playing, recomputes the same thing
locally with the frontend's module at the exact same instant, and fails loudly
on any disagreement:

```
  ok   w1  sync/m4 idx=-1
  ok   w2  sync/m4 idx=-1
  ok   w3  sync/m4 idx=-1
  ok   w4  sync/m4 idx=-1

Go and JS timelines agree on every window.
```

### Seeing sync across genuinely separate windows

A single page animating four `<div>`s proves nothing. Click **Detach** on any
window card — it opens `?window=<id>` in a new browser window showing only that
display. Open two or three, arrange them side by side, then trigger a sync from
the control panel. Better still, open one on a second machine on the same
network: that is the case the clock calibration exists for.

### From a terminal

```bash
# what is on window 1 right now, straight from the server's own timeline
curl -s localhost:8080/api/windows/w1/now | python3 -m json.tool

# trigger a sync and watch every window agree
curl -s -X POST localhost:8080/api/sync \
  -H 'Content-Type: application/json' \
  -d '{"mediaId":"m2","durationSeconds":8}'

curl -s localhost:8080/api/state | python3 -c "
import json,sys
d=json.load(sys.stdin)
for p in d['playback']:
    print(p['windowId'], p['source'], p['mediaId'], p['startedAtMs'])
"

# watch the live event stream
curl -N localhost:8080/api/events
```

---

## API reference

All request and response bodies are JSON. **Every duration is milliseconds and
every instant is a Unix epoch in milliseconds** — the frontend is JavaScript
where `Date.now()` is already epoch-ms, so one unit across the wire removes a
whole class of off-by-1000 bugs. Endpoints that accept a duration also accept a
`…Seconds` variant for convenience.

### Reads

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/health` | Liveness, uptime, connected SSE clients |
| `GET` | `/api/time` | `{ "serverTimeMs": … }` — the clock-calibration probe |
| `GET` | `/api/state` | **The main endpoint.** Full snapshot: clock, cycle, media, windows, active sync, and resolved playback for every window |
| `GET` | `/api/media` | Media library |
| `GET` | `/api/windows` | Windows and their playlists |
| `GET` | `/api/windows/{id}/now` | What the server thinks this window is showing, plus `ownSequence` (what it would show ignoring any sync) |

<details>
<summary><code>GET /api/state</code> — response shape</summary>

```json
{
  "serverTimeMs": 1789466864364,
  "cycleMs": 18000000,
  "cycleAnchorMs": 1789466829807,
  "media": [
    {
      "id": "m1",
      "name": "M1",
      "type": "image",
      "url": "https://picsum.photos/seed/sequencer-m1/1280/720",
      "defaultDurationMs": 8000,
      "createdAtMs": 1789466829807
    }
  ],
  "windows": [
    {
      "id": "w1",
      "name": "Window 1 — Lobby",
      "playlist": [
        { "id": "w1-i1", "mediaId": "m1", "durationMs": 8000 }
      ]
    }
  ],
  "activeSync": null,
  "playback": [
    {
      "windowId": "w1",
      "source": "sequence",
      "mediaId": "m1",
      "itemId": "w1-i1",
      "itemIndex": 0,
      "startedAtMs": 1789466829807,
      "endsAtMs": 1789466837807,
      "remainingMs": 283,
      "cycleIndex": 0,
      "offsetInCycleMs": 7717,
      "media": { "…": "resolved library entry" }
    }
  ]
}
```

`source` is one of `sequence`, `sync`, or `idle`.

</details>

### Mutations

| Method | Path | Body | Notes |
|---|---|---|---|
| `POST` | `/api/media` | `{ name, type, url?, defaultDurationSeconds? }` | `type` is `image`, `video` or `blank`. `url` required unless blank. |
| `POST` | `/api/windows` | `{ name }` | Starts with an empty playlist |
| `POST` | `/api/windows/{id}/playlist` | `{ mediaId, durationSeconds?, position? }` | Duration falls back to the media's default. Omit `position` to append. |
| `DELETE` | `/api/windows/{id}/playlist/{itemId}` | — | `204` on success |
| `POST` | `/api/sync` | `{ mediaId, durationSeconds?, startAtMs? }` | Returns the scheduled event. `startAtMs` lets a caller name an exact future instant; otherwise `now + SYNC_LEAD_MS`. |
| `DELETE` | `/api/sync` | — | Cancels an armed or running takeover |
| `POST` | `/api/cycle/reset` | — | Re-anchors the cycle; every window restarts at item 0 |

Errors come back as `{ "error": "…" }` with `400` for validation and `404` for
unknown ids.

### Event stream

`GET /api/events` — Server-Sent Events.

On connect the server immediately pushes a `snapshot` event, so a new display
renders without a second request. After that it emits `playlist.updated`,
`sync.scheduled`, `sync.cancelled`, `media.created`, `window.created` and
`cycle.reset`.

**Every event carries a complete snapshot**, not a delta. A client can therefore
never apply patches out of order or on top of a state it never received. A
heartbeat comment every 20 s keeps intermediaries from culling an idle
connection.

<details>
<summary>Why SSE rather than WebSockets</summary>

- The data flow is strictly server → client; clients mutate over REST. A duplex
  protocol buys nothing here.
- SSE is plain HTTP, so it passes through every proxy, CDN and PaaS router
  without an upgrade handshake or sticky-session configuration.
- `EventSource` reconnects on its own, and because playback is computed from the
  clock, a reconnect gap is invisible to the viewer.
- No third-party dependency, which keeps the Go module stdlib-only.

</details>

---

## Project layout

```
media-sequencer/
├── backend/
│   ├── cmd/server/main.go              entrypoint, graceful shutdown
│   ├── internal/
│   │   ├── api/
│   │   │   ├── server.go               routes, handlers, snapshot assembly
│   │   │   ├── middleware.go           CORS, logging, panic recovery
│   │   │   ├── hub.go                  SSE fan-out
│   │   │   └── server_test.go          HTTP integration tests
│   │   ├── scheduler/
│   │   │   ├── scheduler.go            ★ the timeline algorithm
│   │   │   └── scheduler_test.go       cycle, looping, truncation, sync
│   │   ├── storage/
│   │   │   ├── store.go                the persistence interface
│   │   │   └── jsonstore/              crash-safe file implementation
│   │   ├── models/models.go            domain types
│   │   ├── seed/seed.go                seed windows and media
│   │   └── config/config.go            environment configuration
│   ├── Dockerfile                      API-only image
│   └── fly.toml
├── frontend/
│   ├── src/
│   │   ├── lib/
│   │   │   ├── timeline.js             ★ JS port of the same algorithm
│   │   │   ├── api.js                  HTTP client + clock calibration
│   │   │   └── useSequencer.js         snapshot, SSE, clock state
│   │   ├── components/
│   │   │   ├── WindowPlayer.jsx        one display, self-scheduling
│   │   │   ├── MediaSurface.jsx        image / video / blank, mid-join seek
│   │   │   ├── ControlPanel.jsx        sync trigger and playlist editing
│   │   │   └── StatusBar.jsx           cycle position, clock offset
│   │   └── App.jsx                     grid view + detached window view
│   ├── scripts/verify-parity.mjs       Go ↔ JS agreement check
│   ├── vercel.json / netlify.toml
│   └── vite.config.js
├── Dockerfile.allinone                 single-container: API + UI
├── docker-compose.yml
├── render.yaml
├── Makefile
└── docs/ARCHITECTURE.md
```

The two files marked ★ are the heart of the system and are worth reading first.

---

## Deployment

### Single container (recommended)

One service, one URL, no CORS, SPA routing handled by the Go binary.

```bash
docker build -f Dockerfile.allinone -t media-sequencer .
docker run -p 8080:8080 -v sequencer-data:/app/data media-sequencer
```

This image works as-is on Render, Railway, Fly.io, Google Cloud Run, or any host
that takes a Dockerfile. **Attach a persistent volume at `/app/data`** — without
one, playlists reset on every deploy.

### Two services

`render.yaml` deploys the API (Docker, with a 1 GB disk) and the frontend
(static site) as a blueprint. After the first deploy:

1. Set `VITE_API_BASE_URL` on the static site to the API's public URL, redeploy.
2. Set `ALLOWED_ORIGINS` on the API to the static site's URL.

`frontend/vercel.json` and `frontend/netlify.toml` cover those hosts — build
`npm run build`, publish `dist`, with an SPA rewrite so `?window=w2` deep links
resolve. Set `VITE_API_BASE_URL` in the host's environment settings.

`backend/fly.toml` covers Fly.io:

```bash
cd backend
fly launch --no-deploy --copy-config
fly volumes create sequencer_data --size 1
fly deploy
```

Note `auto_stop_machines = false` — SSE responses are long-lived, and stopping a
machine that looks idle would sever every connected display.

### Deployment checklist

- [ ] Persistent volume mounted at the `DATA_PATH` directory
- [ ] `ALLOWED_ORIGINS` set to the frontend's real origin (not `*`) in production
- [ ] `VITE_API_BASE_URL` set at **build** time if deploying separately — Vite
      inlines it into the bundle, so changing it needs a rebuild, not a restart
- [ ] Health check pointed at `/api/health`
- [ ] Any proxy in front configured not to buffer `text/event-stream`
      (the server already sends `X-Accel-Buffering: no` for nginx)

---

## Configuration

Every value has a working default; nothing must be set to run.

### Backend

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP port. Injected automatically by most PaaS hosts. |
| `DATA_PATH` | `data/state.json` | State file. Created and seeded on first start. |
| `CYCLE_SECONDS` | `18000` | Per-window play size. 18000 = the required 5 hours. Set to `120` for a fast demo. |
| `SYNC_LEAD_MS` | `1200` | How far ahead a sync is scheduled. |
| `SYNC_DEFAULT_SECONDS` | `10` | Fallback sync hold time. |
| `ALLOWED_ORIGINS` | `*` | Comma-separated origins. |
| `STATIC_DIR` | *(empty)* | Point at a built bundle to serve API and UI from one process. |

### Frontend

| Variable | Default | Purpose |
|---|---|---|
| `VITE_API_BASE_URL` | *(empty)* | Empty means same-origin, which covers the dev proxy and the single-container deployment. Set only when deployed separately. |

---

## Assumptions and tradeoffs

### Assumptions

1. **Five hours is the cycle length, not a hard stop.** The assignment says each
   window's "total play size must be treated as 5 hours" and that the list
   restarts after the sequence ends. It is read here as: the playlist loops
   continuously, and at each 5-hour boundary every window realigns to item 0.
   Playback never stops.

2. **The playlist need not divide evenly into 5 hours.** The item in progress at
   the boundary is truncated. Padding the remainder with blank was considered and
   rejected — the brief rules it out explicitly.

3. **A new sync replaces an in-flight one.** Last write wins is the right
   semantic for an operator control: the most recent instruction is the one the
   operator is looking at.

4. **Sync media need not be in any window's playlist.** It is a global takeover,
   so anything in the library is valid.

5. **Clients are within a second or two of real time.** The calibration handles
   ordinary drift. A machine hours off with a firewall blocking `/api/time` would
   degrade to its local clock; playback stays correct, sync accuracy does not.

6. **Media is hosted externally.** Seed data uses public image and video URLs.
   The system stores references, not files — uploads and transcoding are out of
   scope.

7. **Single backend instance.** See the scaling note below.

### Tradeoffs

**Timeline-derived playback instead of server-pushed commands.**
Bought: continuity through outages, correct refresh behaviour, genuine
simultaneity, a trivially scalable server. Cost: the algorithm exists in two
languages and must be kept in step — which is exactly why
`scripts/verify-parity.mjs` exists and runs against a live server.

**JSON file storage instead of SQLite or Postgres.**
This is the choice most worth arguing about, so here is the full reasoning.

The data model is one small aggregate — a handful of windows and media entries —
that is always read and written whole, and mutated by operator actions a few
times a minute. It is not a query workload. There are no joins, no filtering, no
reporting, no growth in row count over time.

A relational engine would add either a cgo toolchain requirement (`mattn/go-sqlite3`)
or an external service to provision (Postgres), and would buy nothing the
workload actually needs.

What *does* matter for correctness is implemented properly:

- all mutations serialised through a single mutex, so no lost updates;
- `Update(fn)` takes a mutation function and applies it under that lock, keeping
  read-modify-write races out of the HTTP handlers entirely;
- every write goes to a temp file → `fsync` → atomic `rename` → directory
  `fsync`, so a crash mid-write cannot leave a truncated or partial file;
- if the mutation or the disk write fails, in-memory state is rolled back, so
  memory and disk never diverge.

Persistence across restart is covered by a test
(`TestStateSurvivesRestart`) and was verified against a real process:

```
w3 playlist after restart: ['m5', 'm1', 'm6']
```

Everything depends on the `storage.Store` interface, never on the driver, so
swapping in Postgres is a new package plus three lines in `main.go`. **If a
Postgres or SQLite implementation is preferred for review purposes, say so —
the interface was designed for exactly that swap and it is a short change.**

**Zero external Go dependencies.**
Bought: instant builds, no supply chain, `go run` works offline, a tiny static
binary. Cost: SSE framing, routing and ID generation are hand-written —
about eighty lines that a library would have provided. Worth it at this size;
it would not be at ten times the scope.

**No authentication.**
Out of scope for the brief. Anyone who can reach the API can trigger a sync.
For a real installation the mutating endpoints would sit behind an operator
login and the display endpoints behind a device token.

### Known limits

**Horizontal scaling.** The design is nearly stateless — clients derive playback
themselves — but two instances would each hold their own JSON file and their own
SSE subscribers, so an edit on one would not reach displays connected to the
other. Fixing it means swapping the store for Postgres and the hub for Redis
pub/sub. Both are behind interfaces already; neither is done, because the brief
calls for one deployment.

**Background tab throttling.** Browsers throttle timers in hidden tabs. A
backgrounded display can drift by a few hundred milliseconds. The app refetches
the snapshot and recalibrates the clock on `visibilitychange`, so it corrects
itself on return. Real signage runs in a foreground kiosk window where this does
not arise.

**Sync lead time is fixed.** 1200 ms is generous for a LAN and adequate for
most internet paths. A display on a connection worse than that would join the
takeover late. Deriving the lead from observed client round trips would be the
next improvement.
