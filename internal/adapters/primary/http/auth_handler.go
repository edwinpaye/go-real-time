package http

import (
	"encoding/json"
	"net/http"

	"github.com/sales-system/go-real-time/internal/core/ports"
)

// AuthHandler exposes HTTP authentication routes.
type AuthHandler struct {
	authService ports.AuthService
}

func NewAuthHandler(authService ports.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req ports.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed JSON request"})
		return
	}

	actCtx := GetAuthContext(r)
	user, err := h.authService.Register(r.Context(), req, actCtx)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req ports.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed JSON request"})
		return
	}

	actCtx := GetAuthContext(r)
	res, err := h.authService.Login(r.Context(), req, actCtx)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, res)
}

func (h *AuthHandler) Profile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	actCtx := GetAuthContext(r)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id": actCtx.UserID,
		"email":   actCtx.UserEmail,
		"role":    actCtx.UserRole,
	})
}
