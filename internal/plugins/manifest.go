// Package plugins provides types and utilities for the Gantry plugin system.
package plugins

// Manifest describes a plugin's metadata, capabilities, and configuration schema.
type Manifest struct {
	Name        string            `json:"name"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Author      string            `json:"author"`
	Category    string            `json:"category"` // integration | widget | entity-kind | action-type | auth-provider
	Source      string            `json:"source,omitempty"`
	BundleURL   string            `json:"bundleUrl,omitempty"`
	IconURL     string            `json:"iconUrl,omitempty"`
	Homepage    string            `json:"homepage,omitempty"`
	// ConfigSchema is a JSON Schema object describing the plugin's configuration fields.
	ConfigSchema map[string]any `json:"configSchema,omitempty"`
	// EntityPanels lists the entity kinds this plugin contributes panels to.
	EntityPanels []string `json:"entityPanels,omitempty"`
	// ActionTypes lists action types this plugin contributes.
	ActionTypes  []string             `json:"actionTypes,omitempty"`
	Requirements []PluginRequirement  `json:"requirements,omitempty"`
}

// PluginRequirement describes a prerequisite for a plugin.
type PluginRequirement struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Optional    bool   `json:"optional,omitempty"`
}

// Plugin represents an installed plugin record combining a manifest with runtime state.
type Plugin struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Enabled     bool           `json:"enabled"`
	Config      map[string]any `json:"config,omitempty"`
	Manifest    *Manifest      `json:"manifest"`
	InstalledAt string         `json:"installedAt"`
	UpdatedAt   string         `json:"updatedAt"`
}

// RegistryEntry is a lightweight summary used in the marketplace browser.
type RegistryEntry struct {
	Name            string         `json:"name"`
	Title           string         `json:"title"`
	Description     string         `json:"description"`
	LongDescription string         `json:"longDescription,omitempty"`
	Features        []string       `json:"features,omitempty"`
	Version         string         `json:"version"`
	Author          string         `json:"author"`
	Category        string         `json:"category"`
	IconURL         string         `json:"iconUrl,omitempty"`
	Homepage        string         `json:"homepage,omitempty"`
	ConfigSchema    map[string]any `json:"configSchema,omitempty"`
	EntityPanels    []string       `json:"entityPanels,omitempty"`
	ActionTypes     []string            `json:"actionTypes,omitempty"`
	Requirements    []PluginRequirement `json:"requirements,omitempty"`
	Source          string              `json:"source,omitempty"`
}
