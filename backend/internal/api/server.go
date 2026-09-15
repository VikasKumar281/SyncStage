package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/VikasKumar281/SyncStage/backend/internal/config"
	"github.com/VikasKumar281/SyncStage/backend/internal/models"
	"github.com/VikasKumar281/SyncStage/backend/internal/scheduler"
	"github.com/VikasKumar281/SyncStage/backend/internal/storage"
)

type Server struct {
	cfg      config.Config
	store    storage.Store
	timeline *scheduler.Timeline
	hub      *Hub
	started  time.Time
	now      func() time.Time
}

func NewServer(cfg config.Config, store storage.Store) *Server {
	return &Server{
		cfg:      cfg,
		store:    store,
		timeline: scheduler.NewWithCycle(cfg.CycleMs),
		hub:      NewHub(),
		started:  time.Now(),
		now:      time.Now,
	}
}

func (s *Server) nowMs() int64 { return s.now().UnixMilli() }

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/time", s.handleTime)
	mux.HandleFunc("GET /api/state", s.handleGetState)

	mux.HandleFunc("GET /api/media", s.handleListMedia)
	mux.HandleFunc("POST /api/media", s.handleCreateMedia)

	mux.HandleFunc("GET /api/windows", s.handleListWindows)
	mux.HandleFunc("POST /api/windows", s.handleCreateWindow)
	mux.HandleFunc("GET /api/windows/{windowID}/now", s.handleWindowNow)
	mux.HandleFunc("POST /api/windows/{windowID}/playlist", s.handleAddPlaylistItem)
	mux.HandleFunc("DELETE /api/windows/{windowID}/playlist/{itemID}", s.handleDeletePlaylistItem)

	mux.HandleFunc("POST /api/sync", s.handleTriggerSync)
	mux.HandleFunc("DELETE /api/sync", s.handleCancelSync)

	mux.HandleFunc("POST /api/cycle/reset", s.handleResetCycle)

	mux.HandleFunc("GET /api/events", s.handleEvents)

	s.mountStatic(mux)

	return withRecovery(withLogging(withCORS(s.cfg.AllowedOrigins, mux)))
}

func (s *Server) mountStatic(mux *http.ServeMux) {
	if s.cfg.StaticDir == "" {
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{
				"service": "SyncStage",
				"docs":    "GET /api/state to begin; see README.md for the full API",
			})
		})
		return
	}

	root := s.cfg.StaticDir
	fileServer := http.FileServer(http.Dir(root))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if clean == "." || clean == "/" {
			http.ServeFile(w, r, filepath.Join(root, "index.html"))
			return
		}
		if _, err := os.Stat(filepath.Join(root, clean)); err != nil {
			http.ServeFile(w, r, filepath.Join(root, "index.html"))
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

type snapshot struct {
	ServerTimeMs  int64                `json:"serverTimeMs"`
	CycleMs       int64                `json:"cycleMs"`
	CycleAnchorMs int64                `json:"cycleAnchorMs"`
	Media         []models.Media       `json:"media"`
	Windows       []models.Window      `json:"windows"`
	ActiveSync    *models.SyncEvent    `json:"activeSync"`
	Playback      []scheduler.Playback `json:"playback"`
	Revision      string               `json:"revision"`
}

func (s *Server) snapshotOf(state *models.State) snapshot {
	now := s.nowMs()

	active := state.ActiveSync
	if !active.IsPendingOrActiveAt(now) {
		active = nil
	}
	view := state.Clone()
	view.ActiveSync = active

	return snapshot{
		ServerTimeMs:  now,
		CycleMs:       s.cfg.CycleMs,
		CycleAnchorMs: state.CycleAnchorMs,
		Media:         view.Media,
		Windows:       view.Windows,
		ActiveSync:    active,
		Playback:      s.timeline.ResolveAll(view, now),
		Revision:      fmt.Sprintf("%d", now),
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"uptimeSeconds": int64(time.Since(s.started).Seconds()),
		"serverTimeMs":  s.nowMs(),
		"sseClients":    s.hub.Count(),
		"cycleMs":       s.cfg.CycleMs,
	})
}

func (s *Server) handleTime(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]int64{"serverTimeMs": s.nowMs()})
}

func (s *Server) handleGetState(w http.ResponseWriter, r *http.Request) {
	state, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, s.snapshotOf(state))
}

