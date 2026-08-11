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

type applicantStore struct {
	db *sql.DB
}

func NewApplicantStore(conn *sql.DB) store.ApplicantStore {
	return &applicantStore{db: conn}
}

func (s *applicantStore) Create(a *models.Applicant) error {
	id, err := db.NextID(s.db, "seq_applicant", "APP", 4)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	a.ID = id
	a.Stage = models.StageApplied
	a.CreatedAt = now
	a.UpdatedAt = now

	notes, _ := json.Marshal(a.Notes)
	_, err = s.db.Exec(`INSERT INTO applicants (id, name, email, phone, job_title, resume_url, stage, recruiter_id, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		a.ID, a.Name, a.Email, a.Phone, a.JobTitle, a.ResumeURL, a.Stage, a.RecruiterID, notes, a.CreatedAt, a.UpdatedAt)
	return err
}

func (s *applicantStore) Get(id string) (*models.Applicant, error) {
	row := s.db.QueryRow(`SELECT id, name, email, phone, job_title, resume_url, stage, recruiter_id, notes, created_at, updated_at
		FROM applicants WHERE id=$1`, id)
	a, err := scanApplicant(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	return a, err
}

func (s *applicantStore) List(jobTitle, stage, recruiterID string) []*models.Applicant {
	query := `SELECT id, name, email, phone, job_title, resume_url, stage, recruiter_id, notes, created_at, updated_at FROM applicants WHERE 1=1`
	args := []interface{}{}
	if jobTitle != "" {
		args = append(args, jobTitle)
		query += " AND job_title = $" + itoa(len(args))
	}
	if stage != "" {
		args = append(args, stage)
		query += " AND stage = $" + itoa(len(args))
	}
	if recruiterID != "" {
		args = append(args, recruiterID)
		query += " AND recruiter_id = $" + itoa(len(args))
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.Applicant, 0)
	for rows.Next() {
		a, err := scanApplicantRows(rows)
		if err == nil {
			out = append(out, a)
		}
	}
	return out
}

func (s *applicantStore) MoveStage(id string, newStage models.ApplicantStage) (*models.Applicant, error) {
	a, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if !models.IsValidStageMove(a.Stage, newStage) {
		return nil, store.ErrInvalidStageMove
	}
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE applicants SET stage=$1, updated_at=$2 WHERE id=$3`, newStage, now, id)
	if err != nil {
		return nil, err
	}
	a.Stage = newStage
	a.UpdatedAt = now
	return a, nil
}

func (s *applicantStore) AddNote(id, authorID, text string) (*models.Applicant, error) {
	a, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	a.Notes = append(a.Notes, models.ApplicantNote{AuthorID: authorID, Text: text, CreatedAt: time.Now().UTC()})
	notes, _ := json.Marshal(a.Notes)
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE applicants SET notes=$1, updated_at=$2 WHERE id=$3`, notes, now, id)
	if err != nil {
		return nil, err
	}
	a.UpdatedAt = now
	return a, nil
}

func scanApplicant(row *sql.Row) (*models.Applicant, error) {
	return scanApplicantGeneric(row)
}
func scanApplicantRows(rows *sql.Rows) (*models.Applicant, error) {
	return scanApplicantGeneric(rows)
}
func scanApplicantGeneric(row rowScanner) (*models.Applicant, error) {
	var a models.Applicant
	var notes []byte
	err := row.Scan(&a.ID, &a.Name, &a.Email, &a.Phone, &a.JobTitle, &a.ResumeURL, &a.Stage, &a.RecruiterID, &notes, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(notes, &a.Notes)
	return &a, nil
}
