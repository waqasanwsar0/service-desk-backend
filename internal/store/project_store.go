package store

import (
	"sort"
	"sync"
	"time"

	"servicedesk/internal/models"
)

type ProjectStore interface {
	Create(p *models.Project) error
	Get(id string) (*models.Project, error)
	List(country, city string) []*models.Project
}

type memoryProjectStore struct {
	mu       sync.RWMutex
	projects map[string]*models.Project
	seq      int
}

func NewMemoryProjectStore() ProjectStore {
	return &memoryProjectStore{projects: make(map[string]*models.Project)}
}

func (s *memoryProjectStore) Create(p *models.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	p.ID = "PRJ-" + padLeft(s.seq, 4)
	p.CreatedAt = time.Now().UTC()
	s.projects[p.ID] = p
	return nil
}

func (s *memoryProjectStore) Get(id string) (*models.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.projects[id]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (s *memoryProjectStore) List(country, city string) []*models.Project {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.Project, 0)
	for _, p := range s.projects {
		if country != "" && p.Country != country {
			continue
		}
		if city != "" && p.City != city {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}
