package store

import (
	"sort"
	"sync"
	"time"

	"servicedesk/internal/models"
)

type OutreachStore interface {
	Create(o *models.OutreachContact) error
	Get(id string) (*models.OutreachContact, error)
	List(recruiterID, status string) []*models.OutreachContact
	UpdateStatus(id string, status models.OutreachStatus) (*models.OutreachContact, error)
	AddNote(id, authorID, text string) (*models.OutreachContact, error)
}

type memoryOutreachStore struct {
	mu       sync.RWMutex
	contacts map[string]*models.OutreachContact
	seq      int
}

func NewMemoryOutreachStore() OutreachStore {
	return &memoryOutreachStore{contacts: make(map[string]*models.OutreachContact)}
}

func (s *memoryOutreachStore) Create(o *models.OutreachContact) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	now := time.Now().UTC()
	o.ID = "OUT-" + padLeft(s.seq, 4)
	o.Status = models.OutreachSent
	o.CreatedAt = now
	o.UpdatedAt = now

	s.contacts[o.ID] = o
	return nil
}

func (s *memoryOutreachStore) Get(id string) (*models.OutreachContact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	o, ok := s.contacts[id]
	if !ok {
		return nil, ErrNotFound
	}
	return o, nil
}

func (s *memoryOutreachStore) List(recruiterID, status string) []*models.OutreachContact {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.OutreachContact, 0)
	for _, o := range s.contacts {
		if recruiterID != "" && o.RecruiterID != recruiterID {
			continue
		}
		if status != "" && string(o.Status) != status {
			continue
		}
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MessageSentAt.After(out[j].MessageSentAt) })
	return out
}

func (s *memoryOutreachStore) UpdateStatus(id string, status models.OutreachStatus) (*models.OutreachContact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	o, ok := s.contacts[id]
	if !ok {
		return nil, ErrNotFound
	}
	o.Status = status
	o.UpdatedAt = time.Now().UTC()
	return o, nil
}

func (s *memoryOutreachStore) AddNote(id, authorID, text string) (*models.OutreachContact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	o, ok := s.contacts[id]
	if !ok {
		return nil, ErrNotFound
	}
	o.Notes = append(o.Notes, models.OutreachNote{AuthorID: authorID, Text: text, CreatedAt: time.Now().UTC()})
	o.UpdatedAt = time.Now().UTC()
	return o, nil
}
