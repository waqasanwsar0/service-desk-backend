package postgres

import (
	"database/sql"
	"errors"
	"time"

	"servicedesk/internal/db"
	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type salaryStore struct {
	db *sql.DB
}

func NewSalaryStore(conn *sql.DB) store.SalaryStore {
	return &salaryStore{db: conn}
}

func (s *salaryStore) Create(rec *models.SalaryRecord) error {
	id, err := db.NextID(s.db, "seq_salary", "SAL", 4)
	if err != nil {
		return err
	}
	rec.ID = id
	rec.Status = models.SalaryPending
	rec.CreatedAt = time.Now().UTC()
	_, err = s.db.Exec(`INSERT INTO salaries (id, employee_name, user_id, pay_period, amount, currency, status, notes, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		rec.ID, rec.EmployeeName, rec.UserID, rec.PayPeriod, rec.Amount, rec.Currency, rec.Status, rec.Notes, rec.CreatedAt)
	return err
}

func (s *salaryStore) List(payPeriod string) []*models.SalaryRecord {
	query := `SELECT id, employee_name, user_id, pay_period, amount, currency, status, paid_at, notes, created_at FROM salaries WHERE 1=1`
	args := []interface{}{}
	if payPeriod != "" {
		args = append(args, payPeriod)
		query += " AND pay_period = $1"
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.SalaryRecord, 0)
	for rows.Next() {
		var r models.SalaryRecord
		var paidAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.EmployeeName, &r.UserID, &r.PayPeriod, &r.Amount, &r.Currency, &r.Status, &paidAt, &r.Notes, &r.CreatedAt); err == nil {
			if paidAt.Valid {
				r.PaidAt = &paidAt.Time
			}
			out = append(out, &r)
		}
	}
	return out
}

func (s *salaryStore) MarkPaid(id string) (*models.SalaryRecord, error) {
	now := time.Now().UTC()
	res, err := s.db.Exec(`UPDATE salaries SET status=$1, paid_at=$2 WHERE id=$3`, models.SalaryPaid, now, id)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, store.ErrNotFound
	}
	var r models.SalaryRecord
	var paidAt sql.NullTime
	err = s.db.QueryRow(`SELECT id, employee_name, user_id, pay_period, amount, currency, status, paid_at, notes, created_at FROM salaries WHERE id=$1`, id).
		Scan(&r.ID, &r.EmployeeName, &r.UserID, &r.PayPeriod, &r.Amount, &r.Currency, &r.Status, &paidAt, &r.Notes, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if paidAt.Valid {
		r.PaidAt = &paidAt.Time
	}
	return &r, err
}
