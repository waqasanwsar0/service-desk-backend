package store

import (
	"errors"
	"sort"
	"sync"
	"time"

	"servicedesk/internal/models"
)

var ErrNotFound = errors.New("ticket not found")

// TicketStore is the interface the rest of the app talks to.
// Today it's backed by memoryStore. Swap in a Postgres-backed
// implementation later without touching any handler code.
type TicketStore interface {
	Create(t *models.Ticket) error
	Get(id string) (*models.Ticket, error)
	List(filter ListFilter) []*models.Ticket
	UpdateStatus(id string, newStatus models.TicketStatus) (*models.Ticket, error)
	Assign(id, engineerID, engineerName string) (*models.Ticket, error)
	AddImage(id, imageURL string) (*models.Ticket, error)
}

type ListFilter struct {
	Status      string
	ClientName  string
	Priority    string
	ProjectType string
}

type memoryStore struct {
	mu      sync.RWMutex
	tickets map[string]*models.Ticket
	seq     int
}

func NewMemoryStore() TicketStore {
	return &memoryStore{
		tickets: make(map[string]*models.Ticket),
	}
}

func (s *memoryStore) nextID() string {
	s.seq++
	return "TCK-" + time.Now().Format("20060102") + "-" + padLeft(s.seq, 4)
}

func padLeft(n int, width int) string {
	digits := []rune{}
	for n > 0 {
		digits = append([]rune{rune('0' + n%10)}, digits...)
		n /= 10
	}
	for len(digits) < width {
		digits = append([]rune{'0'}, digits...)
	}
	return string(digits)
}

func (s *memoryStore) Create(t *models.Ticket) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	t.ID = s.nextID()
	t.Status = models.StatusNew
	t.CreatedAt = now
	t.UpdatedAt = now

	s.tickets[t.ID] = t
	return nil
}

func (s *memoryStore) Get(id string) (*models.Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tickets[id]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (s *memoryStore) List(filter ListFilter) []*models.Ticket {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*models.Ticket, 0, len(s.tickets))
	for _, t := range s.tickets {
		if filter.Status != "" && string(t.Status) != filter.Status {
			continue
		}
		if filter.ClientName != "" && t.ClientName != filter.ClientName {
			continue
		}
		if filter.Priority != "" && string(t.Priority) != filter.Priority {
			continue
		}
		if filter.ProjectType != "" && string(t.ProjectType) != filter.ProjectType {
			continue
		}
		result = append(result, t)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result
}

func (s *memoryStore) UpdateStatus(id string, newStatus models.TicketStatus) (*models.Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tickets[id]
	if !ok {
		return nil, ErrNotFound
	}
	if !models.IsValidTransition(t.Status, newStatus) {
		return nil, errors.New("invalid status transition: " + string(t.Status) + " -> " + string(newStatus))
	}
	t.Status = newStatus
	t.UpdatedAt = time.Now().UTC()
	return t, nil
}

func (s *memoryStore) Assign(id, engineerID, engineerName string) (*models.Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tickets[id]
	if !ok {
		return nil, ErrNotFound
	}
	t.AssignedEngineerID = engineerID
	t.AssignedEngineerName = engineerName
	t.UpdatedAt = time.Now().UTC()

	// Assigning an engineer is what moves a ticket from New -> Assigned.
	if t.Status == models.StatusNew {
		t.Status = models.StatusAssigned
	}
	return t, nil
}

func (s *memoryStore) AddImage(id, imageURL string) (*models.Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tickets[id]
	if !ok {
		return nil, ErrNotFound
	}
	t.ImageURLs = append(t.ImageURLs, imageURL)
	t.UpdatedAt = time.Now().UTC()
	return t, nil
}
