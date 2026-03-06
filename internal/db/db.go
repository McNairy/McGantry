// Package db provides database access for Gantry.
// It supports both SQLite (via modernc.org/sqlite, pure Go) and PostgreSQL.
// SQLite is the default for single-binary deployments; PostgreSQL is used
// for production multi-instance setups.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gantrydev/gantry/internal/config"

	// SQLite driver (pure Go, no CGO required).
	_ "modernc.org/sqlite"
	// PostgreSQL driver.
	_ "github.com/lib/pq"
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

// rebind converts SQLite-style ? placeholders to PostgreSQL-style $1, $2, ...
// placeholders. For SQLite it returns the query unchanged.
func (d *DB) rebind(query string) string {
	if d.IsSQLite() {
		return query
	}
	out := make([]byte, 0, len(query)+8)
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			out = fmt.Appendf(out, "$%d", n)
		} else {
			out = append(out, query[i])
		}
	}
	return string(out)
}

// exec is a rebinding wrapper around ExecContext.
func (d *DB) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.ExecContext(ctx, d.rebind(query), args...)
}

// queryRows is a rebinding wrapper around QueryContext.
func (d *DB) queryRows(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.QueryContext(ctx, d.rebind(query), args...)
}

// queryRow is a rebinding wrapper around QueryRowContext.
func (d *DB) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return d.QueryRowContext(ctx, d.rebind(query), args...)
}
