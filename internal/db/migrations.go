package db

// allMigrations returns the ordered list of SQL migration statements.
// Each statement is idempotent (using IF NOT EXISTS) so migrations can be
// run on every startup without harm.
//
// The dbType parameter ("sqlite" or "postgres") selects dialect-appropriate
// SQL where syntax differs between engines.
func allMigrations(dbType string) []string {
	// Pre-computed bcrypt hash of "changeme" at cost 10.
	// Generated with: golang.org/x/crypto/bcrypt.GenerateFromPassword([]byte("changeme"), 10)
	const adminPasswordHash = "$2a$10$eMgfxZdz20Vk.9EKPJ4oP.g99eQ1JgaHQs/JH7v2fpZZykUcN1Q8y"

	// TIMESTAMP works in both SQLite (type affinity: NUMERIC) and PostgreSQL.
	migrations := []string{
		// ------------------------------------------------------------------
		// Table: entities
		// ------------------------------------------------------------------
		`CREATE TABLE IF NOT EXISTS entities (
			id          TEXT PRIMARY KEY,
			kind        TEXT NOT NULL,
			api_version TEXT NOT NULL DEFAULT 'gantry.io/v1',
			name        TEXT NOT NULL,
			namespace   TEXT NOT NULL DEFAULT 'default',
			title       TEXT,
			description TEXT,
			owner       TEXT,
			tags        TEXT,
			annotations TEXT,
			labels      TEXT,
			spec        TEXT,
			created_at  TIMESTAMP,
			updated_at  TIMESTAMP,
			created_by  TEXT,
			UNIQUE(kind, namespace, name)
		)`,

		// Indexes for common query patterns.
		`CREATE INDEX IF NOT EXISTS idx_entities_kind      ON entities(kind)`,
		`CREATE INDEX IF NOT EXISTS idx_entities_namespace  ON entities(namespace)`,
		`CREATE INDEX IF NOT EXISTS idx_entities_owner      ON entities(owner)`,

		// ------------------------------------------------------------------
		// Table: users
		// ------------------------------------------------------------------
		`CREATE TABLE IF NOT EXISTS users (
			id            TEXT PRIMARY KEY,
			username      TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			display_name  TEXT,
			email         TEXT,
			role          TEXT NOT NULL DEFAULT 'viewer',
			created_at    TIMESTAMP,
			updated_at    TIMESTAMP
		)`,

		// ------------------------------------------------------------------
		// Table: audit_log
		// ------------------------------------------------------------------
		`CREATE TABLE IF NOT EXISTS audit_log (
			id            TEXT PRIMARY KEY,
			timestamp     TIMESTAMP NOT NULL,
			user_id       TEXT,
			user_name     TEXT,
			action        TEXT NOT NULL,
			resource_type TEXT,
			resource_id   TEXT,
			resource_name TEXT,
			before_state  TEXT,
			after_state   TEXT,
			source        TEXT,
			ip_address    TEXT
		)`,

		// ------------------------------------------------------------------
		// Table: action_runs
		// ------------------------------------------------------------------
		`CREATE TABLE IF NOT EXISTS action_runs (
			id           TEXT PRIMARY KEY,
			action_name  TEXT NOT NULL,
			status       TEXT NOT NULL DEFAULT 'pending',
			inputs       TEXT,
			outputs      TEXT,
			triggered_by TEXT,
			started_at   TIMESTAMP,
			completed_at TIMESTAMP,
			error        TEXT
		)`,

		// ------------------------------------------------------------------
		// Table: api_keys
		// ------------------------------------------------------------------
		`CREATE TABLE IF NOT EXISTS api_keys (
			id           TEXT PRIMARY KEY,
			user_id      TEXT NOT NULL,
			name         TEXT NOT NULL,
			key_hash     TEXT UNIQUE NOT NULL,
			prefix       TEXT NOT NULL,
			role         TEXT NOT NULL DEFAULT 'developer',
			created_at   TIMESTAMP NOT NULL,
			last_used_at TIMESTAMP,
			expires_at   TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_user_id  ON api_keys(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash)`,

		// ------------------------------------------------------------------
		// Table: plugins
		// ------------------------------------------------------------------
		`CREATE TABLE IF NOT EXISTS plugins (
			id           TEXT PRIMARY KEY,
			name         TEXT UNIQUE NOT NULL,
			version      TEXT NOT NULL,
			enabled      INTEGER NOT NULL DEFAULT 0,
			config       TEXT,
			manifest     TEXT NOT NULL,
			installed_at TIMESTAMP NOT NULL,
			updated_at   TIMESTAMP NOT NULL
		)`,

		// ------------------------------------------------------------------
		// Table: user_history
		// Per-user recently browsed entity history (max 20 per user).
		// ------------------------------------------------------------------
		`CREATE TABLE IF NOT EXISTS user_history (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			username   TEXT NOT NULL,
			kind       TEXT NOT NULL,
			name       TEXT NOT NULL,
			namespace  TEXT NOT NULL DEFAULT 'default',
			viewed_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(username, kind, name, namespace)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_history_username ON user_history(username, viewed_at DESC)`,

		// ------------------------------------------------------------------
		// Table: dashboard_config
		// Single-row global dashboard configuration (id is always 1).
		// ------------------------------------------------------------------
		`CREATE TABLE IF NOT EXISTS dashboard_config (
			id         INTEGER PRIMARY KEY CHECK (id = 1),
			config     TEXT    NOT NULL DEFAULT '{}',
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_by TEXT
		)`,
	}

	// Default admin user — dialect-aware upsert.
	if dbType == "postgres" {
		migrations = append(migrations,
			`INSERT INTO users (id, username, password_hash, display_name, role, created_at, updated_at)
			 VALUES ('00000000-0000-0000-0000-000000000001', 'admin', '`+adminPasswordHash+`', 'Administrator', 'admin', NOW(), NOW())
			 ON CONFLICT (username) DO NOTHING`,
		)
	} else {
		migrations = append(migrations,
			`INSERT OR IGNORE INTO users (id, username, password_hash, display_name, role, created_at, updated_at)
			 VALUES ('00000000-0000-0000-0000-000000000001', 'admin', '`+adminPasswordHash+`', 'Administrator', 'admin', datetime('now'), datetime('now'))`,
		)
	}

	// Seed the single dashboard_config row — dialect-aware.
	if dbType == "postgres" {
		migrations = append(migrations,
			`INSERT INTO dashboard_config (id, config, updated_at)
			 VALUES (1, '{}', NOW())
			 ON CONFLICT (id) DO NOTHING`,
		)
	} else {
		migrations = append(migrations,
			`INSERT OR IGNORE INTO dashboard_config (id, config, updated_at)
			 VALUES (1, '{}', datetime('now'))`,
		)
	}

	// FTS5 is only available on SQLite. For PostgreSQL, full-text search is
	// handled via ILIKE fallback (tsvector/GIN indexes can be added separately).
	if dbType == "sqlite" {
		migrations = append(migrations,
			// FTS5 virtual table for full-text search across entities.
			`CREATE VIRTUAL TABLE IF NOT EXISTS entities_fts USING fts5(
				name,
				title,
				description,
				tags,
				kind,
				owner,
				content='entities',
				content_rowid='rowid'
			)`,

			// Triggers to keep the FTS index in sync with the entities table.
			`CREATE TRIGGER IF NOT EXISTS entities_ai AFTER INSERT ON entities BEGIN
				INSERT INTO entities_fts(rowid, name, title, description, tags, kind, owner)
				VALUES (new.rowid, new.name, new.title, new.description, new.tags, new.kind, new.owner);
			END`,

			`CREATE TRIGGER IF NOT EXISTS entities_ad AFTER DELETE ON entities BEGIN
				INSERT INTO entities_fts(entities_fts, rowid, name, title, description, tags, kind, owner)
				VALUES ('delete', old.rowid, old.name, old.title, old.description, old.tags, old.kind, old.owner);
			END`,

			`CREATE TRIGGER IF NOT EXISTS entities_au AFTER UPDATE ON entities BEGIN
				INSERT INTO entities_fts(entities_fts, rowid, name, title, description, tags, kind, owner)
				VALUES ('delete', old.rowid, old.name, old.title, old.description, old.tags, old.kind, old.owner);
				INSERT INTO entities_fts(rowid, name, title, description, tags, kind, owner)
				VALUES (new.rowid, new.name, new.title, new.description, new.tags, new.kind, new.owner);
			END`,
		)
	}

	return migrations
}
