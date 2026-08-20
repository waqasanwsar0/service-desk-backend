package postgres

import (
	"database/sql"
	"errors"
	"time"

	"servicedesk/internal/db"
	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type projectStore struct {
	db *sql.DB
}

func NewProjectStore(conn *sql.DB) store.ProjectStore {
	return &projectStore{db: conn}
}

func (s *projectStore) Create(p *models.Project) error {
	id, err := db.NextID(s.db, "seq_project", "PRJ", 4)
	if err != nil {
		return err
	}
	p.ID = id
	p.CreatedAt = time.Now().UTC()
	_, err = s.db.Exec(`INSERT INTO projects (id, name, client_name, country, city, type, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		p.ID, p.Name, p.ClientName, p.Country, p.City, p.Type, p.CreatedAt)
	return err
}

func (s *projectStore) Get(id string) (*models.Project, error) {
	var p models.Project
	err := s.db.QueryRow(`SELECT id, name, client_name, country, city, type, created_at FROM projects WHERE id=$1`, id).
		Scan(&p.ID, &p.Name, &p.ClientName, &p.Country, &p.City, &p.Type, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *projectStore) List(country, city string) []*models.Project {
	query := `SELECT id, name, client_name, country, city, type, created_at FROM projects WHERE 1=1`
	args := []interface{}{}
	if country != "" {
		args = append(args, country)
		query += " AND country = $" + itoa(len(args))
	}
	if city != "" {
		args = append(args, city)
		query += " AND city = $" + itoa(len(args))
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.Project, 0)
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.ClientName, &p.Country, &p.City, &p.Type, &p.CreatedAt); err == nil {
			out = append(out, &p)
		}
	}
	return out
}
