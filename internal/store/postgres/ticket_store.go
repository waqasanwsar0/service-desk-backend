package postgres

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"servicedesk/internal/db"
	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type ticketStore struct {
	db *sql.DB
}

func NewTicketStore(conn *sql.DB) store.TicketStore {
	return &ticketStore{db: conn}
}

func (s *ticketStore) Create(t *models.Ticket) error {
	id, err := db.NextID(s.db, "seq_ticket", "TCK", 4)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	t.ID = id
	t.Status = models.StatusNew
	t.CreatedAt = now
	t.UpdatedAt = now

	images, _ := json.Marshal(t.ImageURLs)

	_, err = s.db.Exec(`
		INSERT INTO tickets (id, title, description, client_name, project_id, project_name, project_type,
			country, domain, site_address, priority, sla_due_at, source, status,
			assigned_engineer_id, assigned_engineer_name, image_urls, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		t.ID, t.Title, t.Description, t.ClientName, t.ProjectID, t.ProjectName, t.ProjectType,
		t.Country, t.Domain, t.SiteAddress, t.Priority, t.SLADueAt, t.Source, t.Status,
		t.AssignedEngineerID, t.AssignedEngineerName, images, t.CreatedAt, t.UpdatedAt)
	return err
}

func (s *ticketStore) Get(id string) (*models.Ticket, error) {
	row := s.db.QueryRow(`
		SELECT id, title, description, client_name, project_id, project_name, project_type,
			country, domain, site_address, priority, sla_due_at, source, status,
			assigned_engineer_id, assigned_engineer_name, image_urls, created_at, updated_at
		FROM tickets WHERE id = $1`, id)
	t, err := scanTicket(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	return t, err
}

func (s *ticketStore) List(filter store.ListFilter) []*models.Ticket {
	query := `SELECT id, title, description, client_name, project_id, project_name, project_type,
		country, domain, site_address, priority, sla_due_at, source, status,
		assigned_engineer_id, assigned_engineer_name, image_urls, created_at, updated_at FROM tickets WHERE 1=1`
	args := []interface{}{}
	if filter.Status != "" {
		args = append(args, filter.Status)
		query += " AND status = $" + itoa(len(args))
	}
	if filter.ClientName != "" {
		args = append(args, filter.ClientName)
		query += " AND client_name = $" + itoa(len(args))
	}
	if filter.Priority != "" {
		args = append(args, filter.Priority)
		query += " AND priority = $" + itoa(len(args))
	}
	if filter.ProjectType != "" {
		args = append(args, filter.ProjectType)
		query += " AND project_type = $" + itoa(len(args))
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	out := make([]*models.Ticket, 0)
	for rows.Next() {
		t, err := scanTicketRows(rows)
		if err != nil {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (s *ticketStore) UpdateStatus(id string, newStatus models.TicketStatus) (*models.Ticket, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if !models.IsValidTransition(t.Status, newStatus) {
		return nil, errors.New("invalid status transition: " + string(t.Status) + " -> " + string(newStatus))
	}
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE tickets SET status = $1, updated_at = $2 WHERE id = $3`, newStatus, now, id)
	if err != nil {
		return nil, err
	}
	t.Status = newStatus
	t.UpdatedAt = now
	return t, nil
}

func (s *ticketStore) Assign(id, engineerID, engineerName string) (*models.Ticket, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	newStatus := t.Status
	if t.Status == models.StatusNew {
		newStatus = models.StatusAssigned
	}
	_, err = s.db.Exec(`UPDATE tickets SET assigned_engineer_id=$1, assigned_engineer_name=$2, status=$3, updated_at=$4 WHERE id=$5`,
		engineerID, engineerName, newStatus, now, id)
	if err != nil {
		return nil, err
	}
	t.AssignedEngineerID = engineerID
	t.AssignedEngineerName = engineerName
	t.Status = newStatus
	t.UpdatedAt = now
	return t, nil
}

func (s *ticketStore) AddImage(id, imageURL string) (*models.Ticket, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	t.ImageURLs = append(t.ImageURLs, imageURL)
	images, _ := json.Marshal(t.ImageURLs)
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE tickets SET image_urls=$1, updated_at=$2 WHERE id=$3`, images, now, id)
	if err != nil {
		return nil, err
	}
	t.UpdatedAt = now
	return t, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanTicket(row *sql.Row) (*models.Ticket, error) {
	return scanTicketGeneric(row)
}

func scanTicketRows(rows *sql.Rows) (*models.Ticket, error) {
	return scanTicketGeneric(rows)
}

func scanTicketGeneric(row rowScanner) (*models.Ticket, error) {
	var t models.Ticket
	var slaDueAt sql.NullTime
	var images []byte
	err := row.Scan(&t.ID, &t.Title, &t.Description, &t.ClientName, &t.ProjectID, &t.ProjectName, &t.ProjectType,
		&t.Country, &t.Domain, &t.SiteAddress, &t.Priority, &slaDueAt, &t.Source, &t.Status,
		&t.AssignedEngineerID, &t.AssignedEngineerName, &images, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if slaDueAt.Valid {
		t.SLADueAt = &slaDueAt.Time
	}
	_ = json.Unmarshal(images, &t.ImageURLs)
	return &t, nil
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
