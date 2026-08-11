package handlers

import (
	"encoding/json"
	"net/http"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type AssignmentHandler struct {
	Store     store.AssignmentStore
	Engineers store.EngineerStore
}

func NewAssignmentHandler(s store.AssignmentStore, e store.EngineerStore) *AssignmentHandler {
	return &AssignmentHandler{Store: s, Engineers: e}
}

// CreateAssignment handles POST /api/engineers/{id}/assignments
// This is the SOW's "re-hire" flow: attach an already-active engineer to
// another project, with its own rate and allowances, while any existing
// assignments stay active — the engineer can run on several projects
// at once.
func (h *AssignmentHandler) CreateAssignment(w http.ResponseWriter, r *http.Request) {
	engineerID := r.PathValue("id")
	if _, err := h.Engineers.Get(engineerID); err != nil {
		writeError(w, http.StatusNotFound, "engineer not found")
		return
	}

	var in models.NewAssignmentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.ProjectName == "" || in.ClientName == "" || in.StartDate == "" {
		writeError(w, http.StatusBadRequest, "project_name, client_name and start_date are required")
		return
	}

	a := &models.EngineerAssignment{
		EngineerID:      engineerID,
		ProjectName:     in.ProjectName,
		ClientName:      in.ClientName,
		HourlyRate:      in.HourlyRate,
		DayRate:         in.DayRate,
		Currency:        in.Currency,
		TravelAllowance: in.TravelAllowance,
		ToolsAllowance:  in.ToolsAllowance,
		StartDate:       in.StartDate,
		EndDate:         in.EndDate,
	}
	if err := h.Store.Create(a); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

// ListAssignments handles GET /api/engineers/{id}/assignments
func (h *AssignmentHandler) ListAssignments(w http.ResponseWriter, r *http.Request) {
	engineerID := r.PathValue("id")
	list := h.Store.ListByEngineer(engineerID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"count": len(list), "assignments": list})
}

// EndAssignment handles PATCH /api/assignments/{id}/end
func (h *AssignmentHandler) EndAssignment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, err := h.Store.End(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "assignment not found")
		return
	}
	writeJSON(w, http.StatusOK, a)
}
