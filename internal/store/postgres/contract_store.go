package postgres

import (
	"database/sql"
	"errors"
	"time"

	"servicedesk/internal/db"
	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type contractStore struct {
	db *sql.DB
}

func NewContractStore(conn *sql.DB) store.ContractStore {
	return &contractStore{db: conn}
}

func (s *contractStore) Create(c *models.Contract) error {
	id, err := db.NextID(s.db, "seq_contract", "CTR", 4)
	if err != nil {
		return err
	}
	c.ID = id
	c.CreatedAt = time.Now().UTC()
	_, err = s.db.Exec(`INSERT INTO contracts (id, client_name, entity_id, project_scope, billing_terms, start_date, expiry_date, auto_renew, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		c.ID, c.ClientName, c.EntityID, c.ProjectScope, c.BillingTerms, c.StartDate, c.ExpiryDate, c.AutoRenew, c.CreatedAt)
	return err
}

func (s *contractStore) Get(id string) (*models.Contract, error) {
	var c models.Contract
	err := s.db.QueryRow(`SELECT id, client_name, entity_id, project_scope, billing_terms, start_date, expiry_date, auto_renew, created_at
		FROM contracts WHERE id=$1`, id).
		Scan(&c.ID, &c.ClientName, &c.EntityID, &c.ProjectScope, &c.BillingTerms, &c.StartDate, &c.ExpiryDate, &c.AutoRenew, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *contractStore) List(clientName string) []*models.Contract {
	query := `SELECT id, client_name, entity_id, project_scope, billing_terms, start_date, expiry_date, auto_renew, created_at FROM contracts WHERE 1=1`
	args := []interface{}{}
	if clientName != "" {
		args = append(args, clientName)
		query += " AND client_name = $1"
	}
	query += " ORDER BY expiry_date ASC"
	return s.queryContracts(query, args...)
}

func (s *contractStore) ExpiringWithin(days int) []*models.Contract {
	cutoff := time.Now().UTC().AddDate(0, 0, days).Format("2006-01-02")
	today := time.Now().UTC().Format("2006-01-02")
	query := `SELECT id, client_name, entity_id, project_scope, billing_terms, start_date, expiry_date, auto_renew, created_at
		FROM contracts WHERE expiry_date >= $1 AND expiry_date <= $2 ORDER BY expiry_date ASC`
	return s.queryContracts(query, today, cutoff)
}

func (s *contractStore) queryContracts(query string, args ...interface{}) []*models.Contract {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.Contract, 0)
	for rows.Next() {
		var c models.Contract
		if err := rows.Scan(&c.ID, &c.ClientName, &c.EntityID, &c.ProjectScope, &c.BillingTerms, &c.StartDate, &c.ExpiryDate, &c.AutoRenew, &c.CreatedAt); err == nil {
			out = append(out, &c)
		}
	}
	return out
}
