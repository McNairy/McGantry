package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ListSchemas handles GET /schemas. It returns all registered JSON Schemas
// keyed by kind name.
func (h *Handlers) ListSchemas(w http.ResponseWriter, r *http.Request) {
	schemas := h.Validator.ListSchemas()
	writeJSON(w, http.StatusOK, schemas)
}

// GetSchema handles GET /schemas/{kind}. It returns the JSON Schema for a
// specific entity kind.
func (h *Handlers) GetSchema(w http.ResponseWriter, r *http.Request) {
	kind := chi.URLParam(r, "kind")

	schema, err := h.Validator.GetSchema(kind)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(schema)
}
