package store

import (
	"sort"
	"strings"
	"sync"
	"time"

	"servicedesk/internal/models"
)

type EngineerStore interface {
	Create(e *models.Engineer) error
	Get(id string) (*models.Engineer, error)
	Search(filter EngineerFilter) []*models.Engineer
	SetAvailability(id string, available bool) (*models.Engineer, error)
}

// EngineerFilter mirrors the SOW's "Service Desk Search Feature":
// auto filtering by location, skill, billing rate, availability, project —
// with the minimum-cost engineer shown first.
type EngineerFilter struct {
	Location      string
	Skill         string
	Project       string // if set, only engineers approved for this named project
	OnlyAvailable bool
}

type memoryEngineerStore struct {
	mu        sync.RWMutex
	engineers map[string]*models.Engineer
	seq       int
}

func NewMemoryEngineerStore() EngineerStore {
	return &memoryEngineerStore{engineers: make(map[string]*models.Engineer)}
}

func (s *memoryEngineerStore) Create(e *models.Engineer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	now := time.Now().UTC()
	e.ID = "ENG-" + padLeft(s.seq, 4)
	e.CreatedAt = now
	e.UpdatedAt = now
	e.Available = true

	s.engineers[e.ID] = e
	return nil
}

func (s *memoryEngineerStore) Get(id string) (*models.Engineer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.engineers[id]
	if !ok {
		return nil, ErrNotFound
	}
	return e, nil
}

func (s *memoryEngineerStore) Search(filter EngineerFilter) []*models.Engineer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*models.Engineer, 0)
	for _, e := range s.engineers {
		if filter.OnlyAvailable && !e.Available {
			continue
		}
		if filter.Location != "" && !strings.EqualFold(e.Location, filter.Location) {
			continue
		}
		if filter.Skill != "" && !hasSkill(e.Skills, filter.Skill) {
			continue
		}
		if filter.Project != "" && len(e.ApprovedProjects) > 0 && !hasSkill(e.ApprovedProjects, filter.Project) {
			// Engineer has an approved-project list and this project isn't
			// on it -> this is a Named Project they're not cleared for.
			continue
		}
		result = append(result, e)
	}

	// Minimum-cost engineer shown first, per the SOW.
	sort.Slice(result, func(i, j int) bool {
		return effectiveRate(result[i]) < effectiveRate(result[j])
	})
	return result
}

func (s *memoryEngineerStore) SetAvailability(id string, available bool) (*models.Engineer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.engineers[id]
	if !ok {
		return nil, ErrNotFound
	}
	e.Available = available
	e.UpdatedAt = time.Now().UTC()
	return e, nil
}

func hasSkill(list []string, target string) bool {
	for _, s := range list {
		if strings.EqualFold(s, target) {
			return true
		}
	}
	return false
}

// effectiveRate picks whichever rate is set for cost comparison; prefers
// hourly if both are present since most dispatch tickets are hourly-first.
func effectiveRate(e *models.Engineer) float64 {
	if e.HourlyRate > 0 {
		return e.HourlyRate
	}
	return e.DayRate
}
