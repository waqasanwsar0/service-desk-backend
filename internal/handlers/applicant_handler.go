package handlers

import (
	"encoding/json"
	"net/http"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type ApplicantHandler struct {
	Store store.ApplicantStore
}

func NewApplicantHandler(s store.ApplicantStore) *ApplicantHandler {
	return &ApplicantHandler{Store: s}
}

// CreateApplicant handles POST /api/applicants
func (h *ApplicantHandler) CreateApplicant(w http.ResponseWriter, r *http.Request) {
	var in models.NewApplicantInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.Name == "" || in.Email == "" || in.JobTitle == "" {
		writeError(w, http.StatusBadRequest, "name, email and job_title are required")
		return
	}

	a := &models.Applicant{
		Name:        in.Name,
		Email:       in.Email,
		Phone:       in.Phone,
		JobTitle:    in.JobTitle,
		ResumeURL:   in.ResumeURL,
		RecruiterID: in.RecruiterID,
	}
	if err := h.Store.Create(a); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

// GetApplicant handles GET /api/applicants/{id}
func (h *ApplicantHandler) GetApplicant(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, err := h.Store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "applicant not found")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// ListApplicants handles GET /api/applicants?job_title=&stage=&recruiter_id=
func (h *ApplicantHandler) ListApplicants(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list := h.Store.List(q.Get("job_title"), q.Get("stage"), q.Get("recruiter_id"))
	writeJSON(w, http.StatusOK, map[string]interface{}{"count": len(list), "applicants": list})
}

// MoveStage handles PATCH /api/applicants/{id}/stage
// Body: {"stage": "Interview"}
func (h *ApplicantHandler) MoveStage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		Stage string `json:"stage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	newStage := models.ApplicantStage(body.Stage)
	if !models.IsValidStage(newStage) {
		writeError(w, http.StatusBadRequest, "unknown stage: "+body.Stage)
		return
	}

	a, err := h.Store.MoveStage(id, newStage)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "applicant not found")
			return
		}
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// RecruitmentDashboard handles GET /api/applicants/dashboard
// SOW: "Project-Wise Recruitment Status Dashboard" — real-time pipeline
// counts per project (job title), visible to both Recruitment and
// Service Delivery teams.
func (h *ApplicantHandler) RecruitmentDashboard(w http.ResponseWriter, r *http.Request) {
	all := h.Store.List("", "", "")

	byProject := map[string]map[string]int{}
	for _, a := range all {
		if byProject[a.JobTitle] == nil {
			byProject[a.JobTitle] = map[string]int{}
		}
		byProject[a.JobTitle][string(a.Stage)]++
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"by_project":     byProject,
		"total_open":     countNotTerminal(all),
		"total_hired":    countByStage(all, models.StageHired),
		"total_rejected": countByStage(all, models.StageRejected),
	})
}

// RecruiterPerformance handles GET /api/recruiters/performance
// SOW: "Recruiter Performance Tracking" — how many candidates each
// recruiter is handling and their hire rate.
func (h *ApplicantHandler) RecruiterPerformance(w http.ResponseWriter, r *http.Request) {
	all := h.Store.List("", "", "")

	type perf struct {
		RecruiterID  string  `json:"recruiter_id"`
		TotalHandled int     `json:"total_handled"`
		Hired        int     `json:"hired"`
		Rejected     int     `json:"rejected"`
		InProgress   int     `json:"in_progress"`
		HireRate     float64 `json:"hire_rate"`
	}

	byRecruiter := map[string]*perf{}
	for _, a := range all {
		if a.RecruiterID == "" {
			continue
		}
		p, ok := byRecruiter[a.RecruiterID]
		if !ok {
			p = &perf{RecruiterID: a.RecruiterID}
			byRecruiter[a.RecruiterID] = p
		}
		p.TotalHandled++
		switch a.Stage {
		case models.StageHired:
			p.Hired++
		case models.StageRejected:
			p.Rejected++
		default:
			p.InProgress++
		}
	}
	result := make([]*perf, 0, len(byRecruiter))
	for _, p := range byRecruiter {
		if p.TotalHandled > 0 {
			p.HireRate = round2f(float64(p.Hired) / float64(p.TotalHandled) * 100)
		}
		result = append(result, p)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"recruiters": result})
}

func countNotTerminal(applicants []*models.Applicant) int {
	n := 0
	for _, a := range applicants {
		if a.Stage != models.StageHired && a.Stage != models.StageRejected {
			n++
		}
	}
	return n
}

func countByStage(applicants []*models.Applicant, stage models.ApplicantStage) int {
	n := 0
	for _, a := range applicants {
		if a.Stage == stage {
			n++
		}
	}
	return n
}

// This backs the SOW's "Chat with Recruiter" feature — a running,
// timestamped thread of notes/messages tied to the applicant.
// Body: {"text": "..."}
func (h *ApplicantHandler) AddNote(w http.ResponseWriter, r *http.Request) {
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

	a, err := h.Store.AddNote(id, authorID, body.Text)
	if err != nil {
		writeError(w, http.StatusNotFound, "applicant not found")
		return
	}
	writeJSON(w, http.StatusOK, a)
}
