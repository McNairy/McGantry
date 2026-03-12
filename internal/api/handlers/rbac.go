package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go2engle/gantry/internal/api/middleware"
	"github.com/go2engle/gantry/internal/auth"
	"github.com/go2engle/gantry/internal/db"
	"github.com/go2engle/gantry/internal/events"
)

// ---------------------------------------------------------------------------
// Group endpoints
// ---------------------------------------------------------------------------

type createGroupRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
	Role        string `json:"role,omitempty"`
}

type updateGroupRequest struct {
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
	Role        string `json:"role,omitempty"`
}

type addMemberRequest struct {
	UserID string `json:"userId"`
}

// ListGroups handles GET /groups.
func (h *Handlers) ListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.DB.ListGroups(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list groups: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

// CreateGroup handles POST /groups.
func (h *Handlers) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Role == "" {
		req.Role = "viewer"
	}
	if !auth.IsValidRole(req.Role) {
		writeError(w, http.StatusBadRequest, "invalid role: "+req.Role)
		return
	}

	g := &db.Group{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Source:      "local",
		Role:        req.Role,
	}
	if err := h.DB.CreateGroup(r.Context(), g); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create group: "+err.Error())
		return
	}

	h.Events.Publish(events.Event{
		Type: events.GroupCreated,
		Data: map[string]any{"groupId": g.ID, "name": g.Name},
	})

	writeJSON(w, http.StatusCreated, g)
}

// GetGroup handles GET /groups/{id}.
func (h *Handlers) GetGroupDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	g, err := h.DB.GetGroup(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "group not found")
		return
	}

	members, err := h.DB.ListGroupMembers(r.Context(), id)
	if err != nil {
		members = []*db.User{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"group":   g,
		"members": members,
	})
}

// UpdateGroup handles PUT /groups/{id}.
func (h *Handlers) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	g, err := h.DB.GetGroup(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "group not found")
		return
	}

	var req updateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DisplayName != "" {
		g.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		g.Description = req.Description
	}
	if req.Role != "" {
		if !auth.IsValidRole(req.Role) {
			writeError(w, http.StatusBadRequest, "invalid role: "+req.Role)
			return
		}
		g.Role = req.Role
	}

	if err := h.DB.UpdateGroup(r.Context(), g); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update group: "+err.Error())
		return
	}

	h.Events.Publish(events.Event{
		Type: events.GroupUpdated,
		Data: map[string]any{"groupId": g.ID, "name": g.Name},
	})

	writeJSON(w, http.StatusOK, g)
}

// DeleteGroup handles DELETE /groups/{id}.
func (h *Handlers) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	g, err := h.DB.GetGroup(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "group not found")
		return
	}

	if g.Source == "system" {
		writeError(w, http.StatusBadRequest, "cannot delete built-in system groups")
		return
	}
	if g.Source != "local" {
		writeError(w, http.StatusBadRequest, "cannot delete SSO-synced groups; disable sync instead")
		return
	}

	if err := h.DB.DeleteGroup(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete group: "+err.Error())
		return
	}

	h.Events.Publish(events.Event{
		Type: events.GroupDeleted,
		Data: map[string]any{"groupId": id, "name": g.Name},
	})

	w.WriteHeader(http.StatusNoContent)
}

// ListGroupMembers handles GET /groups/{id}/members.
func (h *Handlers) ListGroupMembers(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	members, err := h.DB.ListGroupMembers(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list members: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, members)
}

// AddGroupMember handles POST /groups/{id}/members.
func (h *Handlers) AddGroupMember(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "id")
	var req addMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "userId is required")
		return
	}

	// Verify group and user exist.
	if _, err := h.DB.GetGroup(r.Context(), groupID); err != nil {
		writeError(w, http.StatusNotFound, "group not found")
		return
	}
	if _, err := h.DB.GetUserByID(r.Context(), req.UserID); err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	if err := h.DB.AddUserToGroup(r.Context(), req.UserID, groupID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add member: "+err.Error())
		return
	}

	h.Events.Publish(events.Event{
		Type: events.GroupUpdated,
		Data: map[string]any{"groupId": groupID, "action": "member_added", "userId": req.UserID},
	})

	w.WriteHeader(http.StatusNoContent)
}