func (s *Server) handleListMedia(w http.ResponseWriter, r *http.Request) {
	state, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"media": state.Media})
}

type createMediaRequest struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	URL               string `json:"url"`
	DefaultDurationMs *int64 `json:"defaultDurationMs"`
	DefaultDurationS  *int64 `json:"defaultDurationSeconds"`
}

func (s *Server) handleCreateMedia(w http.ResponseWriter, r *http.Request) {
	var req createMediaRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	duration := int64(8000)
	if req.DefaultDurationS != nil {
		duration = *req.DefaultDurationS * 1000
	}
	if req.DefaultDurationMs != nil {
		duration = *req.DefaultDurationMs
	}

	media := models.Media{
		ID:                newID("med"),
		Name:              strings.TrimSpace(req.Name),
		Type:              models.MediaType(strings.TrimSpace(strings.ToLower(req.Type))),
		URL:               strings.TrimSpace(req.URL),
		DefaultDurationMs: duration,
		CreatedAtMs:       s.nowMs(),
	}
	if media.Type == models.MediaBlank {
		media.URL = ""
	}
	if err := media.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	state, err := s.store.Update(func(st *models.State) error {
		st.Media = append(st.Media, media)
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.broadcastState(state, "media.created")
	writeJSON(w, http.StatusCreated, map[string]any{"media": media})
}

func (s *Server) handleListWindows(w http.ResponseWriter, r *http.Request) {
	state, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"windows": state.Windows})
}

type createWindowRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateWindow(w http.ResponseWriter, r *http.Request) {
	var req createWindowRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}

	win := models.Window{ID: newID("win"), Name: name, Playlist: []models.PlaylistItem{}}
	state, err := s.store.Update(func(st *models.State) error {
		st.Windows = append(st.Windows, win)
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.broadcastState(state, "window.created")
	writeJSON(w, http.StatusCreated, map[string]any{"window": win})
}

// handleWindowNow answers "what is on this window right now" straight from the
// server's own copy of the timeline. It is not used by the player (the player
// computes this locally) — it exists so the Go and JS implementations can be
// compared, and so the behaviour is verifiable from a terminal.
func (s *Server) handleWindowNow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("windowID")
	state, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	win, ok := state.FindWindow(id)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("window %q not found", id))
		return
	}
	now := s.nowMs()
	writeJSON(w, http.StatusOK, map[string]any{
		"serverTimeMs": now,
		"playback":     s.timeline.Resolve(state, win, now),
		"ownSequence":  s.timeline.ResolveSequence(state, win, now),
	})
}

type addPlaylistItemRequest struct {
	MediaID    string `json:"mediaId"`
	DurationMs *int64 `json:"durationMs"`
	DurationS  *int64 `json:"durationSeconds"`
	// Position inserts at an index; nil appends.
	Position *int `json:"position"`
}

