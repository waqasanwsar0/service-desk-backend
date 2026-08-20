package store

import (
	"sort"
	"sync"
	"time"

	"servicedesk/internal/models"
)

type SalaryStore interface {
	Create(s *models.SalaryRecord) error
	List(payPeriod string) []*models.SalaryRecord
	MarkPaid(id string) (*models.SalaryRecord, error)
}

type memorySalaryStore struct {
	mu       sync.RWMutex
	salaries map[string]*models.SalaryRecord
	seq      int
}

func NewMemorySalaryStore() SalaryStore {
	return &memorySalaryStore{salaries: make(map[string]*models.SalaryRecord)}
}

func (s *memorySalaryStore) Create(rec *models.SalaryRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	rec.ID = "SAL-" + padLeft(s.seq, 4)
	rec.Status = models.SalaryPending
	rec.CreatedAt = time.Now().UTC()
	s.salaries[rec.ID] = rec
	return nil
}

func (s *memorySalaryStore) List(payPeriod string) []*models.SalaryRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.SalaryRecord, 0)
	for _, r := range s.salaries {
		if payPeriod != "" && r.PayPeriod != payPeriod {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *memorySalaryStore) MarkPaid(id string) (*models.SalaryRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.salaries[id]
	if !ok {
		return nil, ErrNotFound
	}
	now := time.Now().UTC()
	r.Status = models.SalaryPaid
	r.PaidAt = &now
	return r, nil
}