// RemoveGroupMember handles DELETE /groups/{id}/members/{userId}.
func (h *Handlers) RemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")

	if err := h.DB.RemoveUserFromGroup(r.Context(), userID, groupID); err != nil {
		writeError(w, http.StatusNotFound, "membership not found")
		return
	}

	h.Events.Publish(events.Event{
		Type: events.GroupUpdated,
		Data: map[string]any{"groupId": groupID, "action": "member_removed", "userId": userID},
	})

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Permission Rule endpoints
// ---------------------------------------------------------------------------

type createRuleRequest struct {
	SubjectType    string `json:"subjectType"`
	SubjectID      string `json:"subjectId"`
	ResourceType   string `json:"resourceType"`
	ResourceFilter string `json:"resourceFilter,omitempty"`
	Action         string `json:"action"`
	Effect         string `json:"effect"`
}

// ListPermissionRules handles GET /rbac/rules.
func (h *Handlers) ListPermissionRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.DB.ListPermissionRules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list rules: "+err.Error())
		return
	}

	// Enrich with subject names for display.
	for _, rule := range rules {
		rule.SubjectName = h.resolveSubjectName(r, rule.SubjectType, rule.SubjectID)
	}

	writeJSON(w, http.StatusOK, rules)
}

// CreatePermissionRule handles POST /rbac/rules.
func (h *Handlers) CreatePermissionRule(w http.ResponseWriter, r *http.Request) {
	var req createRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.SubjectType != "user" && req.SubjectType != "group" {
		writeError(w, http.StatusBadRequest, "subjectType must be 'user' or 'group'")
		return
	}
	if req.SubjectID == "" {
		writeError(w, http.StatusBadRequest, "subjectId is required")
		return
	}
	validResourceTypes := map[string]bool{"entity": true, "action": true, "plugin": true, "*": true}
	if !validResourceTypes[req.ResourceType] {
		writeError(w, http.StatusBadRequest, "resourceType must be 'entity', 'action', 'plugin', or '*'")
		return
	}
	validActions := map[string]bool{"read": true, "write": true, "delete": true, "execute": true, "admin": true, "*": true}
	if !validActions[req.Action] {
		writeError(w, http.StatusBadRequest, "action must be 'read', 'write', 'delete', 'execute', 'admin', or '*'")
		return
	}
	if req.Effect != "allow" && req.Effect != "deny" {
		writeError(w, http.StatusBadRequest, "effect must be 'allow' or 'deny'")
		return
	}

	rule := &db.PermissionRule{
		SubjectType:    req.SubjectType,
		SubjectID:      req.SubjectID,
		ResourceType:   req.ResourceType,
		ResourceFilter: req.ResourceFilter,
		Action:         req.Action,
		Effect:         req.Effect,
	}
	if err := h.DB.CreatePermissionRule(r.Context(), rule); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create rule: "+err.Error())
		return
	}

	h.Events.Publish(events.Event{
		Type: events.RBACRuleCreated,
		Data: map[string]any{"ruleId": rule.ID},
	})

	writeJSON(w, http.StatusCreated, rule)
}

// DeletePermissionRule handles DELETE /rbac/rules/{id}.
func (h *Handlers) DeletePermissionRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.DB.DeletePermissionRule(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "rule not found")
		return
	}

	h.Events.Publish(events.Event{
		Type: events.RBACRuleDeleted,
		Data: map[string]any{"ruleId": id},
	})

	w.WriteHeader(http.StatusNoContent)
}

