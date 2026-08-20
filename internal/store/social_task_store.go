package store

import (
	"sort"
	"sync"
	"time"

	"servicedesk/internal/models"
)

type SocialTaskStore interface {
	Create(t *models.SocialMediaTask) error
	Get(id string) (*models.SocialMediaTask, error)
	List(platform, status, ownerID string) []*models.SocialMediaTask
	UpdateStatus(id string, status models.SocialTaskStatus) (*models.SocialMediaTask, error)
	AddNote(id, authorID, text string) (*models.SocialMediaTask, error)
}

type memorySocialTaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*models.SocialMediaTask
	seq   int
}

func NewMemorySocialTaskStore() SocialTaskStore {
	return &memorySocialTaskStore{tasks: make(map[string]*models.SocialMediaTask)}
}

func (s *memorySocialTaskStore) Create(t *models.SocialMediaTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	now := time.Now().UTC()
	t.ID = "SMT-" + padLeft(s.seq, 4)
	t.Status = models.SocialTaskDraft
	t.CreatedAt = now
	t.UpdatedAt = now

	s.tasks[t.ID] = t
	return nil
}

func (s *memorySocialTaskStore) Get(id string) (*models.SocialMediaTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[id]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (s *memorySocialTaskStore) List(platform, status, ownerID string) []*models.SocialMediaTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.SocialMediaTask, 0)
	for _, t := range s.tasks {
		if platform != "" && t.Platform != platform {
			continue
		}
		if status != "" && string(t.Status) != status {
			continue
		}
		if ownerID != "" && t.OwnerID != ownerID {
			continue
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *memorySocialTaskStore) UpdateStatus(id string, status models.SocialTaskStatus) (*models.SocialMediaTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return nil, ErrNotFound
	}
	t.Status = status
	t.UpdatedAt = time.Now().UTC()
	return t, nil
}

func (s *memorySocialTaskStore) AddNote(id, authorID, text string) (*models.SocialMediaTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return nil, ErrNotFound
	}
	t.Notes = append(t.Notes, models.SocialTaskNote{AuthorID: authorID, Text: text, CreatedAt: time.Now().UTC()})
	t.UpdatedAt = time.Now().UTC()
	return t, nil
}
