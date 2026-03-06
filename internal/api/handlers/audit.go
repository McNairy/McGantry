package handlers

import (
	"net/http"
	"strconv"

	"github.com/go2engle/gantry/internal/db"
)

// ListAuditEntries handles GET /audit?limit=50&offset=0.
// It returns audit log entries ordered by most recent first.
func (h *Handlers) ListAuditEntries(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	entries, err := h.DB.ListAuditEntries(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list audit entries")
		return
	}
	if entries == nil {
		entries = []*db.AuditEntry{}
	}

	writeJSON(w, http.StatusOK, entries)
}
