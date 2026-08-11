package store

import (
	"errors"
	"sort"
	"sync"
	"time"

	"servicedesk/internal/models"
)

var ErrInvalidStageMove = errors.New("invalid pipeline stage move")

type ApplicantStore interface {
	Create(a *models.Applicant) error
	Get(id string) (*models.Applicant, error)
	List(jobTitle, stage, recruiterID string) []*models.Applicant
	MoveStage(id string, newStage models.ApplicantStage) (*models.Applicant, error)
	AddNote(id, authorID, text string) (*models.Applicant, error)
}

type memoryApplicantStore struct {
	mu         sync.RWMutex
	applicants map[string]*models.Applicant
	seq        int
}

func NewMemoryApplicantStore() ApplicantStore {
	return &memoryApplicantStore{applicants: make(map[string]*models.Applicant)}
}

func (s *memoryApplicantStore) Create(a *models.Applicant) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	now := time.Now().UTC()
	a.ID = "APP-" + padLeft(s.seq, 4)
	a.Stage = models.StageApplied
	a.CreatedAt = now
	a.UpdatedAt = now

	s.applicants[a.ID] = a
	return nil
}

func (s *memoryApplicantStore) Get(id string) (*models.Applicant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	a, ok := s.applicants[id]
	if !ok {
		return nil, ErrNotFound
	}
	return a, nil
}

func (s *memoryApplicantStore) List(jobTitle, stage, recruiterID string) []*models.Applicant {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.Applicant, 0)
	for _, a := range s.applicants {
		if jobTitle != "" && a.JobTitle != jobTitle {
			continue
		}
		if stage != "" && string(a.Stage) != stage {
			continue
		}
		if recruiterID != "" && a.RecruiterID != recruiterID {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *memoryApplicantStore) MoveStage(id string, newStage models.ApplicantStage) (*models.Applicant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.applicants[id]
	if !ok {
		return nil, ErrNotFound
	}
	if !models.IsValidStageMove(a.Stage, newStage) {
		return nil, ErrInvalidStageMove
	}
	a.Stage = newStage
	a.UpdatedAt = time.Now().UTC()
	return a, nil
}

func (s *memoryApplicantStore) AddNote(id, authorID, text string) (*models.Applicant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.applicants[id]
	if !ok {
		return nil, ErrNotFound
	}
	a.Notes = append(a.Notes, models.ApplicantNote{
		AuthorID:  authorID,
		Text:      text,
		CreatedAt: time.Now().UTC(),
	})
	a.UpdatedAt = time.Now().UTC()
	return a, nil
}
