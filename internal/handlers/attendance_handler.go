package handlers

import (
	"encoding/json"
	"net/http"

	"servicedesk/internal/store"
)

type AttendanceHandler struct {
	Store store.AttendanceStore
}

func NewAttendanceHandler(s store.AttendanceStore) *AttendanceHandler {
	return &AttendanceHandler{Store: s}
}

// CheckIn handles POST /api/attendance/checkin
// This is the "I am ON-SITE" button from the SOW. Body: {"engineer_id": "...", "location": "..."}
//
// TODO: once User accounts are linked to Engineer records, derive
// engineer_id from the authenticated caller (ClaimsFromContext) instead of
// trusting the request body.
func (h *AttendanceHandler) CheckIn(w http.ResponseWriter, r *http.Request) {
	var body struct {
		EngineerID string `json:"engineer_id"`
		Location   string `json:"location"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if body.EngineerID == "" {
		writeError(w, http.StatusBadRequest, "engineer_id is required")
		return
	}

	rec, err := h.Store.CheckIn(body.EngineerID, body.Location)
	if err != nil {
		if err == store.ErrAlreadyCheckedIn {
			writeError(w, http.StatusConflict, "already checked in today")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

// CheckOut handles POST /api/attendance/checkout
// This is the "I am OFF-SITE" end-of-day action — closes out today's
// check-in with a check-out time so duration can be shown.
// Body: {"engineer_id": "..."}
func (h *AttendanceHandler) CheckOut(w http.ResponseWriter, r *http.Request) {
	var body struct {
		EngineerID string `json:"engineer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if body.EngineerID == "" {
		writeError(w, http.StatusBadRequest, "engineer_id is required")
		return
	}

	rec, err := h.Store.CheckOut(body.EngineerID)
	if err != nil {
		if err == store.ErrNotCheckedInYet {
			writeError(w, http.StatusConflict, "no check-in found for today — check in first")
			return
		}
		if err == store.ErrAlreadyCheckedOut {
			writeError(w, http.StatusConflict, "already checked out today")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

// RequestLeave handles POST /api/attendance/leave-request
// Body: {"engineer_id": "...", "from_date": "2026-08-10", "to_date": "2026-08-12", "reason": "..."}
func (h *AttendanceHandler) RequestLeave(w http.ResponseWriter, r *http.Request) {
	var body struct {
		EngineerID string `json:"engineer_id"`
		FromDate   string `json:"from_date"`
		ToDate     string `json:"to_date"`
		Reason     string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if body.EngineerID == "" || body.FromDate == "" || body.ToDate == "" {
		writeError(w, http.StatusBadRequest, "engineer_id, from_date and to_date are required")
		return
	}

	lr, err := h.Store.RequestLeave(body.EngineerID, body.FromDate, body.ToDate, body.Reason)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, lr)
}

// DecideLeave handles PATCH /api/attendance/leave-request/{id}/decision
// Body: {"decision": "approved"} or {"decision": "rejected"}
func (h *AttendanceHandler) DecideLeave(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		Decision string `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if body.Decision != "approved" && body.Decision != "rejected" {
		writeError(w, http.StatusBadRequest, `decision must be "approved" or "rejected"`)
		return
	}

	claims := ClaimsFromContext(r.Context())
	decidedBy := ""
	if claims != nil {
		decidedBy = claims.UserID
	}

	lr, err := h.Store.DecideLeave(id, decidedBy, body.Decision == "approved")
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "leave request not found")
			return
		}
		if err == store.ErrLeaveNotPending {
			writeError(w, http.StatusConflict, "leave request already decided")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, lr)
}

// ListAttendance handles GET /api/attendance?engineer_id=
func (h *AttendanceHandler) ListAttendance(w http.ResponseWriter, r *http.Request) {
	engineerID := r.URL.Query().Get("engineer_id")
	records := h.Store.List(engineerID)
	leaves := h.Store.ListLeaveRequests(engineerID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"attendance":     records,
		"leave_requests": leaves,
	})
}
