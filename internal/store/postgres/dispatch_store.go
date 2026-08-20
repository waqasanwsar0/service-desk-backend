package postgres

import (
	"database/sql"
	"time"

	"servicedesk/internal/db"
	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type dispatchStore struct {
	db *sql.DB
}

func NewDispatchStore(conn *sql.DB) store.DispatchStore {
	return &dispatchStore{db: conn}
}

func (s *dispatchStore) Create(d *models.Dispatch) error {
	id, err := db.NextID(s.db, "seq_dispatch", "DSP", 4)
	if err != nil {
		return err
	}
	d.ID = id
	d.CreatedAt = time.Now().UTC()
	_, err = s.db.Exec(`INSERT INTO dispatches (id, ticket_id, project_id, client_name, site_address, engineer_id, scheduled_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		d.ID, d.TicketID, d.ProjectID, d.ClientName, d.SiteAddress, d.EngineerID, d.ScheduledAt, d.CreatedAt)
	return err
}

func (s *dispatchStore) List(projectID string) []*models.Dispatch {
	query := `SELECT id, ticket_id, project_id, client_name, site_address, engineer_id, scheduled_at, created_at FROM dispatches WHERE 1=1`
	args := []interface{}{}
	if projectID != "" {
		args = append(args, projectID)
		query += " AND project_id = $1"
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.Dispatch, 0)
	for rows.Next() {
		var d models.Dispatch
		if err := rows.Scan(&d.ID, &d.TicketID, &d.ProjectID, &d.ClientName, &d.SiteAddress, &d.EngineerID, &d.ScheduledAt, &d.CreatedAt); err == nil {
			out = append(out, &d)
		}
	}
	return out
}