// GetEffectivePermissions handles GET /rbac/effective/{userId}.
// Accessible by admins or by the user themselves.
func (h *Handlers) GetEffectivePermissions(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	claims := middleware.GetClaims(r.Context())
	effectiveRole := middleware.GetEffectiveRole(r.Context())

	// Only admins can view other users' permissions.
	if claims.UserID != userID && effectiveRole != "admin" {
		writeError(w, http.StatusForbidden, "can only view your own effective permissions")
		return
	}

	user, err := h.DB.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	groups, err := h.DB.ListUserGroups(r.Context(), userID)
	if err != nil {
		groups = []*db.Group{}
	}

	groupRoles := make([]string, len(groups))
	groupNames := make([]string, len(groups))
	for i, g := range groups {
		groupRoles[i] = g.Role
		groupNames[i] = g.Name
	}

	rules, err := h.DB.GetEffectiveRules(r.Context(), userID)
	if err != nil {
		rules = []*db.PermissionRule{}
	}
	for _, rule := range rules {
		rule.SubjectName = h.resolveSubjectName(r, rule.SubjectType, rule.SubjectID)
	}

	// Import auth package for EffectiveRole computation.
	computedRole := auth.EffectiveRole(user.Role, groupRoles)

	writeJSON(w, http.StatusOK, map[string]any{
		"userId":        userID,
		"username":      user.Username,
		"directRole":    user.Role,
		"effectiveRole": computedRole,
		"groups":        groupNames,
		"rules":         rules,
	})
}

// ---------------------------------------------------------------------------
// RBAC Export/Import
// ---------------------------------------------------------------------------

type rbacConfig struct {
	Groups           []rbacGroup      `json:"groups"`
	GroupMemberships []rbacMembership `json:"groupMemberships"`
	PermissionRules  []rbacRule       `json:"permissionRules"`
}

type rbacGroup struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
	Role        string `json:"role"`
}

type rbacMembership struct {
	Group string   `json:"group"`
	Users []string `json:"users"`
}

type rbacRule struct {
	SubjectType    string `json:"subjectType"`
	SubjectName    string `json:"subjectName"`
	ResourceType   string `json:"resourceType"`
	ResourceFilter string `json:"resourceFilter,omitempty"`
	Action         string `json:"action"`
	Effect         string `json:"effect"`
}

// ExportRBACConfig handles GET /rbac/export. Returns the full RBAC config as JSON.
func (h *Handlers) ExportRBACConfig(w http.ResponseWriter, r *http.Request) {
	groups, err := h.DB.ListGroups(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list groups: "+err.Error())
		return
	}

	var cfg rbacConfig

	for _, g := range groups {
		cfg.Groups = append(cfg.Groups, rbacGroup{
			Name:        g.Name,
			DisplayName: g.DisplayName,
			Description: g.Description,
			Role:        g.Role,
		})

		members, err := h.DB.ListGroupMembers(r.Context(), g.ID)
		if err == nil && len(members) > 0 {
			usernames := make([]string, len(members))
			for i, m := range members {
				usernames[i] = m.Username
			}
			cfg.GroupMemberships = append(cfg.GroupMemberships, rbacMembership{
				Group: g.Name,
				Users: usernames,
			})
		}
	}

	rules, err := h.DB.ListPermissionRules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list rules: "+err.Error())
		return
	}
	for _, rule := range rules {
		subjectName := h.resolveSubjectName(r, rule.SubjectType, rule.SubjectID)
		cfg.PermissionRules = append(cfg.PermissionRules, rbacRule{
			SubjectType:    rule.SubjectType,
			SubjectName:    subjectName,
			ResourceType:   rule.ResourceType,
			ResourceFilter: rule.ResourceFilter,
			Action:         rule.Action,
			Effect:         rule.Effect,
		})
	}

	if cfg.Groups == nil {
		cfg.Groups = []rbacGroup{}
	}
	if cfg.GroupMemberships == nil {
		cfg.GroupMemberships = []rbacMembership{}
	}
	if cfg.PermissionRules == nil {
		cfg.PermissionRules = []rbacRule{}
	}

	writeJSON(w, http.StatusOK, cfg)
}

