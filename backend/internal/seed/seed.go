package seed

import (
	"time"

	"github.com/VikasKumar281/SyncStage/backend/internal/models"
)

const (
	sec = int64(1000)
)

const (
	M1    = "m1"
	M2    = "m2"
	M3    = "m3"
	M4    = "m4"
	M5    = "m5"
	M6    = "m6"
	Blank = "m-blank"
)

func Default(now time.Time) *models.State {
	nowMs := now.UnixMilli()

	media := []models.Media{
		{
			ID: M1, Name: "M1", Type: models.MediaImage,
			URL:               "https://picsum.photos/seed/sequencer-m1/1280/720",
			DefaultDurationMs: 8 * sec, CreatedAtMs: nowMs,
		},
		{
			ID: M2, Name: "M2", Type: models.MediaImage,
			URL:               "https://picsum.photos/seed/sequencer-m2/1280/720",
			DefaultDurationMs: 8 * sec, CreatedAtMs: nowMs,
		},
		{
			ID: M3, Name: "M3", Type: models.MediaImage,
			URL:               "https://picsum.photos/seed/sequencer-m3/1280/720",
			DefaultDurationMs: 8 * sec, CreatedAtMs: nowMs,
		},
		{
			ID: M4, Name: "M4 (video)", Type: models.MediaVideo,
			URL:               "/videos/m4.mp4",
			DefaultDurationMs: 15 * sec, CreatedAtMs: nowMs,
		},
		{
			ID: M5, Name: "M5 (video)", Type: models.MediaVideo,
			URL:               "/videos/m5.mp4",
			DefaultDurationMs: 15 * sec, CreatedAtMs: nowMs,
		},
		{
			ID: M6, Name: "M6", Type: models.MediaImage,
			URL:               "https://picsum.photos/seed/sequencer-m6/1280/720",
			DefaultDurationMs: 10 * sec, CreatedAtMs: nowMs,
		},
		{
			ID: Blank, Name: "Blank", Type: models.MediaBlank,
			DefaultDurationMs: 5 * sec, CreatedAtMs: nowMs,
		},
	}

	windows := []models.Window{
		{
			ID: "w1", Name: "Window 1 — Lobby",
			Playlist: []models.PlaylistItem{
				{ID: "w1-i1", MediaID: M1, DurationMs: 8 * sec},
				{ID: "w1-i2", MediaID: M2, DurationMs: 8 * sec},
				{ID: "w1-i3", MediaID: M4, DurationMs: 15 * sec},
			},
		},
		{
			ID: "w2", Name: "Window 2 — Reception",
			Playlist: []models.PlaylistItem{
				{ID: "w2-i1", MediaID: M3, DurationMs: 10 * sec},
				{ID: "w2-i2", MediaID: Blank, DurationMs: 5 * sec},
				{ID: "w2-i3", MediaID: M2, DurationMs: 8 * sec},
				{ID: "w2-i4", MediaID: M6, DurationMs: 10 * sec},
			},
		},
		{
			ID: "w3", Name: "Window 3 — Cafeteria",
			Playlist: []models.PlaylistItem{
				{ID: "w3-i1", MediaID: M5, DurationMs: 15 * sec},
				{ID: "w3-i2", MediaID: M1, DurationMs: 6 * sec},
			},
		},
		{
			ID: "w4", Name: "Window 4 — Corridor",
			Playlist: []models.PlaylistItem{
				{ID: "w4-i1", MediaID: M6, DurationMs: 10 * sec},
				{ID: "w4-i2", MediaID: M3, DurationMs: 7 * sec},
				{ID: "w4-i3", MediaID: M2, DurationMs: 7 * sec},
				{ID: "w4-i4", MediaID: M4, DurationMs: 15 * sec},
				{ID: "w4-i5", MediaID: Blank, DurationMs: 4 * sec},
			},
		},
	}

	return &models.State{
		CycleAnchorMs: nowMs,
		Media:         media,
		Windows:       windows,
		ActiveSync:    nil,
	}
}