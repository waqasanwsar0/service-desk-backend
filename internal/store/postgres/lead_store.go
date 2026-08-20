package postgres

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"servicedesk/internal/db"
	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type leadStore struct {
	db *sql.DB
}

func NewLeadStore(conn *sql.DB) store.LeadStore {
	return &leadStore{db: conn}
}

func (s *leadStore) Create(l *models.Lead) error {
	id, err := db.NextID(s.db, "seq_lead", "LEAD", 4)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	l.ID = id
	l.Status = models.LeadNew
	l.CreatedAt = now
	l.UpdatedAt = now

	notes, _ := json.Marshal(l.Notes)
	_, err = s.db.Exec(`INSERT INTO leads (id, company_name, contact_name, contact_email, contact_phone, linkedin_url, source, status, owner_id, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		l.ID, l.CompanyName, l.ContactName, l.ContactEmail, l.ContactPhone, l.LinkedInURL, l.Source, l.Status, l.OwnerID, notes, l.CreatedAt, l.UpdatedAt)
	return err
}

func (s *leadStore) Get(id string) (*models.Lead, error) {
	row := s.db.QueryRow(`SELECT id, company_name, contact_name, contact_email, contact_phone, linkedin_url, source, status, owner_id, notes, created_at, updated_at
		FROM leads WHERE id=$1`, id)
	l, err := scanLead(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	return l, err
}

func (s *leadStore) List(ownerID, status string) []*models.Lead {
	query := `SELECT id, company_name, contact_name, contact_email, contact_phone, linkedin_url, source, status, owner_id, notes, created_at, updated_at FROM leads WHERE 1=1`
	args := []interface{}{}
	if ownerID != "" {
		args = append(args, ownerID)
		query += " AND owner_id = $" + itoa(len(args))
	}
	if status != "" {
		args = append(args, status)
		query += " AND status = $" + itoa(len(args))
	}
	query += " ORDER BY updated_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.Lead, 0)
	for rows.Next() {
		l, err := scanLeadRows(rows)
		if err == nil {
			out = append(out, l)
		}
	}
	return out
}

func (s *leadStore) UpdateStatus(id string, status models.LeadStatus) (*models.Lead, error) {
	l, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE leads SET status=$1, updated_at=$2 WHERE id=$3`, status, now, id)
	if err != nil {
		return nil, err
	}
	l.Status = status
	l.UpdatedAt = now
	return l, nil
}

func (s *leadStore) AddNote(id, authorID, text string) (*models.Lead, error) {
	l, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	l.Notes = append(l.Notes, models.LeadNote{AuthorID: authorID, Text: text, CreatedAt: time.Now().UTC()})
	notes, _ := json.Marshal(l.Notes)
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE leads SET notes=$1, updated_at=$2 WHERE id=$3`, notes, now, id)
	if err != nil {
		return nil, err
	}
	l.UpdatedAt = now
	return l, nil
}

func scanLead(row *sql.Row) (*models.Lead, error) {
	return scanLeadGeneric(row)
}
func scanLeadRows(rows *sql.Rows) (*models.Lead, error) {
	return scanLeadGeneric(rows)
}
func scanLeadGeneric(row rowScanner) (*models.Lead, error) {
	var l models.Lead
	var notes []byte
	err := row.Scan(&l.ID, &l.CompanyName, &l.ContactName, &l.ContactEmail, &l.ContactPhone, &l.LinkedInURL, &l.Source, &l.Status, &l.OwnerID, &notes, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(notes, &l.Notes)
	return &l, nil
}