// ImportRBACConfig handles POST /rbac/import. Replaces the RBAC config from JSON.
func (h *Handlers) ImportRBACConfig(w http.ResponseWriter, r *http.Request) {
	var cfg rbacConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	ctx := r.Context()
	var created, updated, skipped int

	// Reconcile groups: create missing, update existing.
	for _, g := range cfg.Groups {
		existing, err := h.DB.GetGroupByName(ctx, g.Name)
		if err != nil {
			// Group doesn't exist — create it.
			newGroup := &db.Group{
				Name:        g.Name,
				DisplayName: g.DisplayName,
				Description: g.Description,
				Source:      "local",
				Role:        g.Role,
			}
			if g.Role == "" {
				newGroup.Role = "viewer"
			}
			if err := h.DB.CreateGroup(ctx, newGroup); err != nil {
				skipped++
				continue
			}
			created++
		} else {
			// Update existing group.
			if g.DisplayName != "" {
				existing.DisplayName = g.DisplayName
			}
			if g.Description != "" {
				existing.Description = g.Description
			}
			if g.Role != "" {
				existing.Role = g.Role
			}
			if err := h.DB.UpdateGroup(ctx, existing); err != nil {
				skipped++
				continue
			}
			updated++
		}
	}

	// Reconcile memberships.
	for _, m := range cfg.GroupMemberships {
		group, err := h.DB.GetGroupByName(ctx, m.Group)
		if err != nil {
			continue
		}
		var memberIDs []string
		for _, username := range m.Users {
			user, err := h.DB.GetUserByUsername(ctx, username)
			if err != nil {
				continue
			}
			memberIDs = append(memberIDs, user.ID)
		}
		// Clear existing members and re-add.
		existingMembers, _ := h.DB.ListGroupMembers(ctx, group.ID)
		for _, em := range existingMembers {
			_ = h.DB.RemoveUserFromGroup(ctx, em.ID, group.ID)
		}
		for _, uid := range memberIDs {
			_ = h.DB.AddUserToGroup(ctx, uid, group.ID)
		}
	}

	// Replace all permission rules.
	_ = h.DB.DeleteAllPermissionRules(ctx)
	rulesCreated := 0
	for _, rule := range cfg.PermissionRules {
		subjectID := h.resolveSubjectID(r, rule.SubjectType, rule.SubjectName)
		if subjectID == "" {
			continue
		}
		pr := &db.PermissionRule{
			SubjectType:    rule.SubjectType,
			SubjectID:      subjectID,
			ResourceType:   rule.ResourceType,
			ResourceFilter: rule.ResourceFilter,
			Action:         rule.Action,
			Effect:         rule.Effect,
		}
		if err := h.DB.CreatePermissionRule(ctx, pr); err == nil {
			rulesCreated++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"groupsCreated": created,
		"groupsUpdated": updated,
		"groupsSkipped": skipped,
		"rulesImported": rulesCreated,
	})
}

// ---------------------------------------------------------------------------
// Role endpoints
// ---------------------------------------------------------------------------

// ListRoles handles GET /rbac/roles. Returns all configured roles.
func (h *Handlers) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.DB.ListRoles(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list roles: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, roles)
}

// GetRoleByID handles GET /rbac/roles/{id}.
func (h *Handlers) GetRoleByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	role, err := h.DB.GetRole(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "role not found")
		return
	}
	writeJSON(w, http.StatusOK, role)
}

type createRoleRequest struct {
	Name        string          `json:"name"`
	DisplayName string          `json:"displayName"`
	Description string          `json:"description"`
	Level       int             `json:"level"`
	Permissions map[string]bool `json:"permissions"`
}

