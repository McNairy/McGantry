package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gantrydev/gantry/internal/entity"
)

// ---------------------------------------------------------------------------
// Local types (defined here to avoid circular imports with auth package)
// ---------------------------------------------------------------------------

// User represents a Gantry user account.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"displayName,omitempty"`
	Email        string    `json:"email,omitempty"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// AuditEntry records a single auditable action in the system.
type AuditEntry struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	UserID       string    `json:"userId,omitempty"`
	UserName     string    `json:"userName,omitempty"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resourceType,omitempty"`
	ResourceID   string    `json:"resourceId,omitempty"`
	ResourceName string    `json:"resourceName,omitempty"`
	BeforeState  string    `json:"beforeState,omitempty"`
	AfterState   string    `json:"afterState,omitempty"`
	Source       string    `json:"source,omitempty"`
	IPAddress    string    `json:"ipAddress,omitempty"`
}

// ActionRun represents a single execution of a self-service action.
type ActionRun struct {
	ID          string     `json:"id"`
	ActionName  string     `json:"actionName"`
	Status      string     `json:"status"`
	Inputs      string     `json:"inputs,omitempty"`
	Outputs     string     `json:"outputs,omitempty"`
	TriggeredBy string     `json:"triggeredBy,omitempty"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// APIKey represents a long-lived API authentication token.
type APIKey struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	Name        string     `json:"name"`
	Prefix      string     `json:"prefix"`
	Role        string     `json:"role"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
}

