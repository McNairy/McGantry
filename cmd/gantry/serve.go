package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go2engle/gantry/internal/api"
	"github.com/go2engle/gantry/internal/api/websocket"
	"github.com/go2engle/gantry/internal/auth"
	"github.com/go2engle/gantry/internal/config"
	"github.com/go2engle/gantry/internal/db"
	"github.com/go2engle/gantry/internal/entity"
	"github.com/go2engle/gantry/internal/events"
	"github.com/go2engle/gantry/internal/plugins"
	"github.com/go2engle/gantry/internal/plugins/external"
	"github.com/go2engle/gantry/internal/search"
	"github.com/spf13/cobra"
)

const banner = `
   ____            _
  / ___| __ _ _ __ | |_ _ __ _   _
 | |  _ / _` + "`" + ` | '_ \| __| '__| | | |
 | |_| | (_| | | | | |_| |  | |_| |
  \____|\__,_|_| |_|\__|_|   \__, |
                              |___/
  The Developer Platform That Just Works
`

func serveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Gantry server",
		Long: `Start the Gantry API server. The server provides a REST API for managing
entities (services, APIs, infrastructure, teams, etc.) along with a real-time
WebSocket endpoint for live updates.`,
		RunE: runServe,
	}

	cmd.Flags().IntP("port", "p", 0, "Port to listen on (default 8080)")
	cmd.Flags().String("db", "", "Database connection string (default: sqlite://data/gantry.db)")
	cmd.Flags().Bool("dev", false, "Enable development mode")
	cmd.Flags().String("admin-password", "", "Initial admin password")
	cmd.Flags().String("config", "", "Path to config file (default: gantry.yaml)")
	cmd.Flags().String("tls-cert", "", "Path to TLS certificate file (enables HTTPS)")
	cmd.Flags().String("tls-key", "", "Path to TLS private key file (enables HTTPS)")
	cmd.Flags().String("plugin-dir", "", "Directory to scan for external plugin binaries (gantry-plugin-*)")

	return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
	// Load configuration: file → env vars → flags.
	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		configPath = "gantry.yaml"
	}
	cfg := config.LoadWithFile(configPath)

	if port, _ := cmd.Flags().GetInt("port"); port != 0 {
		cfg.Port = port
	}

	if dbConn, _ := cmd.Flags().GetString("db"); dbConn != "" {
		cfg.DBDSN = dbConn
		if len(dbConn) > 10 && (dbConn[:11] == "postgres://" || dbConn[:13] == "postgresql://") {
			cfg.DBType = "postgres"
		} else {
			cfg.DBType = "sqlite"
		}
	}

	if dev, _ := cmd.Flags().GetBool("dev"); dev {
		cfg.DevMode = true
	}

	if adminPw, _ := cmd.Flags().GetString("admin-password"); adminPw != "" {
		cfg.AdminPassword = adminPw
	}

	if pluginDir, _ := cmd.Flags().GetString("plugin-dir"); pluginDir != "" {
		cfg.PluginDir = pluginDir
	}

	tlsCert, _ := cmd.Flags().GetString("tls-cert")
	tlsKey, _ := cmd.Flags().GetString("tls-key")

	// Initialize database and run migrations.
	authService := auth.NewService(cfg.JWTSecret)

	// Derive a stable internal token for plugin->Gantry API calls.
	// HMAC-SHA256(jwtSecret, "gantry-plugin-internal") — changes only if JWTSecret rotates.
	mac := hmac.New(sha256.New, []byte(cfg.JWTSecret))
	mac.Write([]byte("gantry-plugin-internal"))
	internalPluginToken := hex.EncodeToString(mac.Sum(nil))

	database, err := db.New(cfg)
	if err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		return fmt.Errorf("running database migrations: %w", err)
	}
	if err := database.InitializeBootstrapAdminPassword(context.Background(), authService, cfg.AdminPassword); err != nil {
		return fmt.Errorf("initializing bootstrap admin password: %w", err)
	}

	// Seed default groups (Admins, Developers, Platform Engineers).
	if err := database.SeedDefaultGroups(context.Background()); err != nil {
		return fmt.Errorf("seeding default groups: %w", err)
	}

	// Add the bootstrap admin user to the "admins" group (idempotent).
	_ = database.AddUserToGroupByName(context.Background(), db.BootstrapAdminUserID, "admins")

	// Seed default roles (viewer, developer, platform-engineer, admin).
	if err := database.SeedDefaultRoles(context.Background()); err != nil {
		return fmt.Errorf("seeding default roles: %w", err)
	}

	// Load roles into in-memory cache for fast hierarchy lookups.
	if roles, err := database.ListRoles(context.Background()); err == nil {
		roleData := make([]auth.RoleData, len(roles))
		for i, r := range roles {
			roleData[i] = auth.RoleData{Name: r.Name, Level: r.Level, Permissions: r.Permissions}
		}
		auth.InitRoleStore(roleData)
	}

	// Encrypt any plugin configs written before encryption was introduced.
	if err := database.MigrateEncryptPluginConfigs(context.Background()); err != nil {
		return fmt.Errorf("encrypting plugin configs: %w", err)
	}

	// Auto-register all bundled plugins in the DB (preserves existing config/enabled state).
	registry, err := plugins.BundledRegistry()
	if err != nil {
		return fmt.Errorf("loading bundled plugin registry: %w", err)
	}
	if err := database.EnsureBundledPlugins(context.Background(), registry); err != nil {
		return fmt.Errorf("registering bundled plugins: %w", err)
	}

	// Create and start the external plugin manager.
	gantryURL := fmt.Sprintf("http://127.0.0.1:%d", cfg.Port)
	extManager := external.NewManager(func(ctx context.Context, name string) (map[string]string, error) {
		p, err := database.GetPlugin(ctx, name)
		if err != nil || p == nil {
			return nil, err
		}
		out := make(map[string]string, len(p.Config)+2)
		for k, v := range p.Config {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
		out["gantryInternalToken"] = internalPluginToken
		out["gantryUrl"] = gantryURL
		return out, nil
	})
	if cfg.PluginDir != "" {
		if err := extManager.StartAll(context.Background(), cfg.PluginDir); err != nil {
			log.Printf("[external-plugins] warning: %v", err)
		}
		for _, ep := range extManager.All() {
			m := ep.GetManifest()
			if m == nil {
				continue
			}
			dbManifest := &plugins.Manifest{
				Name:          m.Name,
				Title:         m.Title,
				Description:   m.Description,
				Version:       m.Version,
				Author:        m.Author,
				Category:      m.Category,
				IconURL:       m.IconURL,
				Homepage:      m.Homepage,
				EntityPanels:  m.EntityPanels,
				ActionTypes:   m.ActionTypes,
				AuthBeginPath: m.AuthBeginPath,
			}
			if m.ConfigSchemaJSON != "" {
				_ = json.Unmarshal([]byte(m.ConfigSchemaJSON), &dbManifest.ConfigSchema)
			}
			for _, r := range m.Requirements {
				dbManifest.Requirements = append(dbManifest.Requirements, plugins.PluginRequirement{
					Name:        r.Name,
					Description: r.Description,
					Optional:    r.Optional,
				})
			}
			if err := database.EnsureExternalPlugin(context.Background(), dbManifest); err != nil {
				log.Printf("[external-plugins] %s: failed to register in DB: %v", m.Name, err)
			}
		}
	}

	// Create core services.
	eventBus := events.New()
	searchService := search.New(database.DB)

	// Load and compile JSON schemas for entity validation.
	validator, err := entity.NewSchemaValidator("")
	if err != nil {
		return fmt.Errorf("loading schemas: %w", err)
	}

	// Create and start WebSocket hub.
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Wire event bus to WebSocket: broadcast all events to connected clients.
	eventBus.SubscribeAll(func(event events.Event) {
		wsHub.Broadcast(event)
	})

	// Create the API server.
	srv := api.NewServer(cfg, database, authService, eventBus, validator, searchService, wsHub, Version)

	// Inject external plugin manager and internal token into handlers.
	srv.Handlers.ExternalManager = extManager
	srv.Handlers.InternalPluginToken = internalPluginToken
	srv.Handlers.GantryURL = gantryURL

	// Mount reverse-proxy routes for HTTP-capable external plugins.
	if cfg.PluginDir != "" {
		for _, ep := range extManager.All() {
			m := ep.GetManifest()
			if m == nil || !m.SupportsHTTP {
				continue
			}
			addr := ep.GetListenAddr()
			if addr == "" {
				continue
			}
			var routes []external.Route
			if m.HTTPRoutesJSON != "" {
				_ = json.Unmarshal([]byte(m.HTTPRoutesJSON), &routes)
			}
			for _, route := range routes {
				srv.MountPluginProxy(route.Path, "http://"+addr)
				log.Printf("[external-plugins] %s: proxying %s -> http://%s", m.Name, route.Path, addr)
			}
		}
	}

	// Initialize GitOps service if the plugin is installed and enabled.
	srv.Handlers.InitGitOps()

	// Wire entity events to GitOps push (skip events originating from gitops itself).
	// These closures read srv.Handlers.GitOps so they pick up dynamic init/shutdown.
	eventBus.Subscribe(events.EntityCreated, func(event events.Event) {
		if srv.Handlers.GitOps == nil {
			return
		}
		if src, _ := event.Data["source"].(string); src == "gitops" {
			return
		}
		kind, _ := event.Data["kind"].(string)
		name, _ := event.Data["name"].(string)
		namespace, _ := event.Data["namespace"].(string)
		srv.Handlers.GitOps.QueueChange(kind, namespace, name, "write")
	})
	eventBus.Subscribe(events.EntityUpdated, func(event events.Event) {
		if srv.Handlers.GitOps == nil {
			return
		}
		if src, _ := event.Data["source"].(string); src == "gitops" {
			return
		}
		kind, _ := event.Data["kind"].(string)
		name, _ := event.Data["name"].(string)
		namespace, _ := event.Data["namespace"].(string)
		srv.Handlers.GitOps.QueueChange(kind, namespace, name, "write")
	})
	eventBus.Subscribe(events.EntityDeleted, func(event events.Event) {
		if srv.Handlers.GitOps == nil {
			return
		}
		if src, _ := event.Data["source"].(string); src == "gitops" {
			return
		}
		kind, _ := event.Data["kind"].(string)
		name, _ := event.Data["name"].(string)
		namespace, _ := event.Data["namespace"].(string)
		srv.Handlers.GitOps.QueueChange(kind, namespace, name, "delete")
	})

	// Wire RBAC events to GitOps push — any group or rule change triggers a config file write.
	rbacEventHandler := func(event events.Event) {
		if srv.Handlers.GitOps == nil {
			return
		}
		srv.Handlers.GitOps.QueueChange("_config", "rbac", "rbac", "write")
	}
	eventBus.Subscribe(events.GroupCreated, rbacEventHandler)
	eventBus.Subscribe(events.GroupUpdated, rbacEventHandler)
	eventBus.Subscribe(events.GroupDeleted, rbacEventHandler)
	eventBus.Subscribe(events.RBACRuleCreated, rbacEventHandler)
	eventBus.Subscribe(events.RBACRuleDeleted, rbacEventHandler)
	eventBus.Subscribe(events.RoleCreated, rbacEventHandler)
	eventBus.Subscribe(events.RoleUpdated, rbacEventHandler)
	eventBus.Subscribe(events.RoleDeleted, rbacEventHandler)

	// Print startup banner.
	printBanner(cfg, tlsCert != "")

	// Set up HTTP server.
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      srv.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT or SIGTERM.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		if tlsCert != "" && tlsKey != "" {
			serverErr <- httpServer.ListenAndServeTLS(tlsCert, tlsKey)
		} else {
			serverErr <- httpServer.ListenAndServe()
		}
	}()

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case sig := <-shutdown:
		fmt.Printf("\n  Received %v, shutting down gracefully...\n", sig)

		if srv.Handlers.GitOps != nil {
			srv.Handlers.GitOps.Stop()
		}

		extManager.StopAll()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
		fmt.Println("  Server stopped.")
	}

	return nil
}

func printBanner(cfg *config.Config, tls bool) {
	mode := "production"
	if cfg.DevMode {
		mode = "development"
	}
	scheme := "http"
	if tls {
		scheme = "https"
	}

	fmt.Print(banner)
	fmt.Printf("  Version:  %s\n", Version)
	fmt.Printf("  Port:     %d\n", cfg.Port)
	fmt.Printf("  Database: %s (%s)\n", cfg.DBType, cfg.DBDSN)
	fmt.Printf("  Mode:     %s\n", mode)
	fmt.Println()
	fmt.Printf("  → %s://localhost:%d\n\n", scheme, cfg.Port)
}
