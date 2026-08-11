package handlers

import (
	"encoding/json"
	"net/http"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type AccountingHandler struct {
	Accounting store.AccountingStore
	Timesheets store.TimesheetStore
	Tickets    store.TicketStore
}

func NewAccountingHandler(a store.AccountingStore, t store.TimesheetStore, tk store.TicketStore) *AccountingHandler {
	return &AccountingHandler{Accounting: a, Timesheets: t, Tickets: tk}
}

// CreateEntity handles POST /api/entities
// Backs the SOW's multi-company requirement (separate Germany/UK/NL books).
func (h *AccountingHandler) CreateEntity(w http.ResponseWriter, r *http.Request) {
	var in models.NewEntityInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.Name == "" || in.DefaultCurrency == "" {
		writeError(w, http.StatusBadRequest, "name and default_currency are required")
		return
	}
	e := &models.Entity{Name: in.Name, Country: in.Country, DefaultCurrency: in.DefaultCurrency}
	if err := h.Accounting.CreateEntity(e); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// ListEntities handles GET /api/entities
func (h *AccountingHandler) ListEntities(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"entities": h.Accounting.ListEntities()})
}

// CreateInvoice handles POST /api/invoices
// Body: {"entity_id": "...", "client_name": "...", "timesheet_ids": ["TSH-0001", ...]}
// Pulls in each approved, not-yet-invoiced timesheet as a line item, enforces
// a single currency per invoice (matching the entity's currency), marks the
// timesheets as invoiced, and advances their tickets: Completed -> Invoice.
func (h *AccountingHandler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	var in models.NewInvoiceInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.EntityID == "" || in.ClientName == "" || len(in.TimesheetIDs) == 0 {
		writeError(w, http.StatusBadRequest, "entity_id, client_name and at least one timesheet_id are required")
		return
	}

	entity, err := h.Accounting.GetEntity(in.EntityID)
	if err != nil {
		writeError(w, http.StatusNotFound, "entity not found")
		return
	}

	lineItems := make([]models.InvoiceLineItem, 0, len(in.TimesheetIDs))
	for _, tsID := range in.TimesheetIDs {
		ts, err := h.Timesheets.Get(tsID)
		if err != nil {
			writeError(w, http.StatusNotFound, "timesheet not found: "+tsID)
			return
		}
		if ts.Status != models.TimesheetApproved {
			writeError(w, http.StatusConflict, "timesheet "+tsID+" is not approved yet")
			return
		}
		if ts.InvoiceID != "" {
			writeError(w, http.StatusConflict, "timesheet "+tsID+" has already been invoiced")
			return
		}
		if ts.Currency != entity.DefaultCurrency {
			writeError(w, http.StatusConflict, "timesheet "+tsID+" is in "+ts.Currency+
				", but entity "+entity.ID+" bills in "+entity.DefaultCurrency)
			return
		}
		lineItems = append(lineItems, models.InvoiceLineItem{
			TicketID:    ts.TicketID,
			TimesheetID: ts.ID,
			Description: "Dispatch work — " + string(ts.JobType),
			Amount:      ts.BilledAmount,
		})
	}

	inv := &models.Invoice{
		EntityID:   in.EntityID,
		ClientName: in.ClientName,
		Currency:   entity.DefaultCurrency,
		LineItems:  lineItems,
	}
	if err := h.Accounting.CreateInvoice(inv); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Now that the invoice exists, mark each timesheet invoiced and push
	// its ticket forward: Completed -> Invoice.
	for _, li := range lineItems {
		_, _ = h.Timesheets.MarkInvoiced(li.TimesheetID, inv.ID)
		_, _ = h.Tickets.UpdateStatus(li.TicketID, models.StatusInvoice)
	}

	writeJSON(w, http.StatusCreated, inv)
}

// GetInvoice handles GET /api/invoices/{id}
func (h *AccountingHandler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inv, err := h.Accounting.GetInvoice(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// ListInvoices handles GET /api/invoices?entity_id=&client_name=&status=
func (h *AccountingHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list := h.Accounting.ListInvoices(q.Get("entity_id"), q.Get("client_name"), q.Get("status"))
	writeJSON(w, http.StatusOK, map[string]interface{}{"count": len(list), "invoices": list})
}

// MarkInvoiceSent handles PATCH /api/invoices/{id}/send
func (h *AccountingHandler) MarkInvoiceSent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inv, err := h.Accounting.MarkSent(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// MarkInvoicePaid handles PATCH /api/invoices/{id}/paid
// Advances every ticket on the invoice: Invoice -> Paid.
func (h *AccountingHandler) MarkInvoicePaid(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inv, err := h.Accounting.MarkPaid(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}
	for _, li := range inv.LineItems {
		_, _ = h.Tickets.UpdateStatus(li.TicketID, models.StatusPaid)
	}
	writeJSON(w, http.StatusOK, inv)
}

// RecordPayment handles PATCH /api/invoices/{id}/payment
// Body: {"amount": 40.00} — supports partial payments (SOW: "Partially Paid").
// Once the invoice is fully paid, advances every linked ticket to Paid.
func (h *AccountingHandler) RecordPayment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if body.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be greater than zero")
		return
	}
	inv, err := h.Accounting.RecordPayment(id, body.Amount)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "invoice not found")
			return
		}
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if inv.Status == models.InvoicePaid {
		for _, li := range inv.LineItems {
			_, _ = h.Tickets.UpdateStatus(li.TicketID, models.StatusPaid)
		}
	}
	writeJSON(w, http.StatusOK, inv)
}

