// Package db provides database access for Gantry.
// It supports both SQLite (via modernc.org/sqlite, pure Go) and PostgreSQL.
// SQLite is the default for single-binary deployments; PostgreSQL is used
// for production multi-instance setups.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gantrydev/gantry/internal/config"

	// SQLite driver (pure Go, no CGO required).
	_ "modernc.org/sqlite"
)

// DB wraps a standard database/sql.DB connection with Gantry-specific helpers.
type DB struct {
	*sql.DB
	cfg *config.Config
}

// New opens a database connection based on the provided configuration.
// For SQLite, it ensures the data directory exists and enables WAL mode
// and foreign keys. For PostgreSQL, it connects using the provided DSN.
func New(cfg *config.Config) (*DB, error) {
	// For SQLite, ensure the data directory exists.
	if cfg.DBType == "sqlite" {
		dir := filepath.Dir(cfg.DBDSN)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("creating data directory %s: %w", dir, err)
		}
	}

	sqlDB, err := sql.Open(cfg.DriverName(), cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Verify connectivity.
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	// Tune connection pool for SQLite (single-writer).
	if cfg.DBType == "sqlite" {
		sqlDB.SetMaxOpenConns(1)
	}

	return &DB{DB: sqlDB, cfg: cfg}, nil
}

// Migrate runs all database migrations to bring the schema up to date.
// Migrations are idempotent (using IF NOT EXISTS) so they are safe to run
// on every startup.
func (d *DB) Migrate() error {
	migrations := allMigrations(d.cfg.DBType)
	for i, m := range migrations {
		if _, err := d.Exec(m); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}
	return nil
}

// IsSQLite returns true if the underlying database is SQLite.
func (d *DB) IsSQLite() bool {
	return d.cfg.DBType == "sqlite"
}
