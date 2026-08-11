package models

import "time"

// Entity represents one legal/billing entity in the multi-company setup
// described in the SOW (e.g. separate Germany / UK / Netherlands books).
type Entity struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Country         string `json:"country"`
	DefaultCurrency string `json:"default_currency"`
}

type InvoiceStatus string

const (
	InvoiceDraft         InvoiceStatus = "Draft"
	InvoiceSent          InvoiceStatus = "Sent"
	InvoicePartiallyPaid InvoiceStatus = "Partially Paid"
	InvoicePaid          InvoiceStatus = "Paid"
	InvoiceOverdue       InvoiceStatus = "Overdue"
	InvoiceCancelled     InvoiceStatus = "Cancelled"
)

type InvoiceLineItem struct {
	TicketID    string  `json:"ticket_id"`
	TimesheetID string  `json:"timesheet_id"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

// Invoice groups one or more approved timesheets into a single bill to a
// client, under one entity and one currency.
type Invoice struct {
	ID         string            `json:"id"`
	EntityID   string            `json:"entity_id"`
	ClientName string            `json:"client_name"`
	Currency   string            `json:"currency"`
	LineItems  []InvoiceLineItem `json:"line_items"`
	Total      float64           `json:"total"`
	Status     InvoiceStatus     `json:"status"`

	// AmountPaid tracks partial payments — when it's more than 0 but less
	// than Total, the invoice sits in "Partially Paid".
	AmountPaid float64    `json:"amount_paid"`
	DueDate    *time.Time `json:"due_date,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	SentAt    *time.Time `json:"sent_at,omitempty"`
	PaidAt    *time.Time `json:"paid_at,omitempty"`
}

// VendorBill is the cost side of a ticket — what the business pays the
// engineer (or a subcontractor) — used alongside the client Invoice to
// compute per-ticket profitability (SOW: "Purchase Order & Vendor Bill
// Linking", "Profitability Tracking").
type VendorBillStatus string

const (
	VendorBillDraft    VendorBillStatus = "Draft"
	VendorBillApproved VendorBillStatus = "Approved"
	VendorBillPaid     VendorBillStatus = "Paid"
)

type VendorBill struct {
	ID         string           `json:"id"`
	EntityID   string           `json:"entity_id"`
	VendorName string           `json:"vendor_name"` // usually the engineer's name
	TicketID   string           `json:"ticket_id,omitempty"`
	Amount     float64          `json:"amount"`
	Currency   string           `json:"currency"`
	Status     VendorBillStatus `json:"status"`
	Notes      string           `json:"notes,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
}

// CreditNote / DebitNote adjust a previously issued invoice — a refund or
// correction (SOW: "Credit notes / Debit notes").
type NoteType string

const (
	NoteCredit NoteType = "Credit"
	NoteDebit  NoteType = "Debit"
)

type AdjustmentNote struct {
	ID        string    `json:"id"`
	InvoiceID string    `json:"invoice_id"`
	Type      NoteType  `json:"type"`
	Amount    float64   `json:"amount"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type NewVendorBillInput struct {
	EntityID   string  `json:"entity_id"`
	VendorName string  `json:"vendor_name"`
	TicketID   string  `json:"ticket_id,omitempty"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	Notes      string  `json:"notes,omitempty"`
}

type NewAdjustmentNoteInput struct {
	InvoiceID string   `json:"invoice_id"`
	Type      NoteType `json:"type"`
	Amount    float64  `json:"amount"`
	Reason    string   `json:"reason"`
}

type NewEntityInput struct {
	Name            string `json:"name"`
	Country         string `json:"country"`
	DefaultCurrency string `json:"default_currency"`
}

type NewInvoiceInput struct {
	EntityID     string   `json:"entity_id"`
	ClientName   string   `json:"client_name"`
	TimesheetIDs []string `json:"timesheet_ids"`
}