// CancelInvoice handles PATCH /api/invoices/{id}/cancel
func (h *AccountingHandler) CancelInvoice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inv, err := h.Accounting.CancelInvoice(id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "invoice not found")
			return
		}
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// MarkInvoiceOverdue handles PATCH /api/invoices/{id}/overdue
// In a full system a scheduled job would call this once due_date passes;
// exposed here as a manual trigger since there's no scheduler yet.
func (h *AccountingHandler) MarkInvoiceOverdue(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inv, err := h.Accounting.MarkOverdue(id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "invoice not found")
			return
		}
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// --- Vendor bills ---

// CreateVendorBill handles POST /api/vendor-bills
func (h *AccountingHandler) CreateVendorBill(w http.ResponseWriter, r *http.Request) {
	var in models.NewVendorBillInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.EntityID == "" || in.VendorName == "" || in.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "entity_id, vendor_name and a positive amount are required")
		return
	}
	b := &models.VendorBill{
		EntityID:   in.EntityID,
		VendorName: in.VendorName,
		TicketID:   in.TicketID,
		Amount:     in.Amount,
		Currency:   in.Currency,
		Notes:      in.Notes,
	}
	if err := h.Accounting.CreateVendorBill(b); err != nil {
		if err == store.ErrEntityNotFound {
			writeError(w, http.StatusNotFound, "entity not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

// ListVendorBills handles GET /api/vendor-bills?entity_id=&ticket_id=
func (h *AccountingHandler) ListVendorBills(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list := h.Accounting.ListVendorBills(q.Get("entity_id"), q.Get("ticket_id"))
	writeJSON(w, http.StatusOK, map[string]interface{}{"count": len(list), "vendor_bills": list})
}

// ApproveVendorBill handles PATCH /api/vendor-bills/{id}/approve
func (h *AccountingHandler) ApproveVendorBill(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	b, err := h.Accounting.ApproveVendorBill(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "vendor bill not found")
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// PayVendorBill handles PATCH /api/vendor-bills/{id}/pay
func (h *AccountingHandler) PayVendorBill(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	b, err := h.Accounting.PayVendorBill(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "vendor bill not found")
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// --- Credit / debit notes ---

// CreateAdjustmentNote handles POST /api/adjustment-notes
func (h *AccountingHandler) CreateAdjustmentNote(w http.ResponseWriter, r *http.Request) {
	var in models.NewAdjustmentNoteInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.InvoiceID == "" || in.Amount <= 0 || (in.Type != models.NoteCredit && in.Type != models.NoteDebit) {
		writeError(w, http.StatusBadRequest, `invoice_id, a positive amount, and type ("Credit" or "Debit") are required`)
		return
	}
	n := &models.AdjustmentNote{InvoiceID: in.InvoiceID, Type: in.Type, Amount: in.Amount, Reason: in.Reason}
	if err := h.Accounting.CreateAdjustmentNote(n); err != nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}
	writeJSON(w, http.StatusCreated, n)
}

// ListAdjustmentNotes handles GET /api/invoices/{id}/adjustment-notes
func (h *AccountingHandler) ListAdjustmentNotes(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	list := h.Accounting.ListAdjustmentNotes(id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"count": len(list), "notes": list})
}

// TicketProfitability handles GET /api/tickets/{id}/profitability
// Compares what the client was billed (approved timesheets on this
// ticket) against what was paid out (vendor bills on this ticket) — the
// SOW's "Profitability Tracking (Forecast vs Actual)".
func (h *AccountingHandler) TicketProfitability(w http.ResponseWriter, r *http.Request) {
	ticketID := r.PathValue("id")

	timesheets := h.Timesheets.List(ticketID, "")
	var billed float64
	currency := ""
	for _, ts := range timesheets {
		if ts.Status == models.TimesheetApproved {
			billed += ts.BilledAmount
			if currency == "" {
				currency = ts.Currency
			}
		}
	}

	bills := h.Accounting.ListVendorBills("", ticketID)
	var cost float64
	for _, b := range bills {
		cost += b.Amount
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ticket_id":     ticketID,
		"billed_amount": round2f(billed),
		"vendor_cost":   round2f(cost),
		"profit":        round2f(billed - cost),
		"currency":      currency,
	})
}

func round2f(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}
