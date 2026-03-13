package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go2engle/gantry/internal/api/websocket"
	"github.com/go2engle/gantry/internal/auth"
	"github.com/go2engle/gantry/internal/config"
	"github.com/go2engle/gantry/internal/db"
	"github.com/go2engle/gantry/internal/events"
	"github.com/go2engle/gantry/internal/plugins"
	"github.com/go2engle/gantry/internal/search"
)

type testServerEnv struct {
	server  *Server
	db      *db.DB
	authSvc *auth.Service
}

func newTestServerEnv(t *testing.T) *testServerEnv {
	t.Helper()

	dataDir := t.TempDir()
	cfg := &config.Config{
		Port:          8080,
		DBType:        "sqlite",
		DBDSN:         filepath.Join(dataDir, "gantry.db"),
		JWTSecret:     "test-secret",
		DataDir:       dataDir,
		EncryptionKey: "test-encryption-key",
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("db.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	ctx := context.Background()
	if err := database.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := database.SeedDefaultGroups(ctx); err != nil {
		t.Fatalf("SeedDefaultGroups: %v", err)
	}
	if err := database.SeedDefaultRoles(ctx); err != nil {
		t.Fatalf("SeedDefaultRoles: %v", err)
	}

	authSvc := auth.NewService(cfg.JWTSecret)
	server := NewServer(cfg, database, authSvc, events.New(), nil, search.New(database.DB), websocket.NewHub())

	return &testServerEnv{
		server:  server,
		db:      database,
		authSvc: authSvc,
	}
}

func (e *testServerEnv) createUser(t *testing.T, username, role string) (*db.User, string) {
	t.Helper()

	hash, err := e.authSvc.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	user := &db.User{
		Username:     username,
		PasswordHash: hash,
		Role:         role,
	}
	if err := e.db.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	token, err := e.authSvc.GenerateToken(&auth.User{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	return user, token
}

func TestCreateAPIKeyRejectsPrivilegeEscalation(t *testing.T) {
	env := newTestServerEnv(t)
	_, token := env.createUser(t, "viewer-user", "viewer")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/apikeys", bytes.NewBufferString(`{"name":"ci","role":"admin"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	env.server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPIKeyDoesNotInheritOwnerGroupRole(t *testing.T) {
	env := newTestServerEnv(t)
	user, _ := env.createUser(t, "group-admin", "viewer")

	if err := env.db.AddUserToGroupByName(context.Background(), user.ID, "admins"); err != nil {
		t.Fatalf("AddUserToGroupByName: %v", err)
	}

	rawKey, keyHash, prefix, err := auth.GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey: %v", err)
	}
	if err := env.db.CreateAPIKey(context.Background(), &db.APIKey{
		UserID: user.ID,
		Name:   "scoped",
		Prefix: prefix,
		Role:   "viewer",
	}, keyHash); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/users", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)

	rec := httptest.NewRecorder()
	env.server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPluginConfigRequiresDeveloperRole(t *testing.T) {
	env := newTestServerEnv(t)
	_, token := env.createUser(t, "viewer-plugin", "viewer")

	if err := env.db.UpsertPlugin(context.Background(), &plugins.Plugin{
		Name:    "github",
		Version: "1.0.0",
		Enabled: true,
		Config: map[string]any{
			"personalAccessToken": "secret-token",
		},
		Manifest: &plugins.Manifest{
			Name: "github",
		},
	}); err != nil {
		t.Fatalf("UpsertPlugin: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/github/config", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	env.server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWebSocketHandshakeRequiresAuthentication(t *testing.T) {
	env := newTestServerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil)
	rec := httptest.NewRecorder()
	env.server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWebSocketHandshakeAcceptsQueryTokenAuth(t *testing.T) {
	env := newTestServerEnv(t)
	_, token := env.createUser(t, "ws-user", "viewer")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws?token="+token, nil)
	rec := httptest.NewRecorder()
	env.server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 after auth passed to websocket upgrader, got %d: %s", rec.Code, rec.Body.String())
	}
}
