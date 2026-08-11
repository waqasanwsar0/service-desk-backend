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

type accountingStore struct {
	db *sql.DB
}

func NewAccountingStore(conn *sql.DB) store.AccountingStore {
	return &accountingStore{db: conn}
}

// --- Entities ---

func (s *accountingStore) CreateEntity(e *models.Entity) error {
	id, err := db.NextID(s.db, "seq_entity", "ENT", 3)
	if err != nil {
		return err
	}
	e.ID = id
	_, err = s.db.Exec(`INSERT INTO entities (id, name, country, default_currency) VALUES ($1,$2,$3,$4)`,
		e.ID, e.Name, e.Country, e.DefaultCurrency)
	return err
}

func (s *accountingStore) ListEntities() []*models.Entity {
	rows, err := s.db.Query(`SELECT id, name, country, default_currency FROM entities ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.Entity, 0)
	for rows.Next() {
		var e models.Entity
		if err := rows.Scan(&e.ID, &e.Name, &e.Country, &e.DefaultCurrency); err == nil {
			out = append(out, &e)
		}
	}
	return out
}

func (s *accountingStore) GetEntity(id string) (*models.Entity, error) {
	var e models.Entity
	err := s.db.QueryRow(`SELECT id, name, country, default_currency FROM entities WHERE id=$1`, id).
		Scan(&e.ID, &e.Name, &e.Country, &e.DefaultCurrency)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrEntityNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// --- Invoices ---

func (s *accountingStore) CreateInvoice(inv *models.Invoice) error {
	if _, err := s.GetEntity(inv.EntityID); err != nil {
		return err
	}
	if len(inv.LineItems) == 0 {
		return store.ErrNoLineItems
	}
	id, err := db.NextID(s.db, "seq_invoice", "INV", 4)
	if err != nil {
		return err
	}
	inv.ID = id
	inv.Status = models.InvoiceDraft
	inv.CreatedAt = time.Now().UTC()

	var total float64
	for _, li := range inv.LineItems {
		total += li.Amount
	}
	inv.Total = round2(total)

	lineItems, _ := json.Marshal(inv.LineItems)
	_, err = s.db.Exec(`INSERT INTO invoices (id, entity_id, client_name, currency, line_items, total, status, amount_paid, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		inv.ID, inv.EntityID, inv.ClientName, inv.Currency, lineItems, inv.Total, inv.Status, 0.0, inv.CreatedAt)
	return err
}

func (s *accountingStore) GetInvoice(id string) (*models.Invoice, error) {
	row := s.db.QueryRow(`SELECT id, entity_id, client_name, currency, line_items, total, status, amount_paid,
		due_date, created_at, sent_at, paid_at FROM invoices WHERE id=$1`, id)
	inv, err := scanInvoice(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	return inv, err
}

func (s *accountingStore) ListInvoices(entityID, clientName, status string) []*models.Invoice {
	query := `SELECT id, entity_id, client_name, currency, line_items, total, status, amount_paid,
		due_date, created_at, sent_at, paid_at FROM invoices WHERE 1=1`
	args := []interface{}{}
	if entityID != "" {
		args = append(args, entityID)
		query += " AND entity_id = $" + itoa(len(args))
	}
	if clientName != "" {
		args = append(args, clientName)
		query += " AND client_name = $" + itoa(len(args))
	}
	if status != "" {
		args = append(args, status)
		query += " AND status = $" + itoa(len(args))
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.Invoice, 0)
	for rows.Next() {
		inv, err := scanInvoiceRows(rows)
		if err == nil {
			out = append(out, inv)
		}
	}
	return out
}

func (s *accountingStore) MarkSent(id string) (*models.Invoice, error) {
	inv, err := s.GetInvoice(id)
	if err != nil {
		return nil, store.ErrNotFound
	}
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE invoices SET status=$1, sent_at=$2 WHERE id=$3`, models.InvoiceSent, now, id)
	if err != nil {
		return nil, err
	}
	inv.Status = models.InvoiceSent
	inv.SentAt = &now
	return inv, nil
}

func (s *accountingStore) MarkPaid(id string) (*models.Invoice, error) {
	inv, err := s.GetInvoice(id)
	if err != nil {
		return nil, store.ErrNotFound
	}
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE invoices SET status=$1, amount_paid=total, paid_at=$2 WHERE id=$3`, models.InvoicePaid, now, id)
	if err != nil {
		return nil, err
	}
	inv.Status = models.InvoicePaid
	inv.AmountPaid = inv.Total
	inv.PaidAt = &now
	return inv, nil
}

func (s *accountingStore) RecordPayment(id string, amount float64) (*models.Invoice, error) {
	inv, err := s.GetInvoice(id)
	if err != nil {
		return nil, store.ErrNotFound
	}
	if inv.Status == models.InvoiceCancelled {
		return nil, errors.New("cannot record payment on a cancelled invoice")
	}
	newPaid := round2(inv.AmountPaid + amount)
	now := time.Now().UTC()
	var newStatus models.InvoiceStatus
	var paidAt *time.Time
	if newPaid >= inv.Total {
		newStatus = models.InvoicePaid
		paidAt = &now
	} else {
		newStatus = models.InvoicePartiallyPaid
	}
	_, err = s.db.Exec(`UPDATE invoices SET amount_paid=$1, status=$2, paid_at=$3 WHERE id=$4`, newPaid, newStatus, paidAt, id)
	if err != nil {
		return nil, err
	}
	inv.AmountPaid = newPaid
	inv.Status = newStatus
	inv.PaidAt = paidAt
	return inv, nil
}

func (s *accountingStore) CancelInvoice(id string) (*models.Invoice, error) {
	inv, err := s.GetInvoice(id)
	if err != nil {
		return nil, store.ErrNotFound
	}
	if inv.Status == models.InvoicePaid {
		return nil, errors.New("cannot cancel a fully paid invoice")
	}
	_, err = s.db.Exec(`UPDATE invoices SET status=$1 WHERE id=$2`, models.InvoiceCancelled, id)
	if err != nil {
		return nil, err
	}
	inv.Status = models.InvoiceCancelled
	return inv, nil
}

func (s *accountingStore) MarkOverdue(id string) (*models.Invoice, error) {
	inv, err := s.GetInvoice(id)
	if err != nil {
		return nil, store.ErrNotFound
	}
	if inv.Status == models.InvoicePaid || inv.Status == models.InvoiceCancelled {
		return nil, errors.New("invoice is already " + string(inv.Status))
	}
	_, err = s.db.Exec(`UPDATE invoices SET status=$1 WHERE id=$2`, models.InvoiceOverdue, id)
	if err != nil {
		return nil, err
	}
	inv.Status = models.InvoiceOverdue
	return inv, nil
}

// --- Vendor bills ---

func (s *accountingStore) CreateVendorBill(b *models.VendorBill) error {
	if _, err := s.GetEntity(b.EntityID); err != nil {
		return err
	}
	id, err := db.NextID(s.db, "seq_vendor_bill", "BILL", 4)
	if err != nil {
		return err
	}
	b.ID = id
	b.Status = models.VendorBillDraft
	b.CreatedAt = time.Now().UTC()
	_, err = s.db.Exec(`INSERT INTO vendor_bills (id, entity_id, vendor_name, ticket_id, amount, currency, status, notes, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		b.ID, b.EntityID, b.VendorName, b.TicketID, b.Amount, b.Currency, b.Status, b.Notes, b.CreatedAt)
	return err
}

func (s *accountingStore) ListVendorBills(entityID, ticketID string) []*models.VendorBill {
	query := `SELECT id, entity_id, vendor_name, ticket_id, amount, currency, status, notes, created_at FROM vendor_bills WHERE 1=1`
	args := []interface{}{}
	if entityID != "" {
		args = append(args, entityID)
		query += " AND entity_id = $" + itoa(len(args))
	}
	if ticketID != "" {
		args = append(args, ticketID)
		query += " AND ticket_id = $" + itoa(len(args))
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.VendorBill, 0)
	for rows.Next() {
		var b models.VendorBill
		if err := rows.Scan(&b.ID, &b.EntityID, &b.VendorName, &b.TicketID, &b.Amount, &b.Currency, &b.Status, &b.Notes, &b.CreatedAt); err == nil {
			out = append(out, &b)
		}
	}
	return out
}

func (s *accountingStore) ApproveVendorBill(id string) (*models.VendorBill, error) {
	return s.updateVendorBillStatus(id, models.VendorBillApproved)
}

func (s *accountingStore) PayVendorBill(id string) (*models.VendorBill, error) {
	return s.updateVendorBillStatus(id, models.VendorBillPaid)
}

func (s *accountingStore) updateVendorBillStatus(id string, status models.VendorBillStatus) (*models.VendorBill, error) {
	res, err := s.db.Exec(`UPDATE vendor_bills SET status=$1 WHERE id=$2`, status, id)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, store.ErrNotFound
	}
	var b models.VendorBill
	err = s.db.QueryRow(`SELECT id, entity_id, vendor_name, ticket_id, amount, currency, status, notes, created_at FROM vendor_bills WHERE id=$1`, id).
		Scan(&b.ID, &b.EntityID, &b.VendorName, &b.TicketID, &b.Amount, &b.Currency, &b.Status, &b.Notes, &b.CreatedAt)
	return &b, err
}

// --- Adjustment notes ---

func (s *accountingStore) CreateAdjustmentNote(n *models.AdjustmentNote) error {
	if _, err := s.GetInvoice(n.InvoiceID); err != nil {
		return store.ErrNotFound
	}
	id, err := db.NextID(s.db, "seq_adjustment", "ADJ", 4)
	if err != nil {
		return err
	}
	n.ID = id
	n.CreatedAt = time.Now().UTC()
	_, err = s.db.Exec(`INSERT INTO adjustment_notes (id, invoice_id, type, amount, reason, created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		n.ID, n.InvoiceID, n.Type, n.Amount, n.Reason, n.CreatedAt)
	return err
}

func (s *accountingStore) ListAdjustmentNotes(invoiceID string) []*models.AdjustmentNote {
	rows, err := s.db.Query(`SELECT id, invoice_id, type, amount, reason, created_at FROM adjustment_notes WHERE invoice_id=$1 ORDER BY created_at DESC`, invoiceID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.AdjustmentNote, 0)
	for rows.Next() {
		var n models.AdjustmentNote
		if err := rows.Scan(&n.ID, &n.InvoiceID, &n.Type, &n.Amount, &n.Reason, &n.CreatedAt); err == nil {
			out = append(out, &n)
		}
	}
	return out
}

func scanInvoice(row *sql.Row) (*models.Invoice, error) {
	return scanInvoiceGeneric(row)
}
func scanInvoiceRows(rows *sql.Rows) (*models.Invoice, error) {
	return scanInvoiceGeneric(rows)
}
func scanInvoiceGeneric(row rowScanner) (*models.Invoice, error) {
	var inv models.Invoice
	var lineItems []byte
	var dueDate, sentAt, paidAt sql.NullTime
	err := row.Scan(&inv.ID, &inv.EntityID, &inv.ClientName, &inv.Currency, &lineItems, &inv.Total, &inv.Status,
		&inv.AmountPaid, &dueDate, &inv.CreatedAt, &sentAt, &paidAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(lineItems, &inv.LineItems)
	if dueDate.Valid {
		inv.DueDate = &dueDate.Time
	}
	if sentAt.Valid {
		inv.SentAt = &sentAt.Time
	}
	if paidAt.Valid {
		inv.PaidAt = &paidAt.Time
	}
	return &inv, nil
}
