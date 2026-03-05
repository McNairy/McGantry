// Package config handles Gantry server configuration.
// Configuration is loaded from environment variables with sensible defaults
// suitable for single-binary deployment.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds all Gantry configuration.
type Config struct {
	Port          int    // HTTP listen port
	DBType        string // "sqlite" or "postgres"
	DBDSN         string // connection string or file path
	DevMode       bool   // development mode (verbose logging, CORS *, etc.)
	AdminPassword string // initial admin password (only used on first run)
	JWTSecret     string // secret for signing JWT tokens
	DataDir       string // directory for SQLite database and other data
}

// Default returns a Config with sensible defaults for local development.
func Default() *Config {
	return &Config{
		Port:          8080,
		DBType:        "sqlite",
		DBDSN:         "", // set from DataDir in normalize()
		DevMode:       false,
		AdminPassword: "changeme",
		JWTSecret:     "", // generated if empty in normalize()
		DataDir:       "./data",
	}
}

// Load reads configuration from environment variables, falling back to defaults.
// Environment variables:
//
//	GANTRY_PORT           - HTTP listen port (default: 8080)
//	GANTRY_DB             - Database connection string. Prefix with "postgres://" for PostgreSQL,
//	                        otherwise treated as SQLite file path (default: <DataDir>/gantry.db)
//	GANTRY_DEV            - Enable development mode: "true", "1", or "yes" (default: false)
//	GANTRY_ADMIN_PASSWORD - Initial admin user password (default: "changeme")
//	GANTRY_JWT_SECRET     - JWT signing secret; auto-generated if not set
//	GANTRY_DATA_DIR       - Data directory for SQLite and other files (default: ./data)
func Load() *Config {
	cfg := Default()

	if v := os.Getenv("GANTRY_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil && port > 0 && port <= 65535 {
			cfg.Port = port
		}
	}

	if v := os.Getenv("GANTRY_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}

	if v := os.Getenv("GANTRY_DB"); v != "" {
		if strings.HasPrefix(v, "postgres://") || strings.HasPrefix(v, "postgresql://") {
			cfg.DBType = "postgres"
			cfg.DBDSN = v
		} else {
			cfg.DBType = "sqlite"
			cfg.DBDSN = v
		}
	}

	if v := os.Getenv("GANTRY_DEV"); v != "" {
		v = strings.ToLower(v)
		cfg.DevMode = v == "true" || v == "1" || v == "yes"
	}

	if v := os.Getenv("GANTRY_ADMIN_PASSWORD"); v != "" {
		cfg.AdminPassword = v
	}

	if v := os.Getenv("GANTRY_JWT_SECRET"); v != "" {
		cfg.JWTSecret = v
	}

	cfg.normalize()
	return cfg
}

// normalize fills in derived values and generates secrets where needed.
func (c *Config) normalize() {
	// Default SQLite DSN based on data directory.
	if c.DBDSN == "" {
		c.DBDSN = filepath.Join(c.DataDir, "gantry.db")
	}

	// Generate a random JWT secret if none was provided.
	if c.JWTSecret == "" {
		c.JWTSecret = generateRandomHex(32)
	}
}

// DSN returns the database connection string appropriate for the configured driver.
// For SQLite it returns a URI with WAL mode and foreign keys enabled.
// For PostgreSQL it returns the DSN as-is.
func (c *Config) DSN() string {
	if c.DBType == "sqlite" {
		return fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)", c.DBDSN)
	}
	return c.DBDSN
}

// DriverName returns the database/sql driver name for the configured database type.
func (c *Config) DriverName() string {
	if c.DBType == "postgres" {
		return "postgres"
	}
	return "sqlite"
}

// generateRandomHex produces a cryptographically random hex string of n bytes.
func generateRandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// This should never happen on a properly functioning OS.
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return hex.EncodeToString(b)
}
