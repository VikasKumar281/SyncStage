# Architecture

Supplementary notes to the [README](../README.md). This file covers the design
decisions in more depth than a setup guide should.

---

## The central idea

Everything in this system follows from one decision:

> **Playback is a pure function of time, not a stream of commands.**

```
resolve(cycleAnchor, playlist, activeSync, t) → { media, startedAt, endsAt }
```

No hidden state, no accumulated position, no "current index" stored anywhere.
Hand the function the same inputs on any machine and it returns the same answer.

Once playback is a pure function, most of the hard problems in the brief stop
being problems:

| Problem | How it dissolves |
|---|---|
| Continuous playback without stopping | Nothing to stop — the function always has an answer |
| Windows staying in step | They evaluate the same function against the same clock |
| Sync across all windows | A time interval during which the function returns the same item everywhere |
| Resuming after sync without losing position | Position was never stored, so it cannot be lost |
| Surviving reconnects and refreshes | Recompute from the clock; nothing to recover |
| 5-hour cycle restart | One modulo in the function |

The remaining work is making sure every participant agrees on `t`, and telling
clients when the *inputs* change.

---

## Layers

```
┌───────────────────── browser ─────────────────────┐
│  WindowPlayer ×N                                  │
│     └─ timeline.js  resolve(snapshot, win, now)   │
│  useSequencer                                     │
│     ├─ EventSource /api/events   (input changes)  │
│     └─ clock offset via /api/time (agree on `t`)  │
└───────────────────────┬───────────────────────────┘
                        │ REST + SSE
┌───────────────────────▼───────────────────────────┐
│  api      routing, handlers, SSE hub              │
│  scheduler  resolve() — same algorithm, in Go     │
│  storage    Store interface                       │
│     └─ jsonstore  atomic, crash-safe file         │
└───────────────────────────────────────────────────┘
```

Dependencies point inward. `scheduler` imports only `models`; `api` depends on
`storage` through its interface and never on `jsonstore`.

---

## Why the algorithm is duplicated

`resolve()` exists in Go and in JavaScript. Duplicated logic is normally a
defect, so the reasoning deserves stating.

The frontend **must** have it. If the browser asked the server what to display,
every switch would cost a round trip and a network hiccup would freeze a
display. The whole robustness argument depends on the client computing locally.

The backend **should** have it, for three reasons:

1. `/api/state` ships resolved playback with the snapshot, so a new display
   renders on first paint rather than after its own computation.
2. `/api/windows/{id}/now` makes the system verifiable from a terminal, with no
   browser involved.
3. It gives the JS implementation something to be tested against.

The risk is drift between them. That is handled directly:
`frontend/scripts/verify-parity.mjs` asks the server what each window should be
playing, recomputes locally at the *same instant the server used*, and fails on
any disagreement. It is run in three states — clean sequence, mid-sync, and
after a playlist edit.

If this grew further, the honest fix would be to compile the Go scheduler to
WebAssembly and delete the JS copy. At ~120 lines with a parity check, two
copies is the cheaper trade.

---

## Time handling in detail

### Units

Milliseconds everywhere — durations and instants alike. `Date.now()` is already
epoch-ms, so a single unit across the wire eliminates conversion bugs at the
boundary. Human-facing seconds are accepted on input (`durationSeconds`) and
converted once, at the edge.

### Negative modulo

Go's `%` keeps the sign of the dividend, and JavaScript's does the same. A
client whose clock sits momentarily *behind* the cycle anchor would produce a
negative offset and index off the front of the playlist. Both implementations
use a `floorMod` helper that always returns a non-negative result, and there is
a test for it (`TestClockBehindAnchorDoesNotPanicOrGoNegative`).

### Boundary cushion

The switch timer fires at `endsAt - now + 20ms`. Without the cushion, floating
point and timer imprecision can land the recomputation exactly on the seam,
where the outgoing item resolves one extra time and produces a visible stutter.
Twenty milliseconds is below the threshold of perception and comfortably above
the error.

### Cristian's algorithm

