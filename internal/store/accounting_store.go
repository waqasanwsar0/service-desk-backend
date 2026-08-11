package store

import (
	"errors"
	"sort"
	"sync"
	"time"

	"servicedesk/internal/models"
)

var ErrEntityNotFound = errors.New("entity not found")
var ErrCurrencyMismatch = errors.New("all timesheets on an invoice must share the entity's currency")
var ErrNoLineItems = errors.New("invoice must have at least one line item")

type AccountingStore interface {
	CreateEntity(e *models.Entity) error
	ListEntities() []*models.Entity
	GetEntity(id string) (*models.Entity, error)

	CreateInvoice(inv *models.Invoice) error
	GetInvoice(id string) (*models.Invoice, error)
	ListInvoices(entityID, clientName, status string) []*models.Invoice
	MarkSent(id string) (*models.Invoice, error)
	MarkPaid(id string) (*models.Invoice, error)
	// RecordPayment applies a payment toward an invoice. If the payment
	// covers the remaining balance the invoice becomes Paid; otherwise it
	// becomes Partially Paid.
	RecordPayment(id string, amount float64) (*models.Invoice, error)
	CancelInvoice(id string) (*models.Invoice, error)
	MarkOverdue(id string) (*models.Invoice, error)

	// Vendor bills — the cost side of a ticket, for profitability tracking.
	CreateVendorBill(b *models.VendorBill) error
	ListVendorBills(entityID, ticketID string) []*models.VendorBill
	ApproveVendorBill(id string) (*models.VendorBill, error)
	PayVendorBill(id string) (*models.VendorBill, error)

	// Credit / debit notes against an invoice.
	CreateAdjustmentNote(n *models.AdjustmentNote) error
	ListAdjustmentNotes(invoiceID string) []*models.AdjustmentNote
}

type memoryAccountingStore struct {
	mu          sync.RWMutex
	entities    map[string]*models.Entity
	invoices    map[string]*models.Invoice
	vendorBills map[string]*models.VendorBill
	notes       map[string][]*models.AdjustmentNote // keyed by invoiceID
	entSeq      int
	invSeq      int
	billSeq     int
	noteSeq     int
}

func NewMemoryAccountingStore() AccountingStore {
	return &memoryAccountingStore{
		entities:    make(map[string]*models.Entity),
		invoices:    make(map[string]*models.Invoice),
		vendorBills: make(map[string]*models.VendorBill),
		notes:       make(map[string][]*models.AdjustmentNote),
	}
}

func (s *memoryAccountingStore) CreateEntity(e *models.Entity) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entSeq++
	e.ID = "ENT-" + padLeft(s.entSeq, 3)
	s.entities[e.ID] = e
	return nil
}

func (s *memoryAccountingStore) ListEntities() []*models.Entity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.Entity, 0, len(s.entities))
	for _, e := range s.entities {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *memoryAccountingStore) GetEntity(id string) (*models.Entity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.entities[id]
	if !ok {
		return nil, ErrEntityNotFound
	}
	return e, nil
}

func (s *memoryAccountingStore) CreateInvoice(inv *models.Invoice) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entities[inv.EntityID]; !ok {
		return ErrEntityNotFound
	}
	if len(inv.LineItems) == 0 {
		return ErrNoLineItems
	}

	s.invSeq++
	now := time.Now().UTC()
	inv.ID = "INV-" + time.Now().Format("200601") + "-" + padLeft(s.invSeq, 4)
	inv.Status = models.InvoiceDraft
	inv.CreatedAt = now

	var total float64
	for _, li := range inv.LineItems {
		total += li.Amount
	}
	inv.Total = round2(total)

	s.invoices[inv.ID] = inv
	return nil
}

func (s *memoryAccountingStore) GetInvoice(id string) (*models.Invoice, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	inv, ok := s.invoices[id]
	if !ok {
		return nil, ErrNotFound
	}
	return inv, nil
}

