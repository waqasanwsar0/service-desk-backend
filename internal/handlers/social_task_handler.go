package handlers

import (
	"encoding/json"
	"net/http"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type SocialTaskHandler struct {
	Store store.SocialTaskStore
}

func NewSocialTaskHandler(s store.SocialTaskStore) *SocialTaskHandler {
	return &SocialTaskHandler{Store: s}
}

// CreateSocialTask handles POST /api/social-tasks
func (h *SocialTaskHandler) CreateSocialTask(w http.ResponseWriter, r *http.Request) {
	var in models.NewSocialTaskInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.Title == "" || in.Platform == "" {
		writeError(w, http.StatusBadRequest, "title and platform are required")
		return
	}
	t := &models.SocialMediaTask{
		Title:       in.Title,
		Platform:    in.Platform,
		Description: in.Description,
		OwnerID:     in.OwnerID,
		DueDate:     in.DueDate,
		AssetURL:    in.AssetURL,
	}
	if err := h.Store.Create(t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// GetSocialTask handles GET /api/social-tasks/{id}
func (h *SocialTaskHandler) GetSocialTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.Store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// ListSocialTasks handles GET /api/social-tasks?platform=&status=&owner_id=
func (h *SocialTaskHandler) ListSocialTasks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list := h.Store.List(q.Get("platform"), q.Get("status"), q.Get("owner_id"))
	writeJSON(w, http.StatusOK, map[string]interface{}{"count": len(list), "tasks": list})
}

// UpdateSocialTaskStatus handles PATCH /api/social-tasks/{id}/status
// Body: {"status": "Approved"} — Draft -> In Review -> Approved -> Posted
// is the intended flow but any status can be set directly (a simple
// content approval workflow, not a strict state machine).
func (h *SocialTaskHandler) UpdateSocialTaskStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	status := models.SocialTaskStatus(body.Status)
	if !models.IsValidSocialTaskStatus(status) {
		writeError(w, http.StatusBadRequest, "unknown status: "+body.Status)
		return
	}
	t, err := h.Store.UpdateStatus(id, status)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// AddSocialTaskNote handles POST /api/social-tasks/{id}/notes
func (h *SocialTaskHandler) AddSocialTaskNote(w http.ResponseWriter, r *http.Request) {
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
	t, err := h.Store.AddNote(id, authorID, body.Text)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// SocialTaskDashboard handles GET /api/social-tasks/dashboard
// Status counts by platform — a quick overview of what's in the
// pipeline per channel.
func (h *SocialTaskHandler) SocialTaskDashboard(w http.ResponseWriter, r *http.Request) {
	all := h.Store.List("", "", "")

	byPlatform := map[string]map[string]int{}
	for _, t := range all {
		if byPlatform[t.Platform] == nil {
			byPlatform[t.Platform] = map[string]int{}
		}
		byPlatform[t.Platform][string(t.Status)]++
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":       len(all),
		"by_platform": byPlatform,
	})
}