// CreateAPIKey inserts a new API key record. keyHash is the SHA-256 hex of the raw key.
func (d *DB) CreateAPIKey(ctx context.Context, key *APIKey, keyHash string) error {
	key.ID = newUUID()
	key.CreatedAt = time.Now().UTC()
	_, err := d.exec(ctx,
		`INSERT INTO api_keys (id, user_id, name, key_hash, prefix, role, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		key.ID, key.UserID, key.Name, keyHash, key.Prefix, key.Role, key.CreatedAt)
	return err
}

// GetAPIKeyByHash looks up an API key by its SHA-256 hash and updates last_used_at.
func (d *DB) GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	row := d.queryRow(ctx,
		`SELECT id, user_id, name, prefix, role, created_at, last_used_at, expires_at
		 FROM api_keys WHERE key_hash = ?`, keyHash)
	k := &APIKey{}
	var lastUsed, expires sql.NullTime
	if err := row.Scan(&k.ID, &k.UserID, &k.Name, &k.Prefix, &k.Role,
		&k.CreatedAt, &lastUsed, &expires); err != nil {
		if err == sql.ErrNoRows {
			return nil, entity.ErrEntityNotFound
		}
		return nil, err
	}
	if lastUsed.Valid {
		k.LastUsedAt = &lastUsed.Time
	}
	if expires.Valid {
		k.ExpiresAt = &expires.Time
	}
	// Update last_used_at in background — best effort.
	now := time.Now().UTC()
	_, _ = d.exec(ctx, `UPDATE api_keys SET last_used_at = ? WHERE id = ?`, now, k.ID)
	return k, nil
}

// ListAPIKeys returns all API keys for the given user (hashes not included).
func (d *DB) ListAPIKeys(ctx context.Context, userID string) ([]*APIKey, error) {
	rows, err := d.queryRows(ctx,
		`SELECT id, user_id, name, prefix, role, created_at, last_used_at, expires_at
		 FROM api_keys WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []*APIKey
	for rows.Next() {
		k := &APIKey{}
		var lastUsed, expires sql.NullTime
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.Prefix, &k.Role,
			&k.CreatedAt, &lastUsed, &expires); err != nil {
			return nil, err
		}
		if lastUsed.Valid {
			k.LastUsedAt = &lastUsed.Time
		}
		if expires.Valid {
			k.ExpiresAt = &expires.Time
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// DeleteAPIKey removes an API key by ID, verifying it belongs to userID.
func (d *DB) DeleteAPIKey(ctx context.Context, id, userID string) error {
	res, err := d.exec(ctx, `DELETE FROM api_keys WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return entity.ErrEntityNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// UUID generation
// ---------------------------------------------------------------------------

// newUUID generates a v4 UUID using crypto/rand.
func newUUID() string {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	// Set version 4 and variant bits per RFC 4122.
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// ---------------------------------------------------------------------------
// JSON helpers
// ---------------------------------------------------------------------------

// marshalJSON marshals v to a JSON string, returning "" for nil/empty values.
func marshalJSON(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	s := string(b)
	// Treat null/empty representations as empty string for cleaner storage.
	if s == "null" || s == "[]" || s == "{}" {
		return ""
	}
	return s
}

// unmarshalStringSlice decodes a JSON array stored as a string into a []string.
func unmarshalStringSlice(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	return out
}

// unmarshalStringMap decodes a JSON object stored as a string into a map[string]string.
func unmarshalStringMap(s string) map[string]string {
	if s == "" {
		return nil
	}
	var out map[string]string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	return out
}

// unmarshalAnyMap decodes a JSON object stored as a string into a map[string]any.
func unmarshalAnyMap(s string) map[string]any {
	if s == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	return out
}

// ---------------------------------------------------------------------------
// Entity CRUD
// ---------------------------------------------------------------------------

// CreateEntity inserts a new entity into the database.
// It generates a UUID for the entity and sets timestamps.
func (d *DB) CreateEntity(ctx context.Context, e *entity.Entity) error {
	id := newUUID()
	now := time.Now().UTC()

	if e.Metadata.CreatedAt.IsZero() {
		e.Metadata.CreatedAt = now
	}
	e.Metadata.UpdatedAt = now

	tags := marshalJSON(e.Metadata.Tags)
	annotations := marshalJSON(e.Metadata.Annotations)
	labels := marshalJSON(e.Metadata.Labels)
	spec := marshalJSON(e.Spec)

	_, err := d.exec(ctx,
		`INSERT INTO entities (id, kind, api_version, name, namespace, title, description, owner,
			tags, annotations, labels, spec, created_at, updated_at, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id,
		e.Kind,
		e.APIVersion,
		e.Metadata.Name,
		e.Metadata.Namespace,
		e.Metadata.Title,
		e.Metadata.Description,
		e.Metadata.Owner,
		tags,
		annotations,
		labels,
		spec,
		e.Metadata.CreatedAt,
		e.Metadata.UpdatedAt,
		e.Metadata.CreatedBy,
	)
	if err != nil {
		// Check for unique constraint violation.
		if isUniqueViolation(err) {
			return entity.ErrEntityAlreadyExists
		}
		return fmt.Errorf("inserting entity: %w", err)
	}
	return nil
}

// GetEntity retrieves a single entity by kind, namespace, and name.
func (d *DB) GetEntity(ctx context.Context, kind, namespace, name string) (*entity.Entity, error) {
	row := d.queryRow(ctx,
		`SELECT kind, api_version, name, namespace, title, description, owner,
			tags, annotations, labels, spec, created_at, updated_at, created_by
		 FROM entities
		 WHERE kind = ? AND namespace = ? AND name = ?`,
		kind, namespace, name,
	)
	return scanEntity(row)
}

// ListEntities returns all entities matching the given kind and namespace.
// Pass empty strings to skip filtering on that dimension.
func (d *DB) ListEntities(ctx context.Context, kind, namespace string) ([]*entity.Entity, error) {
	query := `SELECT kind, api_version, name, namespace, title, description, owner,
		tags, annotations, labels, spec, created_at, updated_at, created_by
		FROM entities WHERE 1=1`
	var args []any

	if kind != "" {
		query += " AND kind = ?"
		args = append(args, kind)
	}
	if namespace != "" {
		query += " AND namespace = ?"
		args = append(args, namespace)
	}

	query += " ORDER BY kind, namespace, name"

	rows, err := d.queryRows(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing entities: %w", err)
	}
	defer rows.Close()

	var entities []*entity.Entity
	for rows.Next() {
		e, err := scanEntityFromRows(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating entity rows: %w", err)
	}
	return entities, nil
}

// CountEntitiesByKind returns a map of entity kind to count for all entities.
func (d *DB) CountEntitiesByKind(ctx context.Context) (map[string]int64, error) {
	rows, err := d.queryRows(ctx, `SELECT kind, COUNT(*) FROM entities GROUP BY kind`)
	if err != nil {
		return nil, fmt.Errorf("counting entities by kind: %w", err)
	}
	defer rows.Close()
	counts := make(map[string]int64)
	for rows.Next() {
		var kind string
		var count int64
		if err := rows.Scan(&kind, &count); err != nil {
			return nil, err
		}
		counts[kind] = count
	}
	return counts, rows.Err()
}

// UpdateEntity updates an existing entity identified by kind, namespace, and name.
// It updates all mutable fields and bumps the updated_at timestamp.
func (d *DB) UpdateEntity(ctx context.Context, e *entity.Entity) error {
	e.Metadata.UpdatedAt = time.Now().UTC()

	tags := marshalJSON(e.Metadata.Tags)
	annotations := marshalJSON(e.Metadata.Annotations)
	labels := marshalJSON(e.Metadata.Labels)
	spec := marshalJSON(e.Spec)

	result, err := d.exec(ctx,
		`UPDATE entities
		 SET api_version = ?, title = ?, description = ?, owner = ?,
		     tags = ?, annotations = ?, labels = ?, spec = ?,
		     updated_at = ?, created_by = ?
		 WHERE kind = ? AND namespace = ? AND name = ?`,
		e.APIVersion,
		e.Metadata.Title,
		e.Metadata.Description,
		e.Metadata.Owner,
		tags,
		annotations,
		labels,
		spec,
		e.Metadata.UpdatedAt,
		e.Metadata.CreatedBy,
		e.Kind,
		e.Metadata.Namespace,
		e.Metadata.Name,
	)
	if err != nil {
		return fmt.Errorf("updating entity: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return entity.ErrEntityNotFound
	}
	return nil
}

// DeleteEntity removes an entity by kind, namespace, and name.
func (d *DB) DeleteEntity(ctx context.Context, kind, namespace, name string) error {
	result, err := d.exec(ctx,
		`DELETE FROM entities WHERE kind = ? AND namespace = ? AND name = ?`,
		kind, namespace, name,
	)
	if err != nil {
		return fmt.Errorf("deleting entity: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return entity.ErrEntityNotFound
	}
	return nil
}

// SearchEntities performs a full-text search across entity fields.
// On SQLite it uses FTS5; on PostgreSQL it falls back to LIKE matching.
func (d *DB) SearchEntities(ctx context.Context, query string) ([]*entity.Entity, error) {
	if query == "" {
		return nil, nil
	}

	var sqlQuery string
	var args []any

	if d.IsSQLite() {
		// Use FTS5 for SQLite. The MATCH query supports prefix matching with *.
		sqlQuery = `SELECT e.kind, e.api_version, e.name, e.namespace, e.title, e.description, e.owner,
				e.tags, e.annotations, e.labels, e.spec, e.created_at, e.updated_at, e.created_by
			FROM entities e
			JOIN entities_fts fts ON e.rowid = fts.rowid
			WHERE entities_fts MATCH ?
			ORDER BY rank`
		// Append * for prefix matching so partial words match.
		args = append(args, query+"*")
	} else {
		// Fallback LIKE search for PostgreSQL (tsvector search can be added later).
		likePattern := "%" + query + "%"
		sqlQuery = `SELECT kind, api_version, name, namespace, title, description, owner,
				tags, annotations, labels, spec, created_at, updated_at, created_by
			FROM entities
			WHERE name ILIKE ? OR title ILIKE ? OR description ILIKE ? OR tags ILIKE ? OR kind ILIKE ? OR owner ILIKE ?
			ORDER BY name`
		args = append(args, likePattern, likePattern, likePattern, likePattern, likePattern, likePattern)
	}

	rows, err := d.queryRows(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("searching entities: %w", err)
	}
	defer rows.Close()

	var entities []*entity.Entity
	for rows.Next() {
		e, err := scanEntityFromRows(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating search results: %w", err)
	}
	return entities, nil
}

// ---------------------------------------------------------------------------
// User queries
// ---------------------------------------------------------------------------

// GetUserByUsername retrieves a user by their unique username.
func (d *DB) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	row := d.queryRow(ctx,
		`SELECT id, username, password_hash, display_name, email, role, created_at, updated_at
		 FROM users WHERE username = ?`,
		username,
	)

	u := &User{}
	var displayName, email sql.NullString
	var createdAt, updatedAt sql.NullTime
	err := row.Scan(
		&u.ID, &u.Username, &u.PasswordHash,
		&displayName, &email, &u.Role,
		&createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user %q: %w", username, entity.ErrEntityNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("querying user: %w", err)
	}

	u.DisplayName = displayName.String
	u.Email = email.String
	if createdAt.Valid {
		u.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		u.UpdatedAt = updatedAt.Time
	}
	return u, nil
}

// ListUsers returns all users ordered by creation date (password hash excluded).
func (d *DB) ListUsers(ctx context.Context) ([]*User, error) {
	rows, err := d.queryRows(ctx,
		`SELECT id, username, display_name, email, role, created_at, updated_at
		 FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		var displayName, email sql.NullString
		var createdAt, updatedAt sql.NullTime
		if err := rows.Scan(&u.ID, &u.Username, &displayName, &email, &u.Role, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		u.DisplayName = displayName.String
		u.Email = email.String
		if createdAt.Valid {
			u.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			u.UpdatedAt = updatedAt.Time
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetUserByID retrieves a user by their ID.
func (d *DB) GetUserByID(ctx context.Context, id string) (*User, error) {
	row := d.queryRow(ctx,
		`SELECT id, username, password_hash, display_name, email, role, created_at, updated_at
		 FROM users WHERE id = ?`, id)
	u := &User{}
	var displayName, email sql.NullString
	var createdAt, updatedAt sql.NullTime
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &displayName, &email, &u.Role, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user %q: %w", id, entity.ErrEntityNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("querying user: %w", err)
	}
	u.DisplayName = displayName.String
	u.Email = email.String
	if createdAt.Valid {
		u.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		u.UpdatedAt = updatedAt.Time
	}
	return u, nil
}

// UpdateUser updates the mutable profile fields of a user (displayName, email, role).
func (d *DB) UpdateUser(ctx context.Context, u *User) error {
	u.UpdatedAt = time.Now().UTC()
	_, err := d.exec(ctx,
		`UPDATE users SET display_name = ?, email = ?, role = ?, updated_at = ? WHERE id = ?`,
		u.DisplayName, u.Email, u.Role, u.UpdatedAt, u.ID)
	return err
}

// UpdateUserPassword sets a new bcrypt password hash for the given user.
func (d *DB) UpdateUserPassword(ctx context.Context, userID, passwordHash string) error {
	now := time.Now().UTC()
	_, err := d.exec(ctx,
		`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		passwordHash, now, userID)
	return err
}

// DeleteUser removes a user by ID.
func (d *DB) DeleteUser(ctx context.Context, id string) error {
	_, err := d.exec(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

// CreateUser inserts a new user. The caller must set PasswordHash before calling.
func (d *DB) CreateUser(ctx context.Context, u *User) error {
	if u.ID == "" {
		u.ID = newUUID()
	}
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	u.UpdatedAt = now

	if u.Role == "" {
		u.Role = "viewer"
	}

	_, err := d.exec(ctx,
		`INSERT INTO users (id, username, password_hash, display_name, email, role, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Username, u.PasswordHash,
		u.DisplayName, u.Email, u.Role,
		u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("username %q already exists", u.Username)
		}
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Audit log queries
// ---------------------------------------------------------------------------

// CreateAuditEntry records an audit event.
func (d *DB) CreateAuditEntry(ctx context.Context, entry *AuditEntry) error {
	if entry.ID == "" {
		entry.ID = newUUID()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	_, err := d.exec(ctx,
		`INSERT INTO audit_log (id, timestamp, user_id, user_name, action,
			resource_type, resource_id, resource_name, before_state, after_state,
			source, ip_address)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.Timestamp, entry.UserID, entry.UserName, entry.Action,
		entry.ResourceType, entry.ResourceID, entry.ResourceName,
		entry.BeforeState, entry.AfterState,
		entry.Source, entry.IPAddress,
	)
	if err != nil {
		return fmt.Errorf("inserting audit entry: %w", err)
	}
	return nil
}

// ListAuditEntries returns audit log entries ordered by most recent first.
func (d *DB) ListAuditEntries(ctx context.Context, limit, offset int) ([]*AuditEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := d.queryRows(ctx,
		`SELECT id, timestamp, user_id, user_name, action,
			resource_type, resource_id, resource_name, before_state, after_state,
			source, ip_address
		 FROM audit_log
		 ORDER BY timestamp DESC
		 LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("listing audit entries: %w", err)
	}
	defer rows.Close()

	var entries []*AuditEntry
	for rows.Next() {
		e := &AuditEntry{}
		var userID, userName, resourceType, resourceID, resourceName sql.NullString
		var beforeState, afterState, source, ipAddress sql.NullString
		err := rows.Scan(
			&e.ID, &e.Timestamp,
			&userID, &userName, &e.Action,
			&resourceType, &resourceID, &resourceName,
			&beforeState, &afterState,
			&source, &ipAddress,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning audit entry: %w", err)
		}
		e.UserID = userID.String
		e.UserName = userName.String
		e.ResourceType = resourceType.String
		e.ResourceID = resourceID.String
		e.ResourceName = resourceName.String
		e.BeforeState = beforeState.String
		e.AfterState = afterState.String
		e.Source = source.String
		e.IPAddress = ipAddress.String
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating audit entries: %w", err)
	}
	return entries, nil
}

// ---------------------------------------------------------------------------
// Action run queries
// ---------------------------------------------------------------------------

// CreateActionRun inserts a new action run record.
func (d *DB) CreateActionRun(ctx context.Context, run *ActionRun) error {
	if run.ID == "" {
		run.ID = newUUID()
	}
	if run.Status == "" {
		run.Status = "pending"
	}

	_, err := d.exec(ctx,
		`INSERT INTO action_runs (id, action_name, status, inputs, outputs,
			triggered_by, started_at, completed_at, error)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.ActionName, run.Status,
		run.Inputs, run.Outputs, run.TriggeredBy,
		run.StartedAt, run.CompletedAt, run.Error,
	)
	if err != nil {
		return fmt.Errorf("inserting action run: %w", err)
	}
	return nil
}

// GetActionRun retrieves a single action run by ID.
func (d *DB) GetActionRun(ctx context.Context, id string) (*ActionRun, error) {
	row := d.queryRow(ctx,
		`SELECT id, action_name, status, inputs, outputs,
			triggered_by, started_at, completed_at, error
		 FROM action_runs WHERE id = ?`,
		id,
	)

	run, err := scanActionRun(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("action run %q: %w", id, entity.ErrEntityNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("querying action run: %w", err)
	}
	return run, nil
}

// UpdateActionRun updates a mutable action run record (status, outputs, timestamps, error).
func (d *DB) UpdateActionRun(ctx context.Context, run *ActionRun) error {
	result, err := d.exec(ctx,
		`UPDATE action_runs
		 SET status = ?, inputs = ?, outputs = ?,
		     triggered_by = ?, started_at = ?, completed_at = ?, error = ?
		 WHERE id = ?`,
		run.Status, run.Inputs, run.Outputs,
		run.TriggeredBy, run.StartedAt, run.CompletedAt, run.Error,
		run.ID,
	)
	if err != nil {
		return fmt.Errorf("updating action run: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("action run %q: %w", run.ID, entity.ErrEntityNotFound)
	}
	return nil
}

// ListActionRuns returns action runs for a given action name, ordered by most recent first.
// If actionName is empty, all runs are returned.
func (d *DB) ListActionRuns(ctx context.Context, actionName string) ([]*ActionRun, error) {
	var query string
	var args []any

	if actionName != "" {
		query = `SELECT id, action_name, status, inputs, outputs,
				triggered_by, started_at, completed_at, error
			 FROM action_runs WHERE action_name = ?
			 ORDER BY started_at DESC NULLS LAST`
		args = append(args, actionName)
	} else {
		query = `SELECT id, action_name, status, inputs, outputs,
				triggered_by, started_at, completed_at, error
			 FROM action_runs
			 ORDER BY started_at DESC NULLS LAST`
	}

	rows, err := d.queryRows(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing action runs: %w", err)
	}
	defer rows.Close()

	var runs []*ActionRun
	for rows.Next() {
		run, err := scanActionRunFromRows(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating action runs: %w", err)
	}
	return runs, nil
}

// ---------------------------------------------------------------------------
// Row scanning helpers
// ---------------------------------------------------------------------------

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanEntityFields scans entity columns into an Entity struct from any scanner.
func scanEntityFields(s scanner) (*entity.Entity, error) {
	e := &entity.Entity{}
	var title, description, owner, createdBy sql.NullString
	var tags, annotations, labels, spec sql.NullString
	var createdAt, updatedAt sql.NullTime

	err := s.Scan(
		&e.Kind, &e.APIVersion,
		&e.Metadata.Name, &e.Metadata.Namespace,
		&title, &description, &owner,
		&tags, &annotations, &labels, &spec,
		&createdAt, &updatedAt, &createdBy,
	)
	if err != nil {
		return nil, err
	}

	e.Metadata.Title = title.String
	e.Metadata.Description = description.String
	e.Metadata.Owner = owner.String
	e.Metadata.CreatedBy = createdBy.String
	e.Metadata.Tags = unmarshalStringSlice(tags.String)
	e.Metadata.Annotations = unmarshalStringMap(annotations.String)
	e.Metadata.Labels = unmarshalStringMap(labels.String)
	e.Spec = unmarshalAnyMap(spec.String)
	if createdAt.Valid {
		e.Metadata.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		e.Metadata.UpdatedAt = updatedAt.Time
	}
	return e, nil
}

// scanEntity scans a single entity row from a *sql.Row.
func scanEntity(row *sql.Row) (*entity.Entity, error) {
	e, err := scanEntityFields(row)
	if err == sql.ErrNoRows {
		return nil, entity.ErrEntityNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning entity: %w", err)
	}
	return e, nil
}

// scanEntityFromRows scans a single entity row from a *sql.Rows iterator.
func scanEntityFromRows(rows *sql.Rows) (*entity.Entity, error) {
	e, err := scanEntityFields(rows)
	if err != nil {
		return nil, fmt.Errorf("scanning entity row: %w", err)
	}
	return e, nil
}

// scanActionRunFields scans action_run columns from any scanner.
func scanActionRunFields(s scanner) (*ActionRun, error) {
	run := &ActionRun{}
	var inputs, outputs, triggeredBy, errMsg sql.NullString
	var startedAt, completedAt sql.NullTime

	err := s.Scan(
		&run.ID, &run.ActionName, &run.Status,
		&inputs, &outputs, &triggeredBy,
		&startedAt, &completedAt, &errMsg,
	)
	if err != nil {
		return nil, err
	}

	run.Inputs = inputs.String
	run.Outputs = outputs.String
	run.TriggeredBy = triggeredBy.String
	run.Error = errMsg.String
	if startedAt.Valid {
		run.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		run.CompletedAt = &completedAt.Time
	}
	return run, nil
}

// scanActionRun scans a single action run from a *sql.Row.
func scanActionRun(row *sql.Row) (*ActionRun, error) {
	run, err := scanActionRunFields(row)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("scanning action run: %w", err)
	}
	return run, nil
}

// scanActionRunFromRows scans a single action run from a *sql.Rows iterator.
func scanActionRunFromRows(rows *sql.Rows) (*ActionRun, error) {
	run, err := scanActionRunFields(rows)
	if err != nil {
		return nil, fmt.Errorf("scanning action run row: %w", err)
	}
	return run, nil
}

// ---------------------------------------------------------------------------
// Error detection helpers
// ---------------------------------------------------------------------------

// isUniqueViolation checks if an error is a unique constraint violation.
// It handles both SQLite and PostgreSQL error messages.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// SQLite: "UNIQUE constraint failed: ..."
	// PostgreSQL: "duplicate key value violates unique constraint ..."
	return contains(msg, "UNIQUE constraint failed") ||
		contains(msg, "duplicate key value violates unique constraint")
}

// contains is a simple substring check (avoids importing strings for one call).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
