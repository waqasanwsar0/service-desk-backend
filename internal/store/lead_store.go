package store

import (
	"sort"
	"sync"
	"time"

	"servicedesk/internal/models"
)

type LeadStore interface {
	Create(l *models.Lead) error
	Get(id string) (*models.Lead, error)
	List(ownerID, status string) []*models.Lead
	UpdateStatus(id string, status models.LeadStatus) (*models.Lead, error)
	AddNote(id, authorID, text string) (*models.Lead, error)
}

type memoryLeadStore struct {
	mu    sync.RWMutex
	leads map[string]*models.Lead
	seq   int
}

func NewMemoryLeadStore() LeadStore {
	return &memoryLeadStore{leads: make(map[string]*models.Lead)}
}

func (s *memoryLeadStore) Create(l *models.Lead) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	now := time.Now().UTC()
	l.ID = "LEAD-" + padLeft(s.seq, 4)
	l.Status = models.LeadNew
	l.CreatedAt = now
	l.UpdatedAt = now

	s.leads[l.ID] = l
	return nil
}

func (s *memoryLeadStore) Get(id string) (*models.Lead, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	l, ok := s.leads[id]
	if !ok {
		return nil, ErrNotFound
	}
	return l, nil
}

func (s *memoryLeadStore) List(ownerID, status string) []*models.Lead {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.Lead, 0)
	for _, l := range s.leads {
		if ownerID != "" && l.OwnerID != ownerID {
			continue
		}
		if status != "" && string(l.Status) != status {
			continue
		}
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out
}

func (s *memoryLeadStore) UpdateStatus(id string, status models.LeadStatus) (*models.Lead, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	l, ok := s.leads[id]
	if !ok {
		return nil, ErrNotFound
	}
	l.Status = status
	l.UpdatedAt = time.Now().UTC()
	return l, nil
}

func (s *memoryLeadStore) AddNote(id, authorID, text string) (*models.Lead, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	l, ok := s.leads[id]
	if !ok {
		return nil, ErrNotFound
	}
	l.Notes = append(l.Notes, models.LeadNote{AuthorID: authorID, Text: text, CreatedAt: time.Now().UTC()})
	l.UpdatedAt = time.Now().UTC()
	return l, nil
}
