package handlers

import (
	"encoding/json"
	"net/http"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type TimesheetHandler struct {
	Timesheets store.TimesheetStore
	Tickets    store.TicketStore
	Engineers  store.EngineerStore
}

func NewTimesheetHandler(t store.TimesheetStore, tk store.TicketStore, e store.EngineerStore) *TimesheetHandler {
	return &TimesheetHandler{Timesheets: t, Tickets: tk, Engineers: e}
}

// CreateTimesheet handles POST /api/timesheets
// This is the SOW's "Service desk will upload PDF/JPG of timesheet" step.
// If the linked ticket is currently Onsite, it's automatically advanced to
// "Timesheet Pending" — matching the ticket lifecycle in the SOW.
func (h *TimesheetHandler) CreateTimesheet(w http.ResponseWriter, r *http.Request) {
	var in models.NewTimesheetInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.TicketID == "" || in.EngineerID == "" {
		writeError(w, http.StatusBadRequest, "ticket_id and engineer_id are required")
		return
	}
	if !models.IsValidJobType(in.JobType) {
		writeError(w, http.StatusBadRequest, "job_type must be Hourly, HalfDay, or FullDay")
		return
	}

	ticket, err := h.Tickets.Get(in.TicketID)
	if err != nil {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}
	if _, err := h.Engineers.Get(in.EngineerID); err != nil {
		writeError(w, http.StatusNotFound, "engineer not found")
		return
	}

	ts := &models.Timesheet{
		TicketID:   in.TicketID,
		EngineerID: in.EngineerID,
		CheckInAt:  in.CheckInAt,
		CheckOutAt: in.CheckOutAt,
		JobType:    in.JobType,
		FileURL:    in.FileURL,
	}
	if err := h.Timesheets.Create(ts); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Best-effort workflow advance: Onsite -> Timesheet Pending.
	// Ignored if the ticket is already past this point (e.g. a second
	// timesheet on the same ticket) — the transition simply won't apply.
	if ticket.Status == models.StatusOnsite {
		_, _ = h.Tickets.UpdateStatus(in.TicketID, models.StatusTimesheetPending)
	}

	writeJSON(w, http.StatusCreated, ts)
}

// GetTimesheet handles GET /api/timesheets/{id}
func (h *TimesheetHandler) GetTimesheet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.Timesheets.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "timesheet not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// ListTimesheets handles GET /api/timesheets?ticket_id=&engineer_id=
func (h *TimesheetHandler) ListTimesheets(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list := h.Timesheets.List(q.Get("ticket_id"), q.Get("engineer_id"))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":      len(list),
		"timesheets": list,
	})
}

// ApproveTimesheet handles PATCH /api/timesheets/{id}/approve
// Computes the billed amount from the engineer's rate card and advances
// the linked ticket: Timesheet Pending -> Completed.
func (h *TimesheetHandler) ApproveTimesheet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	ts, err := h.Timesheets.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "timesheet not found")
		return
	}
	engineer, err := h.Engineers.Get(ts.EngineerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "engineer not found for this timesheet")
		return
	}
	currency := engineer.Currency
	if currency == "" {
		currency = "EUR"
	}

	approved, err := h.Timesheets.Approve(id, engineer.HourlyRate, engineer.DayRate, currency)
	if err != nil {
		if err == store.ErrTimesheetNotPending {
			writeError(w, http.StatusConflict, "timesheet has already been decided")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Best-effort workflow advance: Timesheet Pending -> Completed.
	_, _ = h.Tickets.UpdateStatus(approved.TicketID, models.StatusCompleted)

	writeJSON(w, http.StatusOK, approved)
}

// SignTimesheet handles PATCH /api/timesheets/{id}/sign
// The SOW's FTE sign-off requirement: the engineer confirms their hours
// are correct before Service Desk reviews and approves the timesheet.
func (h *TimesheetHandler) SignTimesheet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ts, err := h.Timesheets.Sign(id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "timesheet not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ts)
}

// RejectTimesheet handles PATCH /api/timesheets/{id}/reject
func (h *TimesheetHandler) RejectTimesheet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ts, err := h.Timesheets.Reject(id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "timesheet not found")
			return
		}
		if err == store.ErrTimesheetNotPending {
			writeError(w, http.StatusConflict, "timesheet has already been decided")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ts)
}
