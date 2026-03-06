package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gantrydev/gantry/internal/plugins"
)

// ListPlugins returns all plugins: registry entries merged with installed state.
func (h *Handlers) ListPlugins(w http.ResponseWriter, r *http.Request) {
	// Load bundled registry.
	registry, err := plugins.BundledRegistry()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load plugin registry: "+err.Error())
		return
	}

	// Load installed plugins from DB.
	installed, err := h.DB.ListPlugins(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build a map of installed plugins by name for quick lookup.
	installedMap := make(map[string]*plugins.Plugin, len(installed))
	for i := range installed {
		installedMap[installed[i].Name] = &installed[i]
	}

	type pluginListItem struct {
		plugins.RegistryEntry
		Installed   bool   `json:"installed"`
		Enabled     bool   `json:"enabled"`
		InstalledAt string `json:"installedAt,omitempty"`
	}

	items := make([]pluginListItem, 0, len(registry))
	for _, entry := range registry {
		item := pluginListItem{RegistryEntry: entry}
		if p, ok := installedMap[entry.Name]; ok {
			item.Installed = true
			item.Enabled = p.Enabled
			item.InstalledAt = p.InstalledAt
		}
		items = append(items, item)
	}

	// Also append any installed plugins not in the bundled registry (e.g. community).
	for _, p := range installed {
		found := false
		for _, entry := range registry {
			if entry.Name == p.Name {
				found = true
				break
			}
		}
		if !found && p.Manifest != nil {
			items = append(items, pluginListItem{
				RegistryEntry: plugins.RegistryEntry{
					Name:        p.Manifest.Name,
					Title:       p.Manifest.Title,
					Description: p.Manifest.Description,
					Version:     p.Manifest.Version,
					Author:      p.Manifest.Author,
					Category:    p.Manifest.Category,
				},
				Installed:   true,
				Enabled:     p.Enabled,
				InstalledAt: p.InstalledAt,
			})
		}
	}

	writeJSON(w, http.StatusOK, items)
}

// GetPlugin returns full detail for a single installed plugin.
func (h *Handlers) GetPlugin(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	p, err := h.DB.GetPlugin(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "plugin not installed")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// InstallPlugin "installs" a plugin from the bundled registry into the DB.
func (h *Handlers) InstallPlugin(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// Look up in bundled registry.
	entry, err := plugins.FindInRegistry(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if entry == nil {
		writeError(w, http.StatusNotFound, "plugin not found in registry")
		return
	}

	// Build a manifest from the registry entry.
	manifest := &plugins.Manifest{
		Name:        entry.Name,
		Title:       entry.Title,
		Description: entry.Description,
		Version:     entry.Version,
		Author:      entry.Author,
		Category:    entry.Category,
		Homepage:    entry.Homepage,
	}

	p := &plugins.Plugin{
		ID:       newShortID(),
		Name:     entry.Name,
		Version:  entry.Version,
		Enabled:  false,
		Manifest: manifest,
	}

	if err := h.DB.UpsertPlugin(r.Context(), p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// UninstallPlugin removes an installed plugin.
func (h *Handlers) UninstallPlugin(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := h.DB.DeletePlugin(r.Context(), name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// EnablePlugin enables or disables a plugin.
func (h *Handlers) EnablePlugin(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.DB.UpdatePluginEnabled(r.Context(), name, body.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetPluginConfig returns the config schema and current values for a plugin.
func (h *Handlers) GetPluginConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	p, err := h.DB.GetPlugin(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "plugin not installed")
		return
	}

	var schema map[string]any
	if p.Manifest != nil {
		schema = p.Manifest.ConfigSchema
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"schema": schema,
		"values": p.Config,
	})
}

// UpdatePluginConfig saves plugin configuration.
func (h *Handlers) UpdatePluginConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var config map[string]any
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.DB.UpdatePluginConfig(r.Context(), name, config); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// newShortID generates a short random ID for plugin records.
func newShortID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
