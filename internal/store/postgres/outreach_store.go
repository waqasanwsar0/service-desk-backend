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

type outreachStore struct {
	db *sql.DB
}

func NewOutreachStore(conn *sql.DB) store.OutreachStore {
	return &outreachStore{db: conn}
}

func (s *outreachStore) Create(o *models.OutreachContact) error {
	id, err := db.NextID(s.db, "seq_outreach", "OUT", 4)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	o.ID = id
	o.Status = models.OutreachSent
	o.CreatedAt = now
	o.UpdatedAt = now

	notes, _ := json.Marshal(o.Notes)
	_, err = s.db.Exec(`INSERT INTO outreach_contacts (id, name, linkedin_url, target_role, recruiter_id, status, message_sent_at, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		o.ID, o.Name, o.LinkedInURL, o.TargetRole, o.RecruiterID, o.Status, o.MessageSentAt, notes, o.CreatedAt, o.UpdatedAt)
	return err
}

func (s *outreachStore) Get(id string) (*models.OutreachContact, error) {
	row := s.db.QueryRow(`SELECT id, name, linkedin_url, target_role, recruiter_id, status, message_sent_at, notes, created_at, updated_at
		FROM outreach_contacts WHERE id=$1`, id)
	o, err := scanOutreach(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	return o, err
}

func (s *outreachStore) List(recruiterID, status string) []*models.OutreachContact {
	query := `SELECT id, name, linkedin_url, target_role, recruiter_id, status, message_sent_at, notes, created_at, updated_at FROM outreach_contacts WHERE 1=1`
	args := []interface{}{}
	if recruiterID != "" {
		args = append(args, recruiterID)
		query += " AND recruiter_id = $" + itoa(len(args))
	}
	if status != "" {
		args = append(args, status)
		query += " AND status = $" + itoa(len(args))
	}
	query += " ORDER BY message_sent_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.OutreachContact, 0)
	for rows.Next() {
		o, err := scanOutreachRows(rows)
		if err == nil {
			out = append(out, o)
		}
	}
	return out
}

func (s *outreachStore) UpdateStatus(id string, status models.OutreachStatus) (*models.OutreachContact, error) {
	o, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE outreach_contacts SET status=$1, updated_at=$2 WHERE id=$3`, status, now, id)
	if err != nil {
		return nil, err
	}
	o.Status = status
	o.UpdatedAt = now
	return o, nil
}

func (s *outreachStore) AddNote(id, authorID, text string) (*models.OutreachContact, error) {
	o, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	o.Notes = append(o.Notes, models.OutreachNote{AuthorID: authorID, Text: text, CreatedAt: time.Now().UTC()})
	notes, _ := json.Marshal(o.Notes)
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE outreach_contacts SET notes=$1, updated_at=$2 WHERE id=$3`, notes, now, id)
	if err != nil {
		return nil, err
	}
	o.UpdatedAt = now
	return o, nil
}

func scanOutreach(row *sql.Row) (*models.OutreachContact, error) {
	return scanOutreachGeneric(row)
}
func scanOutreachRows(rows *sql.Rows) (*models.OutreachContact, error) {
	return scanOutreachGeneric(rows)
}
func scanOutreachGeneric(row rowScanner) (*models.OutreachContact, error) {
	var o models.OutreachContact
	var notes []byte
	err := row.Scan(&o.ID, &o.Name, &o.LinkedInURL, &o.TargetRole, &o.RecruiterID, &o.Status, &o.MessageSentAt, &notes, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(notes, &o.Notes)
	return &o, nil
}
