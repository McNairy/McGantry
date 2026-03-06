package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gantrydev/gantry/internal/api/middleware"
	"github.com/gantrydev/gantry/internal/db"
	"github.com/gantrydev/gantry/internal/entity"
	"github.com/gantrydev/gantry/internal/events"
)

// ListActions handles GET /actions. It returns all entities of kind "Action".
func (h *Handlers) ListActions(w http.ResponseWriter, r *http.Request) {
	entities, err := h.DB.ListEntities(r.Context(), "Action", "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list actions")
		return
	}
	if entities == nil {
		entities = []*entity.Entity{}
	}

	writeJSON(w, http.StatusOK, entities)
}

// executeActionRequest represents the JSON body for executing an action.
type executeActionRequest struct {
	Inputs map[string]any `json:"inputs,omitempty"`
}

// ExecuteAction handles POST /actions/{name}/execute. It creates a new action
// run record in the database, publishes an ActionTriggered event, and kicks off
// asynchronous dispatch (webhook, etc.) that transitions the run through
// running → success/failed.
func (h *Handlers) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// Verify the action entity exists and retrieve it for dispatch.
	actionEntity, err := h.DB.GetEntity(r.Context(), "Action", entity.DefaultNamespace, name)
	if err != nil {
		if errors.Is(err, entity.ErrEntityNotFound) {
			writeError(w, http.StatusNotFound, "action not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get action")
		return
	}

	// Parse optional inputs from the request body.
	var req executeActionRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	// Serialize inputs for storage.
	inputsJSON := ""
	if req.Inputs != nil {
		b, err := json.Marshal(req.Inputs)
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to serialize inputs")
			return
		}
		inputsJSON = string(b)
	}

	// Determine who triggered the action.
	triggeredBy := ""
	claims := middleware.GetClaims(r.Context())
	if claims != nil {
		triggeredBy = claims.Username
	}

	now := time.Now().UTC()
	run := &db.ActionRun{
		ActionName:  name,
		Status:      "pending",
		Inputs:      inputsJSON,
		TriggeredBy: triggeredBy,
		StartedAt:   &now,
	}

	if err := h.DB.CreateActionRun(r.Context(), run); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create action run")
		return
	}

	// Publish triggered event.
	h.Events.Publish(events.Event{
		Type: events.ActionTriggered,
		Data: map[string]any{
			"actionName":  name,
			"runId":       run.ID,
			"triggeredBy": triggeredBy,
		},
	})

	// Asynchronously dispatch — updates run status as it progresses.
	go h.Dispatcher.Dispatch(actionEntity, run)

	writeJSON(w, http.StatusCreated, run)
}

// ListAllActionRuns handles GET /actions/runs. It returns recent runs across
// all actions, ordered by most recent first.
func (h *Handlers) ListAllActionRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := h.DB.ListActionRuns(r.Context(), "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list action runs")
		return
	}
	if runs == nil {
		runs = []*db.ActionRun{}
	}
	writeJSON(w, http.StatusOK, runs)
}

// ListActionRuns handles GET /actions/{name}/runs. It returns all runs for a
// given action, ordered by most recent first.
func (h *Handlers) ListActionRuns(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	runs, err := h.DB.ListActionRuns(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list action runs")
		return
	}
	if runs == nil {
		runs = []*db.ActionRun{}
	}

	writeJSON(w, http.StatusOK, runs)
}

// GetActionRun handles GET /actions/{name}/runs/{id}. It returns a specific
// action run by its ID.
func (h *Handlers) GetActionRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	run, err := h.DB.GetActionRun(r.Context(), id)
	if err != nil {
		if errors.Is(err, entity.ErrEntityNotFound) {
			writeError(w, http.StatusNotFound, "action run not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get action run")
		return
	}

	writeJSON(w, http.StatusOK, run)
}
