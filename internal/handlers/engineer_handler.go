package handlers

import (
	"encoding/json"
	"net/http"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type EngineerHandler struct {
	Store store.EngineerStore
}

func NewEngineerHandler(s store.EngineerStore) *EngineerHandler {
	return &EngineerHandler{Store: s}
}

// CreateEngineer handles POST /api/engineers
func (h *EngineerHandler) CreateEngineer(w http.ResponseWriter, r *http.Request) {
	var in models.NewEngineerInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.Name == "" || in.Email == "" {
		writeError(w, http.StatusBadRequest, "name and email are required")
		return
	}

	e := &models.Engineer{
		Name:             in.Name,
		Email:            in.Email,
		Phone:            in.Phone,
		Skills:           in.Skills,
		Location:         in.Location,
		AreaCoverage:     in.AreaCoverage,
		HourlyRate:       in.HourlyRate,
		HalfDayRate:      in.HalfDayRate,
		DayRate:          in.DayRate,
		Currency:         in.Currency,
		TravelCost:       in.TravelCost,
		ResumeURL:        in.ResumeURL,
		Documents:        in.Documents,
		ApprovedProjects: in.ApprovedProjects,
	}
	if err := h.Store.Create(e); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// GetEngineer handles GET /api/engineers/{id}
func (h *EngineerHandler) GetEngineer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	e, err := h.Store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "engineer not found")
		return
	}
	writeJSON(w, http.StatusOK, e)
}

// SearchEngineers handles GET /api/engineers?location=&skill=&project=&available_only=true
// This backs the SOW's "Service Desk Search Feature": filters by location,
// skill, availability and project, with the minimum-cost engineer first.
func (h *EngineerHandler) SearchEngineers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := store.EngineerFilter{
		Location:      q.Get("location"),
		Skill:         q.Get("skill"),
		Project:       q.Get("project"),
		OnlyAvailable: q.Get("available_only") == "true",
	}
	engineers := h.Store.Search(filter)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":     len(engineers),
		"engineers": engineers,
	})
}

// SetAvailability handles PATCH /api/engineers/{id}/availability
// Body: {"available": false}
func (h *EngineerHandler) SetAvailability(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		Available bool `json:"available"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	e, err := h.Store.SetAvailability(id, body.Available)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "engineer not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, e)
}
