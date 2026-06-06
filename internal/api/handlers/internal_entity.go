package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go2engle/gantry/internal/entity"
)

// InternalGetEntity handles GET /api/internal/entity.
// Authenticated via X-Gantry-Internal-Token.
// Query params: kind, namespace, name.
// When name is provided returns a single entity; when omitted returns an array.
func (h *Handlers) InternalGetEntity(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Gantry-Internal-Token") != h.InternalPluginToken {
		writeError(w, http.StatusUnauthorized, "invalid internal token")
		return
	}

	kind := r.URL.Query().Get("kind")
	namespace := r.URL.Query().Get("namespace")
	name := r.URL.Query().Get("name")
	ctx := r.Context()

	if name != "" {
		e, err := h.DB.GetEntity(ctx, kind, namespace, name)
		if errors.Is(err, entity.ErrEntityNotFound) || e == nil {
			writeError(w, http.StatusNotFound, "entity not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "get entity: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, e)
		return
	}

	entities, err := h.DB.ListEntities(ctx, kind, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list entities: "+err.Error())
		return
	}
	if entities == nil {
		entities = []*entity.Entity{}
	}
	writeJSON(w, http.StatusOK, entities)
}

// InternalUpsertEntity handles POST /api/internal/entity-upsert.
// Creates the entity if it does not exist; updates it if it does.
func (h *Handlers) InternalUpsertEntity(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Gantry-Internal-Token") != h.InternalPluginToken {
		writeError(w, http.StatusUnauthorized, "invalid internal token")
		return
	}

	var e entity.Entity
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	e.SetDefaults()
	if err := e.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	existing, err := h.DB.GetEntity(ctx, e.Kind, e.Metadata.Namespace, e.Metadata.Name)
	if err != nil && !errors.Is(err, entity.ErrEntityNotFound) {
		writeError(w, http.StatusInternalServerError, "get entity: "+err.Error())
		return
	}

	if existing == nil || errors.Is(err, entity.ErrEntityNotFound) {
		if err := h.DB.CreateEntity(ctx, &e); err != nil {
			writeError(w, http.StatusInternalServerError, "create entity: "+err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, &e)
		return
	}

	e.Metadata.CreatedAt = existing.Metadata.CreatedAt
	if err := h.DB.UpdateEntity(ctx, &e); err != nil {
		writeError(w, http.StatusInternalServerError, "update entity: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, &e)
}

// InternalDeleteEntity handles POST /api/internal/entity-delete.
// Returns 204 on success, 404 if not found (idempotent).
func (h *Handlers) InternalDeleteEntity(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Gantry-Internal-Token") != h.InternalPluginToken {
		writeError(w, http.StatusUnauthorized, "invalid internal token")
		return
	}

	var req struct {
		Kind      string `json:"kind"`
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Kind == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "kind and name are required")
		return
	}
	if req.Namespace == "" {
		req.Namespace = entity.DefaultNamespace
	}

	ctx := r.Context()
	if err := h.DB.DeleteEntity(ctx, req.Kind, req.Namespace, req.Name); err != nil {
		if errors.Is(err, entity.ErrEntityNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, "delete entity: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
