package scheduler

import "github.com/VikasKumar281/SyncStage/backend/internal/models"

const DefaultCycleMs int64 = 5 * 60 * 60 * 1000

type Source string

const (
	SourceSequence Source = "sequence"
	SourceSync     Source = "sync"

	SourceIdle Source = "idle"
)

type Playback struct {
	WindowID        string        `json:"windowId"`
	Source          Source        `json:"source"`
	MediaID         string        `json:"mediaId"`
	ItemID          string        `json:"itemId"`
	ItemIndex       int           `json:"itemIndex"`
	StartedAtMs     int64         `json:"startedAtMs"`
	EndsAtMs        int64         `json:"endsAtMs"`
	RemainingMs     int64         `json:"remainingMs"`
	CycleIndex      int64         `json:"cycleIndex"`
	OffsetInCycleMs int64         `json:"offsetInCycleMs"`
	Media           *models.Media `json:"media"`
}

type Timeline struct {
	CycleMs int64
}

func New() *Timeline { return &Timeline{CycleMs: DefaultCycleMs} }

func NewWithCycle(cycleMs int64) *Timeline {
	if cycleMs <= 0 {
		cycleMs = DefaultCycleMs
	}
	return &Timeline{CycleMs: cycleMs}
}

func floorMod(a, m int64) int64 {
	if m <= 0 {
		return 0
	}
	r := a % m
	if r < 0 {
		r += m
	}
	return r
}

func floorDiv(a, m int64) int64 {
	if m <= 0 {
		return 0
	}
	q := a / m
	if a%m != 0 && (a < 0) != (m < 0) {
		q--
	}
	return q
}

func (t *Timeline) Resolve(state *models.State, w *models.Window, nowMs int64) Playback {
	if state.ActiveSync.IsActiveAt(nowMs) {
		s := state.ActiveSync
		pb := Playback{
			WindowID:    w.ID,
			Source:      SourceSync,
			MediaID:     s.MediaID,
			ItemID:      s.ID,
			ItemIndex:   -1,
			StartedAtMs: s.StartAtMs,
			EndsAtMs:    s.EndsAtMs(),
			RemainingMs: s.EndsAtMs() - nowMs,
		}
		pb.CycleIndex, pb.OffsetInCycleMs = t.cyclePosition(state.CycleAnchorMs, nowMs)
		if m, ok := state.FindMedia(s.MediaID); ok {
			pb.Media = m
		}
		return pb
	}
	return t.ResolveSequence(state, w, nowMs)
}

func (t *Timeline) ResolveSequence(state *models.State, w *models.Window, nowMs int64) Playback {
	cycleIndex, offset := t.cyclePosition(state.CycleAnchorMs, nowMs)
	pb := Playback{
		WindowID:        w.ID,
		Source:          SourceIdle,
		ItemIndex:       -1,
		CycleIndex:      cycleIndex,
		OffsetInCycleMs: offset,
	}

	playlistMs := w.PlaylistDurationMs()
	if len(w.Playlist) == 0 || playlistMs <= 0 {
		pb.StartedAtMs = nowMs
		pb.EndsAtMs = nowMs + (t.CycleMs - offset)
		pb.RemainingMs = pb.EndsAtMs - nowMs
		return pb
	}

	posInPass := floorMod(offset, playlistMs)

	var acc int64
	idx := 0
	for i, item := range w.Playlist {
		if posInPass < acc+item.DurationMs {
			idx = i
			break
		}
		acc += item.DurationMs
		idx = i
	}
	item := w.Playlist[idx]

	itemStart := nowMs - (posInPass - acc)
	itemEnd := itemStart + item.DurationMs

	cycleEnd := nowMs + (t.CycleMs - offset)
	if itemEnd > cycleEnd {
		itemEnd = cycleEnd
	}

	pb.Source = SourceSequence
	pb.MediaID = item.MediaID
	pb.ItemID = item.ID
	pb.ItemIndex = idx
	pb.StartedAtMs = itemStart
	pb.EndsAtMs = itemEnd
	pb.RemainingMs = itemEnd - nowMs
	if m, ok := state.FindMedia(item.MediaID); ok {
		pb.Media = m
	}
	return pb
}

func (t *Timeline) cyclePosition(anchorMs, nowMs int64) (int64, int64) {
	delta := nowMs - anchorMs
	return floorDiv(delta, t.CycleMs), floorMod(delta, t.CycleMs)
}

func (t *Timeline) ResolveAll(state *models.State, nowMs int64) []Playback {
	out := make([]Playback, 0, len(state.Windows))
	for i := range state.Windows {
		out = append(out, t.Resolve(state, &state.Windows[i], nowMs))
	}
	return out
}
