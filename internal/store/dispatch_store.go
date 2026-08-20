package store

import (
	"sort"
	"sync"
	"time"

	"servicedesk/internal/models"
)

type DispatchStore interface {
	Create(d *models.Dispatch) error
	List(projectID string) []*models.Dispatch
}

type memoryDispatchStore struct {
	mu         sync.RWMutex
	dispatches map[string]*models.Dispatch
	seq        int
}

func NewMemoryDispatchStore() DispatchStore {
	return &memoryDispatchStore{dispatches: make(map[string]*models.Dispatch)}
}

func (s *memoryDispatchStore) Create(d *models.Dispatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	d.ID = "DSP-" + padLeft(s.seq, 4)
	d.CreatedAt = time.Now().UTC()
	s.dispatches[d.ID] = d
	return nil
}

func (s *memoryDispatchStore) List(projectID string) []*models.Dispatch {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.Dispatch, 0)
	for _, d := range s.dispatches {
		if projectID != "" && d.ProjectID != projectID {
			continue
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}
