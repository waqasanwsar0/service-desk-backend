package handlers

import (
	"encoding/json"
	"net/http"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type LeadHandler struct {
	Store store.LeadStore
}

func NewLeadHandler(s store.LeadStore) *LeadHandler {
	return &LeadHandler{Store: s}
}

// CreateLead handles POST /api/leads
func (h *LeadHandler) CreateLead(w http.ResponseWriter, r *http.Request) {
	var in models.NewLeadInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.CompanyName == "" {
		writeError(w, http.StatusBadRequest, "company_name is required")
		return
	}
	l := &models.Lead{
		CompanyName:  in.CompanyName,
		ContactName:  in.ContactName,
		ContactEmail: in.ContactEmail,
		ContactPhone: in.ContactPhone,
		LinkedInURL:  in.LinkedInURL,
		Source:       in.Source,
		OwnerID:      in.OwnerID,
	}
	if err := h.Store.Create(l); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

// GetLead handles GET /api/leads/{id}
func (h *LeadHandler) GetLead(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	l, err := h.Store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "lead not found")
		return
	}
	writeJSON(w, http.StatusOK, l)
}

// ListLeads handles GET /api/leads?owner_id=&status=
func (h *LeadHandler) ListLeads(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list := h.Store.List(q.Get("owner_id"), q.Get("status"))
	writeJSON(w, http.StatusOK, map[string]interface{}{"count": len(list), "leads": list})
}

// UpdateLeadStatus handles PATCH /api/leads/{id}/status
// Body: {"status": "Qualified"}
func (h *LeadHandler) UpdateLeadStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	status := models.LeadStatus(body.Status)
	if !models.IsValidLeadStatus(status) {
		writeError(w, http.StatusBadRequest, "unknown status: "+body.Status)
		return
	}
	l, err := h.Store.UpdateStatus(id, status)
	if err != nil {
		writeError(w, http.StatusNotFound, "lead not found")
		return
	}
	writeJSON(w, http.StatusOK, l)
}

// AddLeadNote handles POST /api/leads/{id}/notes
func (h *LeadHandler) AddLeadNote(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if body.Text == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}
	claims := ClaimsFromContext(r.Context())
	authorID := ""
	if claims != nil {
		authorID = claims.UserID
	}
	l, err := h.Store.AddNote(id, authorID, body.Text)
	if err != nil {
		writeError(w, http.StatusNotFound, "lead not found")
		return
	}
	writeJSON(w, http.StatusOK, l)
}

// LeadsReport handles GET /api/leads/report
// SOW: "Leads reporting" — pipeline counts by status and by owner.
func (h *LeadHandler) LeadsReport(w http.ResponseWriter, r *http.Request) {
	all := h.Store.List("", "")

	byStatus := map[string]int{}
	byOwner := map[string]int{}
	for _, l := range all {
		byStatus[string(l.Status)]++
		if l.OwnerID != "" {
			byOwner[l.OwnerID]++
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":     len(all),
		"by_status": byStatus,
		"by_owner":  byOwner,
	})
}
