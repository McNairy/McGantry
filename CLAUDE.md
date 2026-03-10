# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Gantry is an open-source internal developer platform (IDP) — a self-hostable alternative to Backstage that deploys as a single binary. It provides a unified service catalog, self-service portal, and GitOps-native configuration management. Apache 2.0 licensed.

The PRD is in `Gantry-PRD-v1.0.md`.

## Build Commands

```bash
# Full build (frontend + backend)
make build

# Backend only (Go binary → bin/gantry)
go build -o bin/gantry ./cmd/gantry

# Frontend only (React → web/dist/)
cd web && npm install && npm run build

# Run in dev mode
bin/gantry serve --dev

# Frontend dev server with hot reload (proxies API to :8080)
cd web && npm run dev

# Live reload dev (air + vite, requires air: go install github.com/air-verse/air@latest)
make dev-watch

# Run all Go tests
go test ./...

# Run a single Go test
go test ./... -run TestEntityCreate -v

# TypeScript type check
cd web && npx tsc --noEmit

# Lint Go code
golangci-lint run ./...

# Format Go code
make fmt
```

## Tech Stack

- **Backend:** Go — single binary, chi router, embedded SQLite (modernc.org/sqlite, pure Go/no CGO)
- **Frontend:** React 18 + Vite + Tailwind CSS + lucide-react icons
- **Auth:** bcrypt passwords + JWT (golang-jwt/jwt/v5)
- **Real-time:** gorilla/websocket
- **Search:** SQLite FTS5 (full-text search)
- **Schema validation:** santhosh-tekuri/jsonschema/v5
- **CLI:** spf13/cobra
- **Encryption:** AES-256-GCM for plugin config secrets (`internal/crypto/`)
- **Live reload:** github.com/air-verse/air (backend), Vite HMR (frontend)

## Architecture

Modular monolith — single process. Key packages:

- `cmd/gantry/` — CLI entry point (serve, apply, get, describe, export, run, version)
- `internal/api/` — HTTP server, routes, SPA serving
- `internal/api/handlers/` — REST handlers (entities, auth, apikeys, search, schemas, actions, audit, health, graph, plugins, github, dashboard, history)
- `internal/api/middleware/` — Auth (JWT + API key), request logging + metrics
- `internal/api/websocket/` — WebSocket hub with channel subscriptions
- `internal/auth/` — Password hashing (bcrypt) and JWT token lifecycle
- `internal/config/` — Config loading from env vars + YAML file
- `internal/crypto/` — AES-256-GCM encryption for DB-stored secrets
- `internal/db/` — Database layer (SQLite), migrations, entity CRUD queries
- `internal/dispatcher/` — Action dispatchers (webhook, etc.)
- `internal/entity/` — Entity types, built-in kinds, JSON Schema validation (schemas embedded via go:embed)
- `internal/events/` — In-process pub/sub event bus
- `internal/metrics/` — Custom Prometheus text format metrics (no external dep)
- `internal/plugins/` — Plugin manifest types, bundled registry, Kubernetes/GitHub/ArgoCD integrations
- `internal/search/` — FTS5 search service
- `web/` — React SPA (served from web/dist/ by Go binary)
  - `web/src/pages/` — Dashboard, Catalog, EntityDetail, Actions, AuditLog, Settings, Login, Plugins, Users
  - `web/src/components/` — Sidebar, CommandPalette, EntityCard, EntityTable, SchemaForm, ThemeToggle, ErrorBoundary, EntityGraph, GitHubTab, KubernetesTab, ArgoCDTab, ActionWizard, ActionFormBuilder
  - `web/src/hooks/` — useAuth, useTheme
  - `web/src/lib/` — api.ts (API client), types.ts, plugin-runtime.ts
  - `web/src/styles/globals.css` — global styles + markdown rendering
- `website/` — Docusaurus documentation site (separate from the app)

### Entity Model

Everything is an entity with `kind`, `apiVersion`, `metadata`, and `spec` fields. Each kind has a JSON Schema in `internal/entity/schemas/` that drives both backend validation and frontend form generation.

Built-in kinds: Service, API, Infrastructure, Team, Environment, Documentation, Action.

### Data Flow

- **UI Path:** User → API handler → schema validation → DB write → event bus → WebSocket broadcast
- **API auth:** JWT Bearer token in Authorization header; claims stored in request context via middleware. API keys use `gantry_<hex>` prefix format.

### Key Patterns

- DB queries use `database/sql` directly (no ORM). JSON fields (tags, annotations, labels, spec) stored as TEXT and marshaled via `encoding/json`.
- Handlers are methods on `handlers.Handlers` struct which holds all service dependencies.
- The `entity.SchemaValidator` uses `go:embed` to load JSON schemas from `internal/entity/schemas/`.
- Frontend uses CSS custom properties (`--gantry-*`) for theming; dark mode via `.dark` class on `<html>`.
- `SchemaForm` component auto-generates forms from JSON Schema for entity creation/editing and action execution.
- Plugin configs with secret fields are encrypted at rest using AES-256-GCM via `internal/crypto/`.
- Browsing history is tracked per-user in the `history` DB table; surfaced on the Dashboard.

## Config

Env vars: `GANTRY_PORT`, `GANTRY_DB`, `GANTRY_DEV`, `GANTRY_ADMIN_PASSWORD`, `GANTRY_JWT_SECRET`, `GANTRY_DATA_DIR`, `GANTRY_ENCRYPTION_KEY`. CLI flags override env vars. Default: port 8080, SQLite at `./data/gantry.db`, admin/changeme.

`GANTRY_ENCRYPTION_KEY` — base64-encoded 32-byte key for AES-256-GCM encryption of plugin config secrets. If not set, secrets are stored unencrypted.

Config can also be loaded from a YAML file (same field names in camelCase).

## API Patterns

REST endpoints: `/api/v1/entities`, `/api/v1/entities/{kind}`, `/api/v1/entities/{kind}/{name}`.
Actions: `/api/v1/actions/{name}/execute`, `/api/v1/actions/{name}/runs/{id}`.
Auth: `POST /api/v1/auth/login`, `GET /api/v1/auth/me`, `GET /api/v1/auth/github` (OAuth SSO).
Search: `GET /api/v1/search?q=`.
WebSocket: `GET /api/v1/ws`.
Health: `/healthz`, `/readyz`.
Metrics: `GET /metrics` (Prometheus text format).
Dashboard: `GET /api/v1/dashboard`.
History: `GET /api/v1/history`, `POST /api/v1/history`.
Graph: `GET /api/v1/graph/{kind}/{name}`.
Plugins: `GET/POST /api/v1/plugins`, `GET/PUT/DELETE /api/v1/plugins/{name}`, `POST /api/v1/plugins/{name}/enable`, `POST /api/v1/plugins/{name}/disable`, `POST /api/v1/plugins/{name}/sync`.
API Keys: `GET/POST /api/v1/apikeys`, `DELETE /api/v1/apikeys/{id}`.

## CSS Theming

Only these CSS custom properties are defined (do NOT use others):
`--gantry-bg-primary`, `--gantry-bg-secondary`, `--gantry-bg-tertiary`, `--gantry-text-primary`, `--gantry-text-secondary`, `--gantry-border`, `--gantry-accent`, `--gantry-accent-hover`, `--gantry-danger`.

For transparent accent backgrounds use `bg-[var(--gantry-accent)]/10` (Tailwind opacity modifier), NOT `bg-opacity-10`.
