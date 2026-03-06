package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go2engle/gantry/internal/api/middleware"
	"github.com/go2engle/gantry/internal/auth"
	"github.com/go2engle/gantry/internal/db"
	"github.com/go2engle/gantry/internal/entity"
)

type createAPIKeyRequest struct {
	Name string `json:"name"`
	Role string `json:"role,omitempty"`
}

type createAPIKeyResponse struct {
	*db.APIKey
	Key string `json:"key"` // raw key shown once
}

// ListAPIKeys handles GET /auth/apikeys.
// Returns all API keys owned by the authenticated user (hashes not included).
func (h *Handlers) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	keys, err := h.DB.ListAPIKeys(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list api keys")
		return
	}
	if keys == nil {
		keys = []*db.APIKey{}
	}
	writeJSON(w, http.StatusOK, keys)
}

// CreateAPIKey handles POST /auth/apikeys.
// Creates a new API key for the authenticated user and returns it once.
func (h *Handlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req createAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Default role to the caller's role; cap it at the caller's own role level.
	role := req.Role
	if role == "" {
		role = claims.Role
	}

	rawKey, keyHash, prefix, err := auth.GenerateAPIKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate api key")
		return
	}

	key := &db.APIKey{
		UserID: claims.UserID,
		Name:   req.Name,
		Prefix: prefix,
		Role:   role,
	}

	if err := h.DB.CreateAPIKey(r.Context(), key, keyHash); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create api key")
		return
	}

	writeJSON(w, http.StatusCreated, createAPIKeyResponse{APIKey: key, Key: rawKey})
}

// RevokeAPIKey handles DELETE /auth/apikeys/{id}.
// Deletes an API key owned by the authenticated user.
func (h *Handlers) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.DB.DeleteAPIKey(r.Context(), id, claims.UserID); err != nil {
		if errors.Is(err, entity.ErrEntityNotFound) {
			writeError(w, http.StatusNotFound, "api key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to revoke api key")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
