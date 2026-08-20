package postgres

import (
	"database/sql"
	"errors"
	"time"

	"servicedesk/internal/db"
	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type requirementStore struct {
	db *sql.DB
}

func NewRequirementStore(conn *sql.DB) store.RequirementStore {
	return &requirementStore{db: conn}
}

func (s *requirementStore) Create(r *models.Requirement) error {
	id, err := db.NextID(s.db, "seq_requirement", "REQ", 4)
	if err != nil {
		return err
	}
	r.ID = id
	r.CreatedAt = time.Now().UTC()
	_, err = s.db.Exec(`INSERT INTO requirements (id, project_id, client_name, title, description, file_url, source, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		r.ID, r.ProjectID, r.ClientName, r.Title, r.Description, r.FileURL, r.Source, r.CreatedAt)
	return err
}

func (s *requirementStore) Get(id string) (*models.Requirement, error) {
	var r models.Requirement
	err := s.db.QueryRow(`SELECT id, project_id, client_name, title, description, file_url, source, created_at FROM requirements WHERE id=$1`, id).
		Scan(&r.ID, &r.ProjectID, &r.ClientName, &r.Title, &r.Description, &r.FileURL, &r.Source, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *requirementStore) List(projectID, clientName string) []*models.Requirement {
	query := `SELECT id, project_id, client_name, title, description, file_url, source, created_at FROM requirements WHERE 1=1`
	args := []interface{}{}
	if projectID != "" {
		args = append(args, projectID)
		query += " AND project_id = $" + itoa(len(args))
	}
	if clientName != "" {
		args = append(args, clientName)
		query += " AND client_name = $" + itoa(len(args))
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.Requirement, 0)
	for rows.Next() {
		var r models.Requirement
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.ClientName, &r.Title, &r.Description, &r.FileURL, &r.Source, &r.CreatedAt); err == nil {
			out = append(out, &r)
		}
	}
	return out
}
