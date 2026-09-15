package storage

import (
	"errors"

	"github.com/VikasKumar281/SyncStage/backend/internal/models"
)

var ErrNotFound = errors.New("not found")

type Store interface {
	Load() (*models.State, error)

	Update(mutate func(*models.State) error) (*models.State, error)

	Close() error
}
