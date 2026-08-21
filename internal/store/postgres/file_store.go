package postgres

import (
	"database/sql"
	"errors"
	"time"

	"servicedesk/internal/db"
	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type fileStore struct {
	db *sql.DB
}

func NewFileStore(conn *sql.DB) store.FileStore {
	return &fileStore{db: conn}
}

func (s *fileStore) Save(f *models.File) error {
	id, err := db.NextID(s.db, "seq_file", "FILE", 6)
	if err != nil {
		return err
	}
	f.ID = id
	f.CreatedAt = time.Now().UTC()
	_, err = s.db.Exec(`INSERT INTO files (id, filename, content_type, size, uploaded_by, data, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		f.ID, f.Filename, f.ContentType, f.Size, f.UploadedBy, f.Data, f.CreatedAt)
	return err
}

func (s *fileStore) Get(id string) (*models.File, error) {
	var f models.File
	err := s.db.QueryRow(`SELECT id, filename, content_type, size, uploaded_by, data, created_at FROM files WHERE id=$1`, id).
		Scan(&f.ID, &f.Filename, &f.ContentType, &f.Size, &f.UploadedBy, &f.Data, &f.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}
