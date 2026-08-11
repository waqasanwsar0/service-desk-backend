package store

import (
	"sort"
	"sync"
	"time"

	"servicedesk/internal/models"
)

type AssignmentStore interface {
	Create(a *models.EngineerAssignment) error
	ListByEngineer(engineerID string) []*models.EngineerAssignment
	End(id string) (*models.EngineerAssignment, error)
}

type memoryAssignmentStore struct {
	mu          sync.RWMutex
	assignments map[string]*models.EngineerAssignment
	seq         int
}

func NewMemoryAssignmentStore() AssignmentStore {
	return &memoryAssignmentStore{assignments: make(map[string]*models.EngineerAssignment)}
}

func (s *memoryAssignmentStore) Create(a *models.EngineerAssignment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	a.ID = "ASG-" + padLeft(s.seq, 4)
	a.Active = true
	a.CreatedAt = time.Now().UTC()
	s.assignments[a.ID] = a
	return nil
}

func (s *memoryAssignmentStore) ListByEngineer(engineerID string) []*models.EngineerAssignment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.EngineerAssignment, 0)
	for _, a := range s.assignments {
		if engineerID != "" && a.EngineerID != engineerID {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *memoryAssignmentStore) End(id string) (*models.EngineerAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.assignments[id]
	if !ok {
		return nil, ErrNotFound
	}
	a.Active = false
	a.EndDate = time.Now().UTC().Format("2006-01-02")
	return a, nil
}
