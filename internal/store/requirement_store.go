package store

import (
	"sort"
	"sync"
	"time"

	"servicedesk/internal/models"
)

type RequirementStore interface {
	Create(r *models.Requirement) error
	Get(id string) (*models.Requirement, error)
	List(projectID, clientName string) []*models.Requirement
}

type memoryRequirementStore struct {
	mu           sync.RWMutex
	requirements map[string]*models.Requirement
	seq          int
}

func NewMemoryRequirementStore() RequirementStore {
	return &memoryRequirementStore{requirements: make(map[string]*models.Requirement)}
}

func (s *memoryRequirementStore) Create(r *models.Requirement) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	r.ID = "REQ-" + padLeft(s.seq, 4)
	r.CreatedAt = time.Now().UTC()
	s.requirements[r.ID] = r
	return nil
}

func (s *memoryRequirementStore) Get(id string) (*models.Requirement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.requirements[id]
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

func (s *memoryRequirementStore) List(projectID, clientName string) []*models.Requirement {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.Requirement, 0)
	for _, r := range s.requirements {
		if projectID != "" && r.ProjectID != projectID {
			continue
		}
		if clientName != "" && r.ClientName != clientName {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}
