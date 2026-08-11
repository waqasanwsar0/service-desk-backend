package postgres

import (
	"database/sql"
	"errors"
	"time"

	"servicedesk/internal/db"
	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type timesheetStore struct {
	db *sql.DB
}

func NewTimesheetStore(conn *sql.DB) store.TimesheetStore {
	return &timesheetStore{db: conn}
}

func (s *timesheetStore) Create(t *models.Timesheet) error {
	id, err := db.NextID(s.db, "seq_timesheet", "TSH", 4)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	t.ID = id
	t.Status = models.TimesheetPendingReview
	t.CreatedAt = now
	t.UpdatedAt = now

	_, err = s.db.Exec(`INSERT INTO timesheets (id, ticket_id, engineer_id, check_in_at, check_out_at, job_type,
		file_url, status, billed_amount, currency, invoice_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		t.ID, t.TicketID, t.EngineerID, t.CheckInAt, t.CheckOutAt, t.JobType,
		t.FileURL, t.Status, t.BilledAmount, t.Currency, t.InvoiceID, t.CreatedAt, t.UpdatedAt)
	return err
}

func (s *timesheetStore) Get(id string) (*models.Timesheet, error) {
	row := s.db.QueryRow(`SELECT id, ticket_id, engineer_id, check_in_at, check_out_at, job_type,
		file_url, status, billed_amount, currency, invoice_id, created_at, updated_at FROM timesheets WHERE id=$1`, id)
	t, err := scanTimesheet(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	return t, err
}

func (s *timesheetStore) List(ticketID, engineerID string) []*models.Timesheet {
	query := `SELECT id, ticket_id, engineer_id, check_in_at, check_out_at, job_type,
		file_url, status, billed_amount, currency, invoice_id, created_at, updated_at FROM timesheets WHERE 1=1`
	args := []interface{}{}
	if ticketID != "" {
		args = append(args, ticketID)
		query += " AND ticket_id = $" + itoa(len(args))
	}
	if engineerID != "" {
		args = append(args, engineerID)
		query += " AND engineer_id = $" + itoa(len(args))
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.Timesheet, 0)
	for rows.Next() {
		t, err := scanTimesheetRows(rows)
		if err == nil {
			out = append(out, t)
		}
	}
	return out
}

func (s *timesheetStore) Approve(id string, hourlyRate, dayRate float64, currency string) (*models.Timesheet, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if t.Status != models.TimesheetPendingReview {
		return nil, store.ErrTimesheetNotPending
	}
	var amount float64
	switch t.JobType {
	case models.JobHourly:
		hours := t.CheckOutAt.Sub(t.CheckInAt).Hours()
		if hours < 0 {
			hours = 0
		}
		amount = hours * hourlyRate
	case models.JobHalfDay:
		amount = dayRate / 2
	case models.JobFullDay:
		amount = dayRate
	}
	amount = round2(amount)
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE timesheets SET status=$1, billed_amount=$2, currency=$3, updated_at=$4 WHERE id=$5`,
		models.TimesheetApproved, amount, currency, now, id)
	if err != nil {
		return nil, err
	}
	t.Status = models.TimesheetApproved
	t.BilledAmount = amount
	t.Currency = currency
	t.UpdatedAt = now
	return t, nil
}

func (s *timesheetStore) Reject(id string) (*models.Timesheet, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if t.Status != models.TimesheetPendingReview {
		return nil, store.ErrTimesheetNotPending
	}
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE timesheets SET status=$1, updated_at=$2 WHERE id=$3`, models.TimesheetRejected, now, id)
	if err != nil {
		return nil, err
	}
	t.Status = models.TimesheetRejected
	t.UpdatedAt = now
	return t, nil
}

func (s *timesheetStore) MarkInvoiced(id, invoiceID string) (*models.Timesheet, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if t.Status != models.TimesheetApproved {
		return nil, store.ErrTimesheetNotApproved
	}
	if t.InvoiceID != "" {
		return nil, store.ErrTimesheetAlreadyInvoiced
	}
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE timesheets SET invoice_id=$1, updated_at=$2 WHERE id=$3`, invoiceID, now, id)
	if err != nil {
		return nil, err
	}
	t.InvoiceID = invoiceID
	t.UpdatedAt = now
	return t, nil
}

func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}

func scanTimesheet(row *sql.Row) (*models.Timesheet, error) {
	return scanTimesheetGeneric(row)
}
func scanTimesheetRows(rows *sql.Rows) (*models.Timesheet, error) {
	return scanTimesheetGeneric(rows)
}
func scanTimesheetGeneric(row rowScanner) (*models.Timesheet, error) {
	var t models.Timesheet
	err := row.Scan(&t.ID, &t.TicketID, &t.EngineerID, &t.CheckInAt, &t.CheckOutAt, &t.JobType,
		&t.FileURL, &t.Status, &t.BilledAmount, &t.Currency, &t.InvoiceID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
