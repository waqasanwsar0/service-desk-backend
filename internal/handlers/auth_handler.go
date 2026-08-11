package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"servicedesk/internal/auth"
	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type AuthHandler struct {
	Users store.UserStore
}

func NewAuthHandler(u store.UserStore) *AuthHandler {
	return &AuthHandler{Users: u}
}

// Register handles POST /api/auth/register
// In production this should itself be protected (only Admin can create
// users) — left open here so the module is testable standalone. Wire
// RequireRole(models.RoleAdmin) around it once an initial admin exists.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var in models.NewUserInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if in.Name == "" || in.Email == "" || in.Password == "" {
		writeError(w, http.StatusBadRequest, "name, email and password are required")
		return
	}
	if !models.IsValidRole(in.Role) {
		writeError(w, http.StatusBadRequest, "invalid role: "+string(in.Role))
		return
	}
	if len(in.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	u := &models.User{
		Name:         in.Name,
		Email:        in.Email,
		Role:         in.Role,
		PasswordHash: hash,
	}
	if err := h.Users.Create(u); err != nil {
		if err == store.ErrUserExists {
			writeError(w, http.StatusConflict, "a user with this email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, u)
}

// Login handles POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	u, err := h.Users.GetByEmail(in.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if !auth.VerifyPassword(in.Password, u.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := auth.GenerateToken(u.ID, u.Email, string(u.Role), 24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":      token,
		"expires_in": "24h",
		"user":       u,
	})
}

// Me handles GET /api/auth/me — returns the caller's own profile.
// Requires RequireAuth middleware to have populated the context.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	u, err := h.Users.GetByID(claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, u)
}
