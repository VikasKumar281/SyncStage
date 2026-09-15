package models

import "errors"

type MediaType string

const (
	MediaImage MediaType = "image"
	MediaVideo MediaType = "video"

	MediaBlank MediaType = "blank"
)

func (m MediaType) Valid() bool {
	switch m {
	case MediaImage, MediaVideo, MediaBlank:
		return true
	}
	return false
}

type Media struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Type              MediaType `json:"type"`
	URL               string    `json:"url"`
	DefaultDurationMs int64     `json:"defaultDurationMs"`
	CreatedAtMs       int64     `json:"createdAtMs"`
}

func (m *Media) Validate() error {
	if m.Name == "" {
		return errors.New("media: name is required")
	}
	if !m.Type.Valid() {
		return errors.New("media: type must be image, video or blank")
	}
	if m.Type != MediaBlank && m.URL == "" {
		return errors.New("media: url is required for image and video media")
	}
	if m.DefaultDurationMs <= 0 {
		return errors.New("media: defaultDurationMs must be greater than zero")
	}
	return nil
}

type PlaylistItem struct {
	ID         string `json:"id"`
	MediaID    string `json:"mediaId"`
	DurationMs int64  `json:"durationMs"`
}

type Window struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Playlist []PlaylistItem `json:"playlist"`
}

func (w *Window) PlaylistDurationMs() int64 {
	var total int64
	for _, item := range w.Playlist {
		total += item.DurationMs
	}
	return total
}

type SyncEvent struct {
	ID            string `json:"id"`
	MediaID       string `json:"mediaId"`
	StartAtMs     int64  `json:"startAtMs"`
	DurationMs    int64  `json:"durationMs"`
	TriggeredAtMs int64  `json:"triggeredAtMs"`
}

func (s *SyncEvent) EndsAtMs() int64 { return s.StartAtMs + s.DurationMs }

func (s *SyncEvent) IsActiveAt(nowMs int64) bool {
	if s == nil {
		return false
	}
	return nowMs >= s.StartAtMs && nowMs < s.EndsAtMs()
}

func (s *SyncEvent) IsPendingOrActiveAt(nowMs int64) bool {
	if s == nil {
		return false
	}
	return nowMs < s.EndsAtMs()
}

type State struct {
	CycleAnchorMs int64      `json:"cycleAnchorMs"`
	Media         []Media    `json:"media"`
	Windows       []Window   `json:"windows"`
	ActiveSync    *SyncEvent `json:"activeSync"`
}

func (s *State) FindMedia(id string) (*Media, bool) {
	for i := range s.Media {
		if s.Media[i].ID == id {
			return &s.Media[i], true
		}
	}
	return nil, false
}

func (s *State) FindWindow(id string) (*Window, bool) {
	for i := range s.Windows {
		if s.Windows[i].ID == id {
			return &s.Windows[i], true
		}
	}
	return nil, false
}

func (s *State) Clone() *State {
	if s == nil {
		return nil
	}
	out := &State{
		CycleAnchorMs: s.CycleAnchorMs,
		Media:         append([]Media(nil), s.Media...),
		Windows:       make([]Window, len(s.Windows)),
	}
	for i, w := range s.Windows {
		out.Windows[i] = Window{
			ID:       w.ID,
			Name:     w.Name,
			Playlist: append([]PlaylistItem(nil), w.Playlist...),
		}
	}
	if s.ActiveSync != nil {
		sync := *s.ActiveSync
		out.ActiveSync = &sync
	}
	return out
}
