package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type TicketHandler struct {
	Store     store.TicketStore
	Engineers store.EngineerStore
	Notifier  *Notifier
}

func NewTicketHandler(s store.TicketStore, engineers store.EngineerStore, notifier *Notifier) *TicketHandler {
	return &TicketHandler{Store: s, Engineers: engineers, Notifier: notifier}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// CreateTicket handles POST /api/tickets
// Used by: manual ticket creation (Service Desk staff) and the web form.
func (h *TicketHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	var in models.NewTicketInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if in.Title == "" || in.ClientName == "" {
		writeError(w, http.StatusBadRequest, "title and client_name are required")
		return
	}
	if in.Priority == "" {
		in.Priority = models.PriorityMedium
	}
	if in.Source == "" {
		in.Source = models.SourceManual
	}
	if in.ProjectType == "" {
		in.ProjectType = models.ProjectTypeDispatch
	}

	t := &models.Ticket{
		Title:       in.Title,
		Description: in.Description,
		ClientName:  in.ClientName,
		ProjectID:   in.ProjectID,
		ProjectName: in.ProjectName,
		ProjectType: in.ProjectType,
		Country:     in.Country,
		Domain:      in.Domain,
		SiteAddress: in.SiteAddress,
		Priority:    in.Priority,
		Source:      in.Source,
		ImageURLs:   in.ImageURLs,
	}
	if in.SLAHours != nil {
		due := time.Now().UTC().Add(time.Duration(*in.SLAHours) * time.Hour)
		t.SLADueAt = &due
	}

	if err := h.Store.Create(t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// ImportFromEmail handles POST /api/tickets/import/email
// This is the landing point for the Microsoft Graph (Outlook) integration.
// A separate poller/webhook service will call this endpoint once new mail
// arrives; for now it accepts the same payload shape and tags source=email.
func (h *TicketHandler) ImportFromEmail(w http.ResponseWriter, r *http.Request) {
	var in models.NewTicketInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	in.Source = models.SourceEmail
	if in.Priority == "" {
		in.Priority = models.PriorityMedium
	}
	if in.ProjectType == "" {
		in.ProjectType = models.ProjectTypeDispatch
	}

	t := &models.Ticket{
		Title:       in.Title,
		Description: in.Description,
		ClientName:  in.ClientName,
		ProjectName: in.ProjectName,
		ProjectType: in.ProjectType,
		Domain:      in.Domain,
		SiteAddress: in.SiteAddress,
		Priority:    in.Priority,
		Source:      models.SourceEmail,
	}

	if err := h.Store.Create(t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// GetTicket handles GET /api/tickets/{id}
func (h *TicketHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.Store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// ListTickets handles GET /api/tickets?status=&client_name=&priority=&project_type=
func (h *TicketHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := store.ListFilter{
		Status:      q.Get("status"),
		ClientName:  q.Get("client_name"),
		Priority:    q.Get("priority"),
		ProjectType: q.Get("project_type"),
	}
	tickets := h.Store.List(filter)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":   len(tickets),
		"tickets": tickets,
	})
}

// UpdateStatus handles PATCH /api/tickets/{id}/status
// Body: {"status": "Onsite"}
func (h *TicketHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	newStatus := models.TicketStatus(body.Status)
	if !models.IsValidStatus(newStatus) {
		writeError(w, http.StatusBadRequest, "unknown status: "+body.Status)
		return
	}

	t, err := h.Store.UpdateStatus(id, newStatus)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "ticket not found")
			return
		}
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// AssignEngineer handles PATCH /api/tickets/{id}/assign
// Body: {"engineer_id": "ENG-001", "engineer_name": "Ali Raza"}
func (h *TicketHandler) AssignEngineer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		EngineerID   string `json:"engineer_id"`
		EngineerName string `json:"engineer_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if body.EngineerID == "" {
		writeError(w, http.StatusBadRequest, "engineer_id is required")
		return
	}

	t, err := h.Store.Assign(id, body.EngineerID, body.EngineerName)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "ticket not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if h.Notifier != nil && h.Engineers != nil {
		if eng, err := h.Engineers.Get(body.EngineerID); err == nil {
			h.Notifier.Send(eng.Email, "Ticket assigned: "+t.Title,
				"You've been assigned to ticket "+t.ID+" ("+t.Title+") for "+t.ClientName+".\n\nSite: "+t.SiteAddress)
		}
	}

	writeJSON(w, http.StatusOK, t)
}

// AddImage handles PATCH /api/tickets/{id}/images
// Body: {"image_url": "https://…"} — attaches a photo (e.g. of the
// issue, or proof of work) to the ticket.
func (h *TicketHandler) AddImage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		ImageURL string `json:"image_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if body.ImageURL == "" {
		writeError(w, http.StatusBadRequest, "image_url is required")
		return
	}

	t, err := h.Store.AddImage(id, body.ImageURL)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "ticket not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}
