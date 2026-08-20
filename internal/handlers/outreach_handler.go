package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type OutreachHandler struct {
	Store store.OutreachStore
}

func NewOutreachHandler(s store.OutreachStore) *OutreachHandler {
	return &OutreachHandler{Store: s}
}

// CreateOutreach handles POST /api/outreach
// Logged by the recruiter right after they send a LinkedIn (or other
// channel) message — there's no API that reports this automatically,
// so this is the manual follow-up log the SOW's LinkedIn integration
// request becomes in practice.
func (h *OutreachHandler) CreateOutreach(w http.ResponseWriter, r *http.Request) {
	var in models.NewOutreachInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if in.MessageSentAt.IsZero() {
		in.MessageSentAt = time.Now().UTC()
	}

	o := &models.OutreachContact{
		Name:          in.Name,
		LinkedInURL:   in.LinkedInURL,
		TargetRole:    in.TargetRole,
		RecruiterID:   in.RecruiterID,
		MessageSentAt: in.MessageSentAt,
	}
	if err := h.Store.Create(o); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

// GetOutreach handles GET /api/outreach/{id}
func (h *OutreachHandler) GetOutreach(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	o, err := h.Store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "outreach contact not found")
		return
	}
	writeJSON(w, http.StatusOK, o)
}

// ListOutreach handles GET /api/outreach?recruiter_id=&status=
// Adds a needs_follow_up flag per contact: true when still "Sent" and
// no reply has come in after 3+ days — the practical substitute for a
// LinkedIn-provided read/reply status.
func (h *OutreachHandler) ListOutreach(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list := h.Store.List(q.Get("recruiter_id"), q.Get("status"))

	type withFollowUp struct {
		*models.OutreachContact
		NeedsFollowUp bool `json:"needs_follow_up"`
	}
	out := make([]withFollowUp, 0, len(list))
	cutoff := time.Now().UTC().AddDate(0, 0, -3)
	for _, o := range list {
		needsFollowUp := o.Status == models.OutreachSent && o.MessageSentAt.Before(cutoff)
		out = append(out, withFollowUp{OutreachContact: o, NeedsFollowUp: needsFollowUp})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"count": len(out), "outreach": out})
}

// UpdateOutreachStatus handles PATCH /api/outreach/{id}/status
// Body: {"status": "Replied"}
func (h *OutreachHandler) UpdateOutreachStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	status := models.OutreachStatus(body.Status)
	if !models.IsValidOutreachStatus(status) {
		writeError(w, http.StatusBadRequest, "unknown status: "+body.Status)
		return
	}

	o, err := h.Store.UpdateStatus(id, status)
	if err != nil {
		writeError(w, http.StatusNotFound, "outreach contact not found")
		return
	}
	writeJSON(w, http.StatusOK, o)
}

// AddOutreachNote handles POST /api/outreach/{id}/notes
// Body: {"text": "..."}
func (h *OutreachHandler) AddOutreachNote(w http.ResponseWriter, r *http.Request) {
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

	o, err := h.Store.AddNote(id, authorID, body.Text)
	if err != nil {
		writeError(w, http.StatusNotFound, "outreach contact not found")
		return
	}
	writeJSON(w, http.StatusOK, o)
}
