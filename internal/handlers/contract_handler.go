package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type ContractHandler struct {
	Store store.ContractStore
}

func NewContractHandler(s store.ContractStore) *ContractHandler {
	return &ContractHandler{Store: s}
}

// CreateContract handles POST /api/contracts
func (h *ContractHandler) CreateContract(w http.ResponseWriter, r *http.Request) {
	var in models.NewContractInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.ClientName == "" || in.StartDate == "" || in.ExpiryDate == "" {
		writeError(w, http.StatusBadRequest, "client_name, start_date and expiry_date are required")
		return
	}
	c := &models.Contract{
		ClientName:   in.ClientName,
		EntityID:     in.EntityID,
		ProjectScope: in.ProjectScope,
		BillingTerms: in.BillingTerms,
		StartDate:    in.StartDate,
		ExpiryDate:   in.ExpiryDate,
		AutoRenew:    in.AutoRenew,
	}
	if err := h.Store.Create(c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// GetContract handles GET /api/contracts/{id}
func (h *ContractHandler) GetContract(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := h.Store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "contract not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// ListContracts handles GET /api/contracts?client_name=&expiring_within_days=
func (h *ContractHandler) ListContracts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if days := q.Get("expiring_within_days"); days != "" {
		n, err := strconv.Atoi(days)
		if err != nil {
			writeError(w, http.StatusBadRequest, "expiring_within_days must be a number")
			return
		}
		list := h.Store.ExpiringWithin(n)
		writeJSON(w, http.StatusOK, map[string]interface{}{"count": len(list), "contracts": list})
		return
	}
	list := h.Store.List(q.Get("client_name"))
	writeJSON(w, http.StatusOK, map[string]interface{}{"count": len(list), "contracts": list})
}
