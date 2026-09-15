package pgstore

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VikasKumar281/SyncStage/backend/internal/models"
)

type Store struct {
	mu   sync.Mutex
	pool *pgxpool.Pool
}

func Open(databaseURL string, defaultState func() *models.State) (*Store, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("pgstore: DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgstore: create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgstore: database ping failed: %w", err)
	}

	s := &Store{pool: pool}

	if err := s.init(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	var exists bool
	err = pool.QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM sequencer_state WHERE id = 1)`,
	).Scan(&exists)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgstore: check state: %w", err)
	}

	if !exists {
		state := defaultState()

		raw, err := json.Marshal(state)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("pgstore: encode seed state: %w", err)
		}

		_, err = pool.Exec(
			ctx,
			`INSERT INTO sequencer_state (id, state) VALUES (1, $1::jsonb)`,
			raw,
		)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("pgstore: insert seed state: %w", err)
		}
	}

	return s, nil
}

func (s *Store) init(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS sequencer_state (
			id INTEGER PRIMARY KEY,
			state JSONB NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("pgstore: create table: %w", err)
	}

	return nil
}

func (s *Store) Load() (*models.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var raw []byte

	err := s.pool.QueryRow(
		ctx,
		`SELECT state FROM sequencer_state WHERE id = 1`,
	).Scan(&raw)

	if err != nil {
		return nil, fmt.Errorf("pgstore: load state: %w", err)
	}

	var state models.State
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("pgstore: decode state: %w", err)
	}

	return state.Clone(), nil
}

func (s *Store) Update(mutate func(*models.State) error) (*models.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var raw []byte

	if err := s.pool.QueryRow(
		ctx,
		`SELECT state FROM sequencer_state WHERE id = 1`,
	).Scan(&raw); err != nil {
		return nil, fmt.Errorf("pgstore: load before update: %w", err)
	}

	var current models.State
	if err := json.Unmarshal(raw, &current); err != nil {
		return nil, fmt.Errorf("pgstore: decode current state: %w", err)
	}

	draft := current.Clone()

	if err := mutate(draft); err != nil {
		return nil, err
	}

	nextRaw, err := json.Marshal(draft)
	if err != nil {
		return nil, fmt.Errorf("pgstore: encode state: %w", err)
	}

	_, err = s.pool.Exec(
		ctx,
		`UPDATE sequencer_state
		 SET state = $1::jsonb, updated_at = NOW()
		 WHERE id = 1`,
		nextRaw,
	)
	if err != nil {
		return nil, fmt.Errorf("pgstore: save state: %w", err)
	}

	return draft.Clone(), nil
}

func (s *Store) Close() error {
	if s.pool != nil {
		s.pool.Close()
	}
	return nil
}