func (s *Server) handleAddPlaylistItem(w http.ResponseWriter, r *http.Request) {
	windowID := r.PathValue("windowID")

	var req addPlaylistItemRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.MediaID) == "" {
		writeError(w, http.StatusBadRequest, errors.New("mediaId is required"))
		return
	}

	var created models.PlaylistItem
	state, err := s.store.Update(func(st *models.State) error {
		win, ok := st.FindWindow(windowID)
		if !ok {
			return fmt.Errorf("%w: window %q", storage.ErrNotFound, windowID)
		}
		media, ok := st.FindMedia(req.MediaID)
		if !ok {
			return fmt.Errorf("%w: media %q", storage.ErrNotFound, req.MediaID)
		}

		duration := media.DefaultDurationMs
		if req.DurationS != nil {
			duration = *req.DurationS * 1000
		}
		if req.DurationMs != nil {
			duration = *req.DurationMs
		}
		if duration <= 0 {
			return errors.New("duration must be greater than zero")
		}

		created = models.PlaylistItem{ID: newID("itm"), MediaID: media.ID, DurationMs: duration}

		pos := len(win.Playlist)
		if req.Position != nil && *req.Position >= 0 && *req.Position < len(win.Playlist) {
			pos = *req.Position
		}
		win.Playlist = append(win.Playlist, models.PlaylistItem{})
		copy(win.Playlist[pos+1:], win.Playlist[pos:])
		win.Playlist[pos] = created
		return nil
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	// Every window recomputes from the snapshot, so a playlist edit lands
	// without interrupting playback anywhere else.
	s.broadcastState(state, "playlist.updated")
	writeJSON(w, http.StatusCreated, map[string]any{"item": created})
}

func (s *Server) handleDeletePlaylistItem(w http.ResponseWriter, r *http.Request) {
	windowID := r.PathValue("windowID")
	itemID := r.PathValue("itemID")

	state, err := s.store.Update(func(st *models.State) error {
		win, ok := st.FindWindow(windowID)
		if !ok {
			return fmt.Errorf("%w: window %q", storage.ErrNotFound, windowID)
		}
		for i, item := range win.Playlist {
			if item.ID == itemID {
				win.Playlist = append(win.Playlist[:i], win.Playlist[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("%w: item %q", storage.ErrNotFound, itemID)
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	s.broadcastState(state, "playlist.updated")
	w.WriteHeader(http.StatusNoContent)
}

type triggerSyncRequest struct {
	MediaID    string `json:"mediaId"`
	DurationMs *int64 `json:"durationMs"`
	DurationS  *int64 `json:"durationSeconds"`
	// StartAtMs lets a caller schedule a sync at an exact future instant.
	StartAtMs *int64 `json:"startAtMs"`
}

func (s *Server) handleTriggerSync(w http.ResponseWriter, r *http.Request) {
	var req triggerSyncRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.MediaID) == "" {
		writeError(w, http.StatusBadRequest, errors.New("mediaId is required"))
		return
	}

	now := s.nowMs()
	duration := s.cfg.DefaultSyncDurationMs
	if req.DurationS != nil {
		duration = *req.DurationS * 1000
	}
	if req.DurationMs != nil {
		duration = *req.DurationMs
	}
	if duration <= 0 {
		writeError(w, http.StatusBadRequest, errors.New("sync duration must be greater than zero"))
		return
	}

	startAt := now + s.cfg.SyncLeadMs
	if req.StartAtMs != nil && *req.StartAtMs > now {
		startAt = *req.StartAtMs
	}

	event := models.SyncEvent{
		ID:            newID("syn"),
		MediaID:       req.MediaID,
		StartAtMs:     startAt,
		DurationMs:    duration,
		TriggeredAtMs: now,
	}

	state, err := s.store.Update(func(st *models.State) error {
		if _, ok := st.FindMedia(event.MediaID); !ok {
			return fmt.Errorf("%w: media %q", storage.ErrNotFound, event.MediaID)
		}

		st.ActiveSync = &event
		return nil
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	s.broadcastState(state, "sync.scheduled")
	writeJSON(w, http.StatusAccepted, map[string]any{
		"sync":         event,
		"serverTimeMs": now,
	})
}

func (s *Server) handleCancelSync(w http.ResponseWriter, r *http.Request) {
	state, err := s.store.Update(func(st *models.State) error {
		st.ActiveSync = nil
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.broadcastState(state, "sync.cancelled")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleResetCycle(w http.ResponseWriter, r *http.Request) {
	state, err := s.store.Update(func(st *models.State) error {
		st.CycleAnchorMs = s.nowMs()
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.broadcastState(state, "cycle.reset")
	writeJSON(w, http.StatusOK, s.snapshotOf(state))
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New("streaming unsupported"))
		return
	}

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache, no-transform")
	h.Set("Connection", "keep-alive")

	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	ch, unsubscribe := s.hub.Subscribe()
	defer unsubscribe()

	fmt.Fprint(w, "retry: 2000\n\n")

	if state, err := s.store.Load(); err == nil {
		if data, merr := json.Marshal(s.snapshotOf(state)); merr == nil {
			writeSSE(w, "snapshot", data)
		}
	}
	flusher.Flush()

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, open := <-ch:
			if !open {
				return
			}
			writeSSE(w, evt.Name, evt.Data)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) broadcastState(state *models.State, eventName string) {
	s.hub.Broadcast(eventName, s.snapshotOf(state))
}

func writeSSE(w http.ResponseWriter, name string, data []byte) {
	fmt.Fprintf(w, "event: %s\n", name)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

func decodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("api: write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeDomainError(w http.ResponseWriter, err error) {
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeError(w, http.StatusBadRequest, err)
}

func newID(prefix string) string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(buf)
}
