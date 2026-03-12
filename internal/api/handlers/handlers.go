// Package handlers implements HTTP handler functions for the Gantry API.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go2engle/gantry/internal/auth"
	"github.com/go2engle/gantry/internal/db"
	"github.com/go2engle/gantry/internal/dispatcher"
	"github.com/go2engle/gantry/internal/entity"
	"github.com/go2engle/gantry/internal/events"
	"github.com/go2engle/gantry/internal/gitops"
	"github.com/go2engle/gantry/internal/search"
)

// Handlers groups all dependencies needed by API handler functions.
type Handlers struct {
	DB         *db.DB
	Auth       *auth.Service
	Events     *events.Bus
	Validator  *entity.SchemaValidator
	SearchSvc  *search.Service
	Dispatcher *dispatcher.Manager
	GitOps     *gitops.Service
	DataDir    string // root data directory, used for GitOps repo storage
}

// writeJSON serializes v as JSON and writes it to the response with the given
// HTTP status code. The Content-Type header is set to application/json.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// At this point headers are already sent; just log internally.
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
	}
}

// writeError sends a JSON error response with the given status code and message.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
