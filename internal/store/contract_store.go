package store

import (
	"sort"
	"sync"
	"time"

	"servicedesk/internal/models"
)

type ContractStore interface {
	Create(c *models.Contract) error
	Get(id string) (*models.Contract, error)
	List(clientName string) []*models.Contract
	// ExpiringWithin returns contracts whose expiry falls within the next
	// `days` days — backs renewal reminders.
	ExpiringWithin(days int) []*models.Contract
}

type memoryContractStore struct {
	mu        sync.RWMutex
	contracts map[string]*models.Contract
	seq       int
}

func NewMemoryContractStore() ContractStore {
	return &memoryContractStore{contracts: make(map[string]*models.Contract)}
}

func (s *memoryContractStore) Create(c *models.Contract) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	c.ID = "CTR-" + padLeft(s.seq, 4)
	c.CreatedAt = time.Now().UTC()
	s.contracts[c.ID] = c
	return nil
}

func (s *memoryContractStore) Get(id string) (*models.Contract, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.contracts[id]
	if !ok {
		return nil, ErrNotFound
	}
	return c, nil
}

func (s *memoryContractStore) List(clientName string) []*models.Contract {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.Contract, 0)
	for _, c := range s.contracts {
		if clientName != "" && c.ClientName != clientName {
			continue
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ExpiryDate < out[j].ExpiryDate })
	return out
}

func (s *memoryContractStore) ExpiringWithin(days int) []*models.Contract {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cutoff := time.Now().UTC().AddDate(0, 0, days).Format("2006-01-02")
	today := time.Now().UTC().Format("2006-01-02")
	out := make([]*models.Contract, 0)
	for _, c := range s.contracts {
		if c.ExpiryDate >= today && c.ExpiryDate <= cutoff {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ExpiryDate < out[j].ExpiryDate })
	return out
}
