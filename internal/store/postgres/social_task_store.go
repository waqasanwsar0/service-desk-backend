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

type socialTaskStore struct {
	db *sql.DB
}

func NewSocialTaskStore(conn *sql.DB) store.SocialTaskStore {
	return &socialTaskStore{db: conn}
}

func (s *socialTaskStore) Create(t *models.SocialMediaTask) error {
	id, err := db.NextID(s.db, "seq_social_task", "SMT", 4)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	t.ID = id
	t.Status = models.SocialTaskDraft
	t.CreatedAt = now
	t.UpdatedAt = now

	notes, _ := json.Marshal(t.Notes)
	_, err = s.db.Exec(`INSERT INTO social_media_tasks (id, title, platform, description, owner_id, due_date, status, asset_url, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		t.ID, t.Title, t.Platform, t.Description, t.OwnerID, t.DueDate, t.Status, t.AssetURL, notes, t.CreatedAt, t.UpdatedAt)
	return err
}

func (s *socialTaskStore) Get(id string) (*models.SocialMediaTask, error) {
	row := s.db.QueryRow(`SELECT id, title, platform, description, owner_id, due_date, status, asset_url, notes, created_at, updated_at
		FROM social_media_tasks WHERE id=$1`, id)
	t, err := scanSocialTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	return t, err
}

func (s *socialTaskStore) List(platform, status, ownerID string) []*models.SocialMediaTask {
	query := `SELECT id, title, platform, description, owner_id, due_date, status, asset_url, notes, created_at, updated_at FROM social_media_tasks WHERE 1=1`
	args := []interface{}{}
	if platform != "" {
		args = append(args, platform)
		query += " AND platform = $" + itoa(len(args))
	}
	if status != "" {
		args = append(args, status)
		query += " AND status = $" + itoa(len(args))
	}
	if ownerID != "" {
		args = append(args, ownerID)
		query += " AND owner_id = $" + itoa(len(args))
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.SocialMediaTask, 0)
	for rows.Next() {
		t, err := scanSocialTaskRows(rows)
		if err == nil {
			out = append(out, t)
		}
	}
	return out
}

func (s *socialTaskStore) UpdateStatus(id string, status models.SocialTaskStatus) (*models.SocialMediaTask, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE social_media_tasks SET status=$1, updated_at=$2 WHERE id=$3`, status, now, id)
	if err != nil {
		return nil, err
	}
	t.Status = status
	t.UpdatedAt = now
	return t, nil
}

func (s *socialTaskStore) AddNote(id, authorID, text string) (*models.SocialMediaTask, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	t.Notes = append(t.Notes, models.SocialTaskNote{AuthorID: authorID, Text: text, CreatedAt: time.Now().UTC()})
	notes, _ := json.Marshal(t.Notes)
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE social_media_tasks SET notes=$1, updated_at=$2 WHERE id=$3`, notes, now, id)
	if err != nil {
		return nil, err
	}
	t.UpdatedAt = now
	return t, nil
}

func scanSocialTask(row *sql.Row) (*models.SocialMediaTask, error) {
	return scanSocialTaskGeneric(row)
}
func scanSocialTaskRows(rows *sql.Rows) (*models.SocialMediaTask, error) {
	return scanSocialTaskGeneric(rows)
}
func scanSocialTaskGeneric(row rowScanner) (*models.SocialMediaTask, error) {
	var t models.SocialMediaTask
	var notes []byte
	err := row.Scan(&t.ID, &t.Title, &t.Platform, &t.Description, &t.OwnerID, &t.DueDate, &t.Status, &t.AssetURL, &notes, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(notes, &t.Notes)
	return &t, nil
}
