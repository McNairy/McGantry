package handlers

import (
	"net/http"
)

// GetGitOpsStatus returns the current GitOps sync status.
func (h *Handlers) GetGitOpsStatus(w http.ResponseWriter, r *http.Request) {
	p, err := h.DB.GetPlugin(r.Context(), "gitops")
	if err != nil || p == nil {
		writeError(w, http.StatusNotFound, "gitops plugin not installed")
		return
	}
	if !p.Enabled {
		writeError(w, http.StatusBadRequest, "gitops plugin not enabled")
		return
	}

	if h.GitOps == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"connected":    false,
			"lastError":    "service not initialized — check plugin configuration",
			"pendingFiles": 0,
		})
		return
	}

	writeJSON(w, http.StatusOK, h.GitOps.Status())
}

// TriggerGitOpsSync performs a full sync — pushes all entities to the Git repo.
func (h *Handlers) TriggerGitOpsSync(w http.ResponseWriter, r *http.Request) {
	p, err := h.DB.GetPlugin(r.Context(), "gitops")
	if err != nil || p == nil {
		writeError(w, http.StatusNotFound, "gitops plugin not installed")
		return
	}
	if !p.Enabled {
		writeError(w, http.StatusBadRequest, "gitops plugin not enabled")
		return
	}
	if h.GitOps == nil {
		writeError(w, http.StatusBadRequest, "gitops service not initialized")
		return
	}

	// Run full sync in a goroutine; return 202 immediately.
	go func() {
		h.GitOps.FullSync()
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "full sync started — check status for progress",
	})
}

// TriggerGitOpsPull pulls changes from the remote Git repo and reconciles with the database.
func (h *Handlers) TriggerGitOpsPull(w http.ResponseWriter, r *http.Request) {
	p, err := h.DB.GetPlugin(r.Context(), "gitops")
	if err != nil || p == nil {
		writeError(w, http.StatusNotFound, "gitops plugin not installed")
		return
	}
	if !p.Enabled {
		writeError(w, http.StatusBadRequest, "gitops plugin not enabled")
		return
	}
	if h.GitOps == nil {
		writeError(w, http.StatusBadRequest, "gitops service not initialized")
		return
	}

	// Run pull in a goroutine; return 202 immediately.
	go func() {
		h.GitOps.Pull()
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "pull started — check status for progress",
	})
}

// GetGitOpsHistory returns recent sync operations.
func (h *Handlers) GetGitOpsHistory(w http.ResponseWriter, r *http.Request) {
	p, err := h.DB.GetPlugin(r.Context(), "gitops")
	if err != nil || p == nil {
		writeError(w, http.StatusNotFound, "gitops plugin not installed")
		return
	}
	if !p.Enabled {
		writeError(w, http.StatusBadRequest, "gitops plugin not enabled")
		return
	}

	if h.GitOps == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	writeJSON(w, http.StatusOK, h.GitOps.History())
}

// GetGitOpsFiles lists all entity files tracked in the Git repo.
func (h *Handlers) GetGitOpsFiles(w http.ResponseWriter, r *http.Request) {
	p, err := h.DB.GetPlugin(r.Context(), "gitops")
	if err != nil || p == nil {
		writeError(w, http.StatusNotFound, "gitops plugin not installed")
		return
	}
	if !p.Enabled {
		writeError(w, http.StatusBadRequest, "gitops plugin not enabled")
		return
	}

	if h.GitOps == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	files, err := h.GitOps.ListFiles()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, files)
}
