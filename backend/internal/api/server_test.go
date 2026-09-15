package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/VikasKumar281/SyncStage/backend/internal/config"
	"github.com/VikasKumar281/SyncStage/backend/internal/models"
	"github.com/VikasKumar281/SyncStage/backend/internal/seed"
	"github.com/VikasKumar281/SyncStage/backend/internal/storage/jsonstore"
)

func newTestServer(t *testing.T) (http.Handler, *Server) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state.json")
	store, err := jsonstore.Open(path, func() *models.State { return seed.Default(time.Now()) })
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := config.Config{
		Port:                  "0",
		DataPath:              path,
		CycleMs:               5 * 60 * 60 * 1000,
		SyncLeadMs:            1200,
		DefaultSyncDurationMs: 10_000,
		AllowedOrigins:        []string{"*"},
	}
	srv := NewServer(cfg, store)
	return srv.Handler(), srv
}

func do(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestStateSnapshotIsSeededAndComplete(t *testing.T) {
	h, _ := newTestServer(t)
	rec := do(t, h, http.MethodGet, "/api/state", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var snap snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(snap.Windows) != 4 {
		t.Fatalf("expected 4 seeded windows, got %d", len(snap.Windows))
	}
	if len(snap.Media) != 7 {
		t.Fatalf("expected 7 seeded media, got %d", len(snap.Media))
	}
	if len(snap.Playback) != 4 {
		t.Fatalf("expected playback for every window, got %d", len(snap.Playback))
	}
	if snap.ServerTimeMs == 0 || snap.CycleMs != 5*60*60*1000 {
		t.Fatalf("clock/cycle not reported: %+v", snap)
	}
	for _, pb := range snap.Playback {
		if pb.RemainingMs <= 0 {
			t.Fatalf("window %s has non-positive remaining time", pb.WindowID)
		}
	}
}

func TestAddPlaylistItemPersistsAndAppears(t *testing.T) {
	h, _ := newTestServer(t)

	rec := do(t, h, http.MethodPost, "/api/windows/w3/playlist", map[string]any{
		"mediaId":         seed.M2,
		"durationSeconds": 12,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = do(t, h, http.MethodGet, "/api/state", nil)
	var snap snapshot
	_ = json.Unmarshal(rec.Body.Bytes(), &snap)
	for _, w := range snap.Windows {
		if w.ID != "w3" {
			continue
		}
		if len(w.Playlist) != 3 {
			t.Fatalf("expected 3 items in w3, got %d", len(w.Playlist))
		}
		last := w.Playlist[2]
		if last.MediaID != seed.M2 || last.DurationMs != 12_000 {
			t.Fatalf("appended item wrong: %+v", last)
		}
		return
	}
	t.Fatal("w3 missing from snapshot")
}

func TestAddPlaylistItemRejectsUnknownIDs(t *testing.T) {
	h, _ := newTestServer(t)

	if rec := do(t, h, http.MethodPost, "/api/windows/nope/playlist",
		map[string]any{"mediaId": seed.M1}); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown window: expected 404, got %d", rec.Code)
	}
	if rec := do(t, h, http.MethodPost, "/api/windows/w1/playlist",
		map[string]any{"mediaId": "ghost"}); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown media: expected 404, got %d", rec.Code)
	}
	if rec := do(t, h, http.MethodPost, "/api/windows/w1/playlist",
		map[string]any{}); rec.Code != http.StatusBadRequest {
		t.Fatalf("missing mediaId: expected 400, got %d", rec.Code)
	}
}

func TestDeletePlaylistItem(t *testing.T) {
	h, _ := newTestServer(t)
	if rec := do(t, h, http.MethodDelete, "/api/windows/w1/playlist/w1-i2", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	rec := do(t, h, http.MethodGet, "/api/state", nil)
	var snap snapshot
	_ = json.Unmarshal(rec.Body.Bytes(), &snap)
	for _, w := range snap.Windows {
		if w.ID == "w1" && len(w.Playlist) != 2 {
			t.Fatalf("expected 2 remaining items, got %d", len(w.Playlist))
		}
	}
}

func TestSyncIsScheduledInTheFutureAndHitsEveryWindow(t *testing.T) {
	h, srv := newTestServer(t)

	before := srv.nowMs()
	rec := do(t, h, http.MethodPost, "/api/sync", map[string]any{
		"mediaId":         seed.M2,
		"durationSeconds": 6,
	})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Sync models.SyncEvent `json:"sync"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// The whole point of the lead time: the start instant must not be in the past.
	if resp.Sync.StartAtMs <= before {
		t.Fatalf("sync must be scheduled ahead of now; start=%d now=%d", resp.Sync.StartAtMs, before)
	}
	if resp.Sync.DurationMs != 6_000 {
		t.Fatalf("duration not honoured: %d", resp.Sync.DurationMs)
	}

	// Resolve every window at an instant inside the sync window.
	state, _ := srv.store.Load()
	mid := resp.Sync.StartAtMs + 1_000
	for i := range state.Windows {
		pb := srv.timeline.Resolve(state, &state.Windows[i], mid)
		if pb.MediaID != seed.M2 {
			t.Fatalf("window %s shows %q during sync, expected M2", pb.WindowID, pb.MediaID)
		}
	}
	// ...and that they all agree on the exact switch instant.
	first := srv.timeline.Resolve(state, &state.Windows[0], mid)
	for i := range state.Windows {
		pb := srv.timeline.Resolve(state, &state.Windows[i], mid)
		if pb.StartedAtMs != first.StartedAtMs || pb.EndsAtMs != first.EndsAtMs {
			t.Fatalf("window %s out of lockstep: %d-%d vs %d-%d",
				pb.WindowID, pb.StartedAtMs, pb.EndsAtMs, first.StartedAtMs, first.EndsAtMs)
		}
	}
}

func TestSyncRejectsUnknownMedia(t *testing.T) {
	h, _ := newTestServer(t)
	if rec := do(t, h, http.MethodPost, "/api/sync",
		map[string]any{"mediaId": "ghost"}); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestExpiredSyncIsNotReportedAsActive(t *testing.T) {
	h, srv := newTestServer(t)
	_, err := srv.store.Update(func(st *models.State) error {
		st.ActiveSync = &models.SyncEvent{
			ID: "old", MediaID: seed.M1,
			StartAtMs: srv.nowMs() - 60_000, DurationMs: 5_000,
		}
		return nil
	})
	if err != nil {
		t.Fatalf("seed sync: %v", err)
	}

	rec := do(t, h, http.MethodGet, "/api/state", nil)
	var snap snapshot
	_ = json.Unmarshal(rec.Body.Bytes(), &snap)
	if snap.ActiveSync != nil {
		t.Fatalf("expired sync leaked into snapshot: %+v", snap.ActiveSync)
	}
	for _, pb := range snap.Playback {
		if pb.Source != "sequence" {
			t.Fatalf("window %s should be back on its own sequence, got %s", pb.WindowID, pb.Source)
		}
	}
}

func TestCreateMediaValidates(t *testing.T) {
	h, _ := newTestServer(t)

	if rec := do(t, h, http.MethodPost, "/api/media", map[string]any{
		"name": "M9", "type": "image", "url": "https://example.com/a.jpg", "defaultDurationSeconds": 9,
	}); rec.Code != http.StatusCreated {
		t.Fatalf("valid media rejected: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, h, http.MethodPost, "/api/media", map[string]any{
		"name": "bad", "type": "hologram", "url": "https://x",
	}); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad type: expected 400, got %d", rec.Code)
	}
	if rec := do(t, h, http.MethodPost, "/api/media", map[string]any{
		"name": "no url", "type": "video",
	}); rec.Code != http.StatusBadRequest {
		t.Fatalf("missing url: expected 400, got %d", rec.Code)
	}
	// Blank media needs no URL.
	if rec := do(t, h, http.MethodPost, "/api/media", map[string]any{
		"name": "Gap", "type": "blank", "defaultDurationSeconds": 3,
	}); rec.Code != http.StatusCreated {
		t.Fatalf("blank media rejected: %d %s", rec.Code, rec.Body.String())
	}
}

func TestStateSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	open := func() http.Handler {
		store, err := jsonstore.Open(path, func() *models.State { return seed.Default(time.Now()) })
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		t.Cleanup(func() { _ = store.Close() })
		return NewServer(config.Config{
			CycleMs: 18_000_000, SyncLeadMs: 1200, DefaultSyncDurationMs: 10_000,
			AllowedOrigins: []string{"*"},
		}, store).Handler()
	}

	h1 := open()
	if rec := do(t, h1, http.MethodPost, "/api/windows/w3/playlist",
		map[string]any{"mediaId": seed.M6, "durationSeconds": 11}); rec.Code != http.StatusCreated {
		t.Fatalf("add failed: %d", rec.Code)
	}

	// Fresh process, same file.
	h2 := open()
	rec := do(t, h2, http.MethodGet, "/api/state", nil)
	var snap snapshot
	_ = json.Unmarshal(rec.Body.Bytes(), &snap)
	for _, w := range snap.Windows {
		if w.ID == "w3" {
			if len(w.Playlist) != 3 {
				t.Fatalf("playlist did not persist across restart: %d items", len(w.Playlist))
			}
			return
		}
	}
	t.Fatal("w3 missing after restart")
}

func TestCORSPreflight(t *testing.T) {
	h, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodOptions, "/api/state", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing CORS header: %v", rec.Header())
	}
}

func TestWindowNowEndpoint(t *testing.T) {
	h, _ := newTestServer(t)
	rec := do(t, h, http.MethodGet, "/api/windows/w1/now", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Playback struct {
			Source      string `json:"source"`
			RemainingMs int64  `json:"remainingMs"`
		} `json:"playback"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Playback.Source != "sequence" || body.Playback.RemainingMs <= 0 {
		t.Fatalf("unexpected playback: %+v", body.Playback)
	}
	if rec := do(t, h, http.MethodGet, "/api/windows/ghost/now", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown window, got %d", rec.Code)
	}
}