// CreateRole handles POST /rbac/roles.
func (h *Handlers) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req createRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Level < 1 {
		writeError(w, http.StatusBadRequest, "level must be >= 1")
		return
	}
	if req.Permissions == nil {
		req.Permissions = map[string]bool{}
	}

	role := &db.Role{
		Name:        strings.ToLower(req.Name),
		DisplayName: req.DisplayName,
		Description: req.Description,
		Level:       req.Level,
		Permissions: req.Permissions,
	}
	if err := h.DB.CreateRole(r.Context(), role); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	h.reloadRoles(r.Context())
	h.Events.Publish(events.Event{
		Type: events.RoleCreated,
		Data: map[string]any{"roleId": role.ID, "name": role.Name},
	})
	writeJSON(w, http.StatusCreated, role)
}

type updateRoleRequest struct {
	DisplayName string          `json:"displayName"`
	Description string          `json:"description"`
	Level       int             `json:"level"`
	Permissions map[string]bool `json:"permissions"`
}

// UpdateRole handles PUT /rbac/roles/{id}.
func (h *Handlers) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	role, err := h.DB.GetRole(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "role not found")
		return
	}

	var req updateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DisplayName != "" {
		role.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		role.Description = req.Description
	}
	if req.Level > 0 && !role.BuiltIn {
		role.Level = req.Level
	}
	if req.Permissions != nil {
		// Safety: admin role must always retain the admin permission.
		if role.Name == "admin" {
			req.Permissions["admin"] = true
		}
		role.Permissions = req.Permissions
	}

	if err := h.DB.UpdateRole(r.Context(), role); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update role: "+err.Error())
		return
	}

	h.reloadRoles(r.Context())
	h.Events.Publish(events.Event{
		Type: events.RoleUpdated,
		Data: map[string]any{"roleId": id, "name": role.Name},
	})
	writeJSON(w, http.StatusOK, role)
}

// DeleteRole handles DELETE /rbac/roles/{id}.
func (h *Handlers) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	role, err := h.DB.GetRole(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "role not found")
		return
	}

	if role.BuiltIn {
		writeError(w, http.StatusBadRequest, "cannot delete built-in roles")
		return
	}

	inUse, err := h.DB.IsRoleInUse(r.Context(), role.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check role usage")
		return
	}
	if inUse {
		writeError(w, http.StatusBadRequest, "role is still assigned to users or groups; reassign them first")
		return
	}

	if err := h.DB.DeleteRole(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete role: "+err.Error())
		return
	}

	h.reloadRoles(r.Context())
	h.Events.Publish(events.Event{
		Type: events.RoleDeleted,
		Data: map[string]any{"roleId": id, "name": role.Name},
	})
	w.WriteHeader(http.StatusNoContent)
}

// reloadRoles loads all roles from the DB and refreshes the in-memory cache.
func (h *Handlers) reloadRoles(ctx context.Context) {
	roles, err := h.DB.ListRoles(ctx)
	if err != nil {
		return
	}
	data := make([]auth.RoleData, len(roles))
	for i, r := range roles {
		data[i] = auth.RoleData{Name: r.Name, Level: r.Level, Permissions: r.Permissions}
	}
	auth.InitRoleStore(data)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// resolveSubjectName converts a subject type + ID to a human-readable name.
func (h *Handlers) resolveSubjectName(r *http.Request, subjectType, subjectID string) string {
	if subjectType == "user" {
		if u, err := h.DB.GetUserByID(r.Context(), subjectID); err == nil {
			return u.Username
		}
	} else if subjectType == "group" {
		if g, err := h.DB.GetGroup(r.Context(), subjectID); err == nil {
			return g.Name
		}
	}
	return subjectID
}

// resolveSubjectID converts a subject type + name to an ID.
func (h *Handlers) resolveSubjectID(r *http.Request, subjectType, subjectName string) string {
	if subjectType == "user" {
		if u, err := h.DB.GetUserByUsername(r.Context(), subjectName); err == nil {
			return u.ID
		}
	} else if subjectType == "group" {
		if g, err := h.DB.GetGroupByName(r.Context(), subjectName); err == nil {
			return g.ID
		}
	}
	return ""
}

