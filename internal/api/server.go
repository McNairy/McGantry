// Package api provides the HTTP server and routing for the Gantry API.
package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/gantrydev/gantry/internal/api/handlers"
	"github.com/gantrydev/gantry/internal/api/middleware"
	"github.com/gantrydev/gantry/internal/api/websocket"
	"github.com/gantrydev/gantry/internal/auth"
	"github.com/gantrydev/gantry/internal/config"
	"github.com/gantrydev/gantry/internal/db"
	"github.com/gantrydev/gantry/internal/entity"
	"github.com/gantrydev/gantry/internal/events"
	"github.com/gantrydev/gantry/internal/search"
)

// Server is the Gantry HTTP server.
type Server struct {
	config  *config.Config
	handler http.Handler
	port    int
}

// NewServer creates a new Gantry API server with all routes configured.
func NewServer(cfg *config.Config, database *db.DB, authSvc *auth.Service, eventBus *events.Bus, validator *entity.SchemaValidator, searchSvc *search.Service, wsHub *websocket.Hub) *Server {
	r := chi.NewRouter()

	// Core middleware.
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.RequestLogger)

	// CORS middleware.
	if cfg.DevMode {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	} else {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{""}, // same-origin only
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	// Build handlers.
	h := &handlers.Handlers{
		DB:        database,
		Auth:      authSvc,
		Events:    eventBus,
		Validator: validator,
		SearchSvc: searchSvc,
	}

	// Health check routes (public).
	r.Get("/healthz", h.Healthz)
	r.Get("/readyz", h.Readyz)

	// API v1 routes.
	r.Route("/api/v1", func(api chi.Router) {
		// Public auth endpoint.
		api.Post("/auth/login", h.Login)

		// Authenticated routes.
		api.Group(func(protected chi.Router) {
			protected.Use(middleware.RequireAuth(authSvc))

			// Auth endpoints.
			protected.Get("/auth/me", h.GetMe)
			protected.With(middleware.RequireRole("admin")).Post("/auth/register", h.Register)

			// Entity CRUD.
			protected.Get("/entities", h.ListEntities)
			protected.Get("/entities/{kind}", h.ListEntitiesByKind)
			protected.Get("/entities/{kind}/{name}", h.GetEntity)
			protected.Post("/entities", h.CreateEntity)
			protected.Put("/entities/{kind}/{name}", h.UpdateEntity)
			protected.Delete("/entities/{kind}/{name}", h.DeleteEntity)

			// Search.
			protected.Get("/search", h.Search)

			// Schemas.
			protected.Get("/schemas", h.ListSchemas)
			protected.Get("/schemas/{kind}", h.GetSchema)

			// Actions.
			protected.Get("/actions", h.ListActions)
			protected.Post("/actions/{name}/execute", h.ExecuteAction)
			protected.Get("/actions/{name}/runs", h.ListActionRuns)
			protected.Get("/actions/{name}/runs/{id}", h.GetActionRun)

			// Audit log.
			protected.Get("/audit", h.ListAuditEntries)
		})

		// WebSocket (public -- auth is handled inside the WS handshake).
		api.Get("/ws", wsHub.ServeWS)
	})

	// Serve frontend static files from web/dist if the directory exists.
	// Uses SPA fallback: unmatched routes serve index.html.
	webDir := filepath.Join("web", "dist")
	if info, err := os.Stat(webDir); err == nil && info.IsDir() {
		spaHandler := spaFileServer(http.Dir(webDir))
		r.NotFound(spaHandler.ServeHTTP)
	}

	return &Server{
		config:  cfg,
		handler: r,
		port:    cfg.Port,
	}
}

// Router returns the HTTP handler for use with a custom http.Server.
func (s *Server) Router() http.Handler {
	return s.handler
}

// Start begins listening for HTTP requests on the configured port.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	return http.ListenAndServe(addr, s.handler)
}

// spaFileServer returns an http.Handler that serves static files from root.
// If a requested file does not exist, it falls back to serving index.html
// to support single-page application (SPA) client-side routing.
func spaFileServer(root http.FileSystem) http.Handler {
	fs := http.FileServer(root)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to open the requested file.
		path := filepath.Clean(r.URL.Path)
		if path == "" {
			path = "/"
		}

		f, err := root.Open(path)
		if err != nil {
			// File not found -- serve index.html for SPA routing.
			r.URL.Path = "/"
			fs.ServeHTTP(w, r)
			return
		}
		f.Close()

		// File exists -- serve it normally.
		fs.ServeHTTP(w, r)
	})
}
