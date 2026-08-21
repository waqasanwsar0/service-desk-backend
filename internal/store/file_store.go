package store

import (
	"sync"
	"time"

	"servicedesk/internal/models"
)

type FileStore interface {
	Save(f *models.File) error
	Get(id string) (*models.File, error)
}

type memoryFileStore struct {
	mu    sync.RWMutex
	files map[string]*models.File
	seq   int
}

func NewMemoryFileStore() FileStore {
	return &memoryFileStore{files: make(map[string]*models.File)}
}

func (s *memoryFileStore) Save(f *models.File) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	f.ID = "FILE-" + padLeft(s.seq, 6)
	f.CreatedAt = time.Now().UTC()
	s.files[f.ID] = f
	return nil
}

func (s *memoryFileStore) Get(id string) (*models.File, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, ok := s.files[id]
	if !ok {
		return nil, ErrNotFound
	}
	return f, nil
}
