package handlers

import (
	"encoding/json"
	"net/http"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type SalaryHandler struct {
	Store store.SalaryStore
}

func NewSalaryHandler(s store.SalaryStore) *SalaryHandler {
	return &SalaryHandler{Store: s}
}

// CreateSalary handles POST /api/salaries
func (h *SalaryHandler) CreateSalary(w http.ResponseWriter, r *http.Request) {
	var in models.NewSalaryInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.EmployeeName == "" || in.PayPeriod == "" || in.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "employee_name, pay_period and a positive amount are required")
		return
	}
	rec := &models.SalaryRecord{
		EmployeeName: in.EmployeeName,
		UserID:       in.UserID,
		PayPeriod:    in.PayPeriod,
		Amount:       in.Amount,
		Currency:     in.Currency,
		Notes:        in.Notes,
	}
	if err := h.Store.Create(rec); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

// ListSalaries handles GET /api/salaries?pay_period=
func (h *SalaryHandler) ListSalaries(w http.ResponseWriter, r *http.Request) {
	list := h.Store.List(r.URL.Query().Get("pay_period"))
	writeJSON(w, http.StatusOK, map[string]interface{}{"count": len(list), "salaries": list})
}

// MarkSalaryPaid handles PATCH /api/salaries/{id}/paid
func (h *SalaryHandler) MarkSalaryPaid(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rec, err := h.Store.MarkPaid(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "salary record not found")
		return
	}
	writeJSON(w, http.StatusOK, rec)
}