func (s *memoryAccountingStore) ListInvoices(entityID, clientName, status string) []*models.Invoice {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.Invoice, 0)
	for _, inv := range s.invoices {
		if entityID != "" && inv.EntityID != entityID {
			continue
		}
		if clientName != "" && inv.ClientName != clientName {
			continue
		}
		if status != "" && string(inv.Status) != status {
			continue
		}
		out = append(out, inv)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *memoryAccountingStore) MarkSent(id string) (*models.Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, ok := s.invoices[id]
	if !ok {
		return nil, ErrNotFound
	}
	now := time.Now().UTC()
	inv.Status = models.InvoiceSent
	inv.SentAt = &now
	return inv, nil
}

func (s *memoryAccountingStore) MarkPaid(id string) (*models.Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, ok := s.invoices[id]
	if !ok {
		return nil, ErrNotFound
	}
	now := time.Now().UTC()
	inv.Status = models.InvoicePaid
	inv.AmountPaid = inv.Total
	inv.PaidAt = &now
	return inv, nil
}

// RecordPayment applies a partial or full payment. Covers the SOW's
// "Partially Paid" status alongside the simple MarkPaid shortcut above.
func (s *memoryAccountingStore) RecordPayment(id string, amount float64) (*models.Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, ok := s.invoices[id]
	if !ok {
		return nil, ErrNotFound
	}
	if inv.Status == models.InvoiceCancelled {
		return nil, errors.New("cannot record payment on a cancelled invoice")
	}
	inv.AmountPaid = round2(inv.AmountPaid + amount)
	now := time.Now().UTC()
	if inv.AmountPaid >= inv.Total {
		inv.Status = models.InvoicePaid
		inv.PaidAt = &now
	} else {
		inv.Status = models.InvoicePartiallyPaid
	}
	return inv, nil
}

func (s *memoryAccountingStore) CancelInvoice(id string) (*models.Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, ok := s.invoices[id]
	if !ok {
		return nil, ErrNotFound
	}
	if inv.Status == models.InvoicePaid {
		return nil, errors.New("cannot cancel a fully paid invoice")
	}
	inv.Status = models.InvoiceCancelled
	return inv, nil
}

func (s *memoryAccountingStore) MarkOverdue(id string) (*models.Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, ok := s.invoices[id]
	if !ok {
		return nil, ErrNotFound
	}
	if inv.Status == models.InvoicePaid || inv.Status == models.InvoiceCancelled {
		return nil, errors.New("invoice is already " + string(inv.Status))
	}
	inv.Status = models.InvoiceOverdue
	return inv, nil
}

// --- Vendor bills ---

func (s *memoryAccountingStore) CreateVendorBill(b *models.VendorBill) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entities[b.EntityID]; !ok {
		return ErrEntityNotFound
	}
	s.billSeq++
	b.ID = "BILL-" + padLeft(s.billSeq, 4)
	b.Status = models.VendorBillDraft
	b.CreatedAt = time.Now().UTC()
	s.vendorBills[b.ID] = b
	return nil
}

func (s *memoryAccountingStore) ListVendorBills(entityID, ticketID string) []*models.VendorBill {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.VendorBill, 0)
	for _, b := range s.vendorBills {
		if entityID != "" && b.EntityID != entityID {
			continue
		}
		if ticketID != "" && b.TicketID != ticketID {
			continue
		}
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *memoryAccountingStore) ApproveVendorBill(id string) (*models.VendorBill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.vendorBills[id]
	if !ok {
		return nil, ErrNotFound
	}
	b.Status = models.VendorBillApproved
	return b, nil
}

func (s *memoryAccountingStore) PayVendorBill(id string) (*models.VendorBill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.vendorBills[id]
	if !ok {
		return nil, ErrNotFound
	}
	b.Status = models.VendorBillPaid
	return b, nil
}

// --- Credit / debit notes ---

func (s *memoryAccountingStore) CreateAdjustmentNote(n *models.AdjustmentNote) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.invoices[n.InvoiceID]; !ok {
		return ErrNotFound
	}
	s.noteSeq++
	n.ID = "ADJ-" + padLeft(s.noteSeq, 4)
	n.CreatedAt = time.Now().UTC()
	s.notes[n.InvoiceID] = append(s.notes[n.InvoiceID], n)
	return nil
}

func (s *memoryAccountingStore) ListAdjustmentNotes(invoiceID string) []*models.AdjustmentNote {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.notes[invoiceID]
}
