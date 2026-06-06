package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// InternalUpdatePluginConfig handles PATCH /api/internal/plugins/{name}/config.
// Authenticated via X-Gantry-Internal-Token header.
// Merges the provided JSON object into the plugin's existing config without
// wiping fields that are not present in the request body.
// Intended for use by plugin binaries to write back provisioned credentials
// (e.g. clientId, oidcIssuerUrl) after auto-provisioning external services.
func (h *Handlers) InternalUpdatePluginConfig(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Gantry-Internal-Token") != h.InternalPluginToken {
		writeError(w, http.StatusUnauthorized, "invalid internal token")
		return
	}

	name := chi.URLParam(r, "name")
	ctx := r.Context()

	p, err := h.DB.GetPlugin(ctx, name)
	if err != nil || p == nil {
		writeError(w, http.StatusNotFound, "plugin not found")
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	merged := make(map[string]any)
	for k, v := range p.Config {
		merged[k] = v
	}
	for k, v := range updates {
		merged[k] = v
	}

	if err := h.DB.UpdatePluginConfig(ctx, name, merged); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update plugin config")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
