package handlers

import (
	"encoding/json"
	"net/http"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type ProjectHandler struct {
	Store store.ProjectStore
}

func NewProjectHandler(s store.ProjectStore) *ProjectHandler {
	return &ProjectHandler{Store: s}
}

// CreateProject handles POST /api/projects
func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var in models.NewProjectInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.Name == "" || in.ClientName == "" {
		writeError(w, http.StatusBadRequest, "name and client_name are required")
		return
	}
	if in.Type == "" {
		in.Type = models.ProjectTypeDispatch
	}

	p := &models.Project{Name: in.Name, ClientName: in.ClientName, Country: in.Country, City: in.City, Type: in.Type}
	if err := h.Store.Create(p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// GetProject handles GET /api/projects/{id}
func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := h.Store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// ListProjects handles GET /api/projects?country=&city=
func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list := h.Store.List(q.Get("country"), q.Get("city"))
	writeJSON(w, http.StatusOK, map[string]interface{}{"count": len(list), "projects": list})
}
