package entity

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed schemas
var schemasFS embed.FS

// SchemaValidator validates entities against their kind's JSON Schema.
// Schemas are loaded from embedded JSON files at initialization time.
type SchemaValidator struct {
	// compiled holds compiled JSON Schema validators keyed by lowercase kind name.
	compiled map[string]*jsonschema.Schema

	// raw holds the raw JSON bytes of each schema keyed by lowercase kind name.
	raw map[string]json.RawMessage
}

// NewSchemaValidator creates a SchemaValidator by loading and compiling all
// JSON Schema files from the embedded schemas directory.
//
// Schema files are expected to be named <kind>.json (lowercase) within the
// schemas/ directory. If schemasDir is non-empty, it is used as the
// subdirectory within the embedded filesystem; otherwise "schemas" is used.
func NewSchemaValidator(schemasDir string) (*SchemaValidator, error) {
	if schemasDir == "" {
		schemasDir = "schemas"
	}

	entries, err := schemasFS.ReadDir(schemasDir)
	if err != nil {
		return nil, fmt.Errorf("reading schemas directory %q: %w", schemasDir, err)
	}

	v := &SchemaValidator{
		compiled: make(map[string]*jsonschema.Schema),
		raw:      make(map[string]json.RawMessage),
	}

	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft2020

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		kindName := strings.TrimSuffix(entry.Name(), ".json")
		filePath := schemasDir + "/" + entry.Name()

		data, err := schemasFS.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("reading schema file %q: %w", filePath, err)
		}

		// Validate that the file contains valid JSON.
		if !json.Valid(data) {
			return nil, fmt.Errorf("schema file %q contains invalid JSON", filePath)
		}

		// Store raw schema bytes.
		v.raw[strings.ToLower(kindName)] = json.RawMessage(data)

		// Register the schema resource with the compiler.
		resourceURI := "gantry://schemas/" + entry.Name()
		if err := compiler.AddResource(resourceURI, strings.NewReader(string(data))); err != nil {
			return nil, fmt.Errorf("adding schema resource %q: %w", resourceURI, err)
		}

		compiled, err := compiler.Compile(resourceURI)
		if err != nil {
			return nil, fmt.Errorf("compiling schema for kind %q: %w", kindName, err)
		}

		v.compiled[strings.ToLower(kindName)] = compiled
	}

	return v, nil
}

// Validate validates an entity against its kind's JSON Schema.
//
// The method first runs base entity validation (required fields), then
// validates the spec portion against the kind's schema. If no schema is
// registered for the entity's kind, only the base validation is performed.
func (v *SchemaValidator) Validate(e *Entity) error {
	// Always validate the base entity structure.
	if err := e.Validate(); err != nil {
		return err
	}

	// Look up a compiled schema for this kind.
	schema, ok := v.compiled[strings.ToLower(e.Kind)]
	if !ok {
		// No schema registered for this kind; base validation is sufficient.
		return nil
	}

	// If spec is nil or empty, validate an empty object against the schema.
	spec := e.Spec
	if spec == nil {
		spec = make(map[string]any)
	}

	// The jsonschema library validates against interface{} values directly.
	if err := schema.Validate(spec); err != nil {
		validationErr, ok := err.(*jsonschema.ValidationError)
		if ok {
			return fmt.Errorf("spec validation failed for kind %q: %s", e.Kind, formatValidationError(validationErr))
		}
		return fmt.Errorf("spec validation failed for kind %q: %w", e.Kind, err)
	}

	return nil
}

// GetSchema returns the raw JSON Schema for the given kind (case-insensitive).
// It returns an error if no schema is registered for the kind.
func (v *SchemaValidator) GetSchema(kind string) (json.RawMessage, error) {
	raw, ok := v.raw[strings.ToLower(kind)]
	if !ok {
		return nil, fmt.Errorf("no schema found for kind %q", kind)
	}
	return raw, nil
}

// ListSchemas returns all registered schemas as a map from lowercase kind name
// to raw JSON Schema bytes.
func (v *SchemaValidator) ListSchemas() map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(v.raw))
	for k, raw := range v.raw {
		result[k] = raw
	}
	return result
}

// HasSchema reports whether a schema is registered for the given kind (case-insensitive).
func (v *SchemaValidator) HasSchema(kind string) bool {
	_, ok := v.compiled[strings.ToLower(kind)]
	return ok
}

// formatValidationError produces a human-readable summary of a JSON Schema
// validation error tree.
func formatValidationError(ve *jsonschema.ValidationError) string {
	var msgs []string
	collectErrors(ve, "", &msgs)
	if len(msgs) == 0 {
		return ve.Error()
	}
	return strings.Join(msgs, "; ")
}

// collectErrors recursively walks the validation error tree and appends
// user-friendly messages to msgs.
func collectErrors(ve *jsonschema.ValidationError, path string, msgs *[]string) {
	currentPath := path
	if ve.InstanceLocation != "" {
		currentPath = ve.InstanceLocation
	}

	if len(ve.Causes) == 0 {
		// Leaf error: this contains the actual validation message.
		msg := ve.Message
		if currentPath != "" {
			msg = currentPath + ": " + msg
		}
		*msgs = append(*msgs, msg)
		return
	}

	for _, cause := range ve.Causes {
		collectErrors(cause, currentPath, msgs)
	}
}