```
sentAt     = Date.now()
serverTime = GET /api/time
receivedAt = Date.now()

roundTrip  = receivedAt - sentAt
offset     = serverTime - (sentAt + roundTrip / 2)
```

The midpoint assumption — that the response was generated halfway through the
round trip — is wrong whenever the path is asymmetric. Taking five samples and
keeping the one with the smallest round trip minimises how wrong it can be,
since a fast probe leaves little room for asymmetry to hide in.

Recalibration runs every two minutes and on `visibilitychange`.

---

## Concurrency

**Store.** All mutations funnel through `Update(fn)`, which holds a mutex for
the whole read-modify-write. Handlers never see a race because they never do
the read and the write as separate steps. The mutation runs against a clone; if
it returns an error, or the disk write fails, the in-memory state is rolled back
so memory and disk cannot diverge.

**Hub.** Subscribers are held under an `RWMutex`; broadcasts take the read lock.
Each subscriber has a 16-slot buffered channel and a non-blocking send. A client
too slow to drain its buffer gets events dropped rather than being allowed to
block the broadcaster — correctness for everyone beats completeness for one. The
dropped client reconnects and gets a fresh snapshot, so nothing is permanently
lost.

**Snapshot coherence.** `ResolveAll` takes one `nowMs` and passes it to every
window, rather than calling `time.Now()` per window. Without that, a snapshot
could show windows resolved at microscopically different instants — enough to
report two windows on opposite sides of a switch.

---

## Durability

```
marshal → temp file in same dir → write → fsync → close → rename → fsync dir
```

`rename(2)` within a filesystem is atomic: a reader sees either the old file or
the new one, never a partial write. The temp file is created in the destination
directory specifically so the rename stays within one filesystem. The final
directory `fsync` makes the rename itself durable, not just the file contents.

This is the same sequence a database WAL uses for its checkpoint, and it is the
part of "use a real database" that actually matters at this data size.

---

## Swapping the storage driver

The interface is four methods:

```go
type Store interface {
	Load() (*models.State, error)
	Update(mutate func(*models.State) error) (*models.State, error)
	Close() error
}
```

A Postgres implementation would:

1. Keep the aggregate in a single `jsonb` column, or normalise into
   `media` / `windows` / `playlist_items` / `sync_events` tables.
2. Implement `Update` as `BEGIN; SELECT … FOR UPDATE; …; COMMIT;` — the row lock
   replaces the mutex and, unlike the mutex, works across instances.
3. Change three lines in `cmd/server/main.go`.

Nothing else in the codebase would change, because nothing else imports
`jsonstore`.

---

## Scaling past one instance

The design is most of the way there already: clients compute playback
themselves, so the server handles configuration reads and writes and nothing
per-frame. Two gaps remain:

1. **Shared state** — each instance would own its own JSON file. Fix: Postgres,
   per above.
2. **Cross-instance fan-out** — an edit on instance A must reach displays
   connected to instance B. Fix: Redis pub/sub behind the `Hub` API, which is
   already just `Subscribe`, `Broadcast`, `Count`.

The cycle anchor is stored data, so every instance would derive identical
timelines automatically. No leader election, no coordination.

Neither is implemented, because the brief specifies one deployment and building
for imagined scale is its own kind of mistake.

---

## What was considered and rejected

**WebSocket transport.** Duplex is unnecessary — clients mutate over REST. SSE
is plain HTTP, survives every proxy without an upgrade handshake, reconnects
automatically, and needs no dependency.

**Delta events.** Sending only what changed would be smaller, but it introduces
ordering and missed-message failure modes for a payload measured in kilobytes.
Full snapshots make the client a pure function of the last message it received.

**A server-driven "now playing" push.** The design the whole architecture exists
to avoid. Discussed at length in the README.

**Padding the cycle remainder with blank.** Ruled out explicitly by the brief.
The boundary truncates the item in progress instead.

**Storing each window's current index.** This is the one that quietly ruins the
sync requirement: with a stored index you need save-and-restore logic around
every takeover, and any missed message leaves a window permanently offset.
Deriving position from the clock means there is nothing to save and nothing to
get wrong.
