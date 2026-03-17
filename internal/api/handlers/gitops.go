package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go2engle/gantry/internal/gitops"
)

// InitGitOps initializes or reinitializes the GitOps service based on the
// current plugin configuration in the database. Safe to call at any time —
// on startup, after enabling the plugin, or after changing config.
func (h *Handlers) InitGitOps() {
	p, err := h.DB.GetPlugin(context.Background(), "gitops")
	if err != nil || p == nil || !p.Enabled {
		if h.GitOps != nil {
			h.GitOps.Stop()
			h.GitOps = nil
		}
		return
	}

	cfg := gitops.ConfigFromPlugin(p.Config, h.DataDir)

	if h.GitOps != nil {
		// Reinit existing service — handles URL changes, re-clones if needed.
		if err := h.GitOps.Reinit(cfg); err != nil {
			log.Printf("[gitops] reinit error: %v", err)
		}
		return
	}

	svc, err := gitops.New(cfg, h.DB)
	if err != nil {
		log.Printf("[gitops] init error: %v", err)
		return
	}
	h.GitOps = svc

	// Start periodic pull if configured.
	if cfg.SyncInterval != "" {
		if interval, err := time.ParseDuration(cfg.SyncInterval); err == nil && interval > 0 {
			svc.StartPullLoop(interval)
		}
	}

	log.Println("[gitops] service initialized")
}

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

// TriggerGitOpsBidirectionalSync performs a bidirectional sync — pulls remote changes
// into the database, then pushes all local entities back to the repo.
func (h *Handlers) TriggerGitOpsBidirectionalSync(w http.ResponseWriter, r *http.Request) {
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

	// Run bidirectional sync in a goroutine; return 202 immediately.
	go func() {
		// Step 1: Pull remote changes and reconcile the database.
		if _, err := h.GitOps.Pull(); err != nil {
			log.Printf("[gitops] bidisync pull error: %v", err)
		}
		// Step 2: Push all local entities back to the repo.
		if _, err := h.GitOps.FullSync(); err != nil {
			log.Printf("[gitops] bidisync sync error: %v", err)
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "bidirectional sync started — check status for progress",
	})
}

// GetGitOpsFileContent returns the raw YAML content of a tracked entity file.
func (h *Handlers) GetGitOpsFileContent(w http.ResponseWriter, r *http.Request) {
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

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		writeError(w, http.StatusBadRequest, "path query parameter required")
		return
	}

	content, err := h.GitOps.GetFileContent(filePath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"content": content})
}
