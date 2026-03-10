package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go2engle/gantry/internal/api/middleware"
)

// RecordHistory handles POST /api/v1/history.
// Records that the authenticated user viewed a specific entity.
// Body: { "kind": "...", "name": "...", "namespace": "..." }
func (h *Handlers) RecordHistory(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		Kind      string `json:"kind"`
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Kind == "" || body.Name == "" {
		writeError(w, http.StatusBadRequest, "kind and name are required")
		return
	}
	if body.Namespace == "" {
		body.Namespace = "default"
	}

	if err := h.DB.RecordEntityView(r.Context(), claims.Username, body.Kind, body.Name, body.Namespace); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record history")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetHistory handles GET /api/v1/history.
// Returns the authenticated user's recently browsed entities.
// Query param: limit (default 10, max 20).
func (h *Handlers) GetHistory(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 10
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 && l <= 20 {
			limit = l
		}
	}

	entries, err := h.DB.GetUserHistory(r.Context(), claims.Username, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get history")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}
