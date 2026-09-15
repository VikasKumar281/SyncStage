
package jsonstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/vikas/media-sequencer/backend/internal/models"
)

type Store struct {
	mu    sync.Mutex
	path  string
	state *models.State
}


func Open(path string, defaultState func() *models.State) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("jsonstore: create data dir: %w", err)
	}

	s := &Store{path: path}

	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		var st models.State
		if uerr := json.Unmarshal(raw, &st); uerr != nil {
			return nil, fmt.Errorf("jsonstore: %s is corrupt: %w", path, uerr)
		}
		s.state = &st
	case os.IsNotExist(err):
		s.state = defaultState()
		if werr := s.persistLocked(); werr != nil {
			return nil, werr
		}
	default:
		return nil, fmt.Errorf("jsonstore: read %s: %w", path, err)
	}

	return s, nil
}

func (s *Store) Load() (*models.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.Clone(), nil
}


func (s *Store) Update(mutate func(*models.State) error) (*models.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	draft := s.state.Clone()
	if err := mutate(draft); err != nil {
		return nil, err
	}

	previous := s.state
	s.state = draft
	if err := s.persistLocked(); err != nil {
		s.state = previous
		return nil, err
	}
	return s.state.Clone(), nil
}

func (s *Store) persistLocked() error {
	raw, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("jsonstore: encode state: %w", err)
	}

	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".state-*.json.tmp")
	if err != nil {
		return fmt.Errorf("jsonstore: create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) 

	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return fmt.Errorf("jsonstore: write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("jsonstore: fsync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("jsonstore: close temp file: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("jsonstore: atomic rename: %w", err)
	}

	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

func (s *Store) Close() error { return nil }
