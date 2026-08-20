package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type DispatchHandler struct {
	Dispatches store.DispatchStore
	Tickets    store.TicketStore
	Engineers  store.EngineerStore
	Notifier   *Notifier
}

func NewDispatchHandler(d store.DispatchStore, t store.TicketStore, e store.EngineerStore, n *Notifier) *DispatchHandler {
	return &DispatchHandler{Dispatches: d, Tickets: t, Engineers: e, Notifier: n}
}

// CreateDispatch handles POST /api/dispatches
// This is the SOW's "tickets automatically generate when a dispatch is
// created" flow: creating a dispatch record creates the ticket for it in
// one step, and — if an engineer is specified — assigns them immediately,
// which is what makes the ticket visible to that engineer (and to the
// client, via the ticket's assigned_engineer fields) without any
// separate "share" action.
func (h *DispatchHandler) CreateDispatch(w http.ResponseWriter, r *http.Request) {
	var in models.NewDispatchInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.ClientName == "" || in.Title == "" {
		writeError(w, http.StatusBadRequest, "client_name and title are required")
		return
	}
	if in.Priority == "" {
		in.Priority = models.PriorityMedium
	}

	ticket := &models.Ticket{
		Title:       in.Title,
		Description: in.Description,
		ClientName:  in.ClientName,
		ProjectID:   in.ProjectID,
		ProjectType: models.ProjectTypeDispatch,
		Domain:      in.Domain,
		SiteAddress: in.SiteAddress,
		Priority:    in.Priority,
		Source:      models.SourceManual,
	}
	if err := h.Tickets.Create(ticket); err != nil {
		writeError(w, http.StatusInternalServerError, "failed creating ticket: "+err.Error())
		return
	}

	var engineerName string
	if in.EngineerID != "" {
		eng, err := h.Engineers.Get(in.EngineerID)
		if err != nil {
			writeError(w, http.StatusNotFound, "engineer not found")
			return
		}
		engineerName = eng.Name
		updated, err := h.Tickets.Assign(ticket.ID, in.EngineerID, eng.Name)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed assigning engineer: "+err.Error())
			return
		}
		ticket = updated

		if h.Notifier != nil {
			h.Notifier.Send(eng.Email, "New dispatch assigned: "+ticket.Title,
				"You've been assigned to a new dispatch for "+in.ClientName+".\n\nTicket: "+ticket.ID+"\nSite: "+in.SiteAddress)
		}
	}

	scheduledAt := in.ScheduledAt
	if scheduledAt.IsZero() {
		scheduledAt = time.Now().UTC()
	}
	d := &models.Dispatch{
		TicketID:    ticket.ID,
		ProjectID:   in.ProjectID,
		ClientName:  in.ClientName,
		SiteAddress: in.SiteAddress,
		EngineerID:  in.EngineerID,
		ScheduledAt: scheduledAt,
	}
	if err := h.Dispatches.Create(d); err != nil {
		writeError(w, http.StatusInternalServerError, "failed recording dispatch: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"dispatch":      d,
		"ticket":        ticket,
		"engineer_name": engineerName,
	})
}

// ListDispatches handles GET /api/dispatches?project_id=
func (h *DispatchHandler) ListDispatches(w http.ResponseWriter, r *http.Request) {
	list := h.Dispatches.List(r.URL.Query().Get("project_id"))
	writeJSON(w, http.StatusOK, map[string]interface{}{"count": len(list), "dispatches": list})
}
