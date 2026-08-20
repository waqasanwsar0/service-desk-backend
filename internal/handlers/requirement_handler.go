package handlers

import (
	"encoding/json"
	"net/http"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type RequirementHandler struct {
	Store store.RequirementStore
}

func NewRequirementHandler(s store.RequirementStore) *RequirementHandler {
	return &RequirementHandler{Store: s}
}

// CreateRequirement handles POST /api/requirements
// The client-portal upload path — SOW: "Client requirement upload be
// ho jae".
func (h *RequirementHandler) CreateRequirement(w http.ResponseWriter, r *http.Request) {
	var in models.NewRequirementInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.ClientName == "" || in.Title == "" {
		writeError(w, http.StatusBadRequest, "client_name and title are required")
		return
	}
	req := &models.Requirement{
		ProjectID:   in.ProjectID,
		ClientName:  in.ClientName,
		Title:       in.Title,
		Description: in.Description,
		FileURL:     in.FileURL,
		Source:      models.RequirementSourcePortal,
	}
	if err := h.Store.Create(req); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, req)
}

// ImportFromEmail handles POST /api/requirements/import/email
// Landing point for a future email-parsing service — SOW: "per email
// requirement be a jae". Same shape as the ticket email importer: this
// endpoint is ready now; the piece that watches an inbox and calls it
// (Microsoft Graph, same as ticket intake) is the same future
// integration already noted for ticket email import.
func (h *RequirementHandler) ImportFromEmail(w http.ResponseWriter, r *http.Request) {
	var in models.NewRequirementInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.ClientName == "" || in.Title == "" {
		writeError(w, http.StatusBadRequest, "client_name and title are required")
		return
	}
	req := &models.Requirement{
		ProjectID:   in.ProjectID,
		ClientName:  in.ClientName,
		Title:       in.Title,
		Description: in.Description,
		FileURL:     in.FileURL,
		Source:      models.RequirementSourceEmail,
	}
	if err := h.Store.Create(req); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, req)
}

// GetRequirement handles GET /api/requirements/{id}
func (h *RequirementHandler) GetRequirement(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	req, err := h.Store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "requirement not found")
		return
	}
	writeJSON(w, http.StatusOK, req)
}

// ListRequirements handles GET /api/requirements?project_id=&client_name=
func (h *RequirementHandler) ListRequirements(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list := h.Store.List(q.Get("project_id"), q.Get("client_name"))
	writeJSON(w, http.StatusOK, map[string]interface{}{"count": len(list), "requirements": list})
}
