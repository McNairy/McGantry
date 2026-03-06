package plugins

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed bundled/registry.json
var bundledRegistryJSON []byte

// BundledRegistry returns the list of plugins from the embedded registry JSON.
// This provides an offline catalog that ships with the binary.
func BundledRegistry() ([]RegistryEntry, error) {
	var entries []RegistryEntry
	if err := json.Unmarshal(bundledRegistryJSON, &entries); err != nil {
		return nil, fmt.Errorf("parse bundled registry: %w", err)
	}
	return entries, nil
}

// FindInRegistry returns the registry entry with the given name, or nil if not found.
func FindInRegistry(name string) (*RegistryEntry, error) {
	entries, err := BundledRegistry()
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].Name == name {
			return &entries[i], nil
		}
	}
	return nil, nil
}
