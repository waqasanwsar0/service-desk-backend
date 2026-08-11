package postgres

import (
	"database/sql"
	"errors"
	"time"

	"servicedesk/internal/db"
	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type userStore struct {
	db *sql.DB
}

func NewUserStore(conn *sql.DB) store.UserStore {
	return &userStore{db: conn}
}

func (s *userStore) Create(u *models.User) error {
	var exists bool
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE lower(email) = lower($1))`, u.Email).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return store.ErrUserExists
	}
	id, err := db.NextID(s.db, "seq_user", "USR", 4)
	if err != nil {
		return err
	}
	u.ID = id
	u.CreatedAt = time.Now().UTC()
	_, err = s.db.Exec(`INSERT INTO users (id, name, email, role, password_hash, created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		u.ID, u.Name, u.Email, u.Role, u.PasswordHash, u.CreatedAt)
	return err
}

func (s *userStore) GetByEmail(email string) (*models.User, error) {
	row := s.db.QueryRow(`SELECT id, name, email, role, password_hash, created_at FROM users WHERE lower(email) = lower($1)`, email)
	return scanUser(row)
}

func (s *userStore) GetByID(id string) (*models.User, error) {
	row := s.db.QueryRow(`SELECT id, name, email, role, password_hash, created_at FROM users WHERE id = $1`, id)
	return scanUser(row)
}

func (s *userStore) List() []*models.User {
	rows, err := s.db.Query(`SELECT id, name, email, role, password_hash, created_at FROM users`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.PasswordHash, &u.CreatedAt); err == nil {
			out = append(out, &u)
		}
	}
	return out
}

func scanUser(row *sql.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
