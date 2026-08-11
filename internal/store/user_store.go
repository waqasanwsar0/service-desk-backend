package store

import (
	"errors"
	"strings"
	"sync"
	"time"

	"servicedesk/internal/models"
)

var ErrUserExists = errors.New("user with this email already exists")
var ErrUserNotFound = errors.New("user not found")

type UserStore interface {
	Create(u *models.User) error
	GetByEmail(email string) (*models.User, error)
	GetByID(id string) (*models.User, error)
	List() []*models.User
}

type memoryUserStore struct {
	mu    sync.RWMutex
	byID  map[string]*models.User
	byEml map[string]string // email(lower) -> id
	seq   int
}

func NewMemoryUserStore() UserStore {
	return &memoryUserStore{
		byID:  make(map[string]*models.User),
		byEml: make(map[string]string),
	}
}

func (s *memoryUserStore) Create(u *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := strings.ToLower(u.Email)
	if _, exists := s.byEml[key]; exists {
		return ErrUserExists
	}

	s.seq++
	u.ID = "USR-" + padLeft(s.seq, 4)
	u.CreatedAt = time.Now().UTC()

	s.byID[u.ID] = u
	s.byEml[key] = u.ID
	return nil
}

func (s *memoryUserStore) GetByEmail(email string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.byEml[strings.ToLower(email)]
	if !ok {
		return nil, ErrUserNotFound
	}
	return s.byID[id], nil
}

func (s *memoryUserStore) GetByID(id string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.byID[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (s *memoryUserStore) List() []*models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.User, 0, len(s.byID))
	for _, u := range s.byID {
		out = append(out, u)
	}
	return out
}
