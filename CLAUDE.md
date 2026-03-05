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

# Run all Go tests
go test ./...

# Run a single Go test
go test ./... -run TestEntityCreate -v

# TypeScript type check
cd web && npx tsc --noEmit

# Lint Go code
golangci-lint run ./...
```

## Tech Stack

- **Backend:** Go — single binary, chi router, embedded SQLite (modernc.org/sqlite, pure Go/no CGO)
- **Frontend:** React 18 + Vite + Tailwind CSS + lucide-react icons
- **Auth:** bcrypt passwords + JWT (golang-jwt/jwt/v5)
- **Real-time:** gorilla/websocket
- **Search:** SQLite FTS5 (full-text search)
- **Schema validation:** santhosh-tekuri/jsonschema/v5
- **CLI:** spf13/cobra

## Architecture

Modular monolith — single process. Key packages:

- `cmd/gantry/` — CLI entry point (serve, apply, get, describe, version)
- `internal/api/` — HTTP server, routes, SPA serving
- `internal/api/handlers/` — REST handlers (entities, auth, search, schemas, actions, audit, health)
- `internal/api/middleware/` — Auth (JWT), request logging
- `internal/api/websocket/` — WebSocket hub with channel subscriptions
- `internal/auth/` — Password hashing (bcrypt) and JWT token lifecycle
- `internal/config/` — Config loading from env vars with defaults
- `internal/db/` — Database layer (SQLite/Postgres), migrations, entity CRUD queries
- `internal/entity/` — Entity types, built-in kinds, JSON Schema validation (schemas embedded via go:embed)
- `internal/events/` — In-process pub/sub event bus
- `internal/search/` — FTS5 search service
- `web/` — React SPA (served from web/dist/ by Go binary)

### Entity Model

Everything is an entity with `kind`, `apiVersion`, `metadata`, and `spec` fields. Each kind has a JSON Schema in `internal/entity/schemas/` that drives both backend validation and frontend form generation.

Built-in kinds: Service, API, Infrastructure, Team, Environment, Documentation, Action.

### Data Flow

- **UI Path:** User → API handler → schema validation → DB write → event bus → WebSocket broadcast
- **API auth:** JWT Bearer token in Authorization header; claims stored in request context via middleware

### Key Patterns

- DB queries use `database/sql` directly (no ORM). JSON fields (tags, annotations, labels, spec) stored as TEXT and marshaled via `encoding/json`.
- Handlers are methods on `handlers.Handlers` struct which holds all service dependencies.
- The `entity.SchemaValidator` uses `go:embed` to load JSON schemas from `internal/entity/schemas/`.
- Frontend uses CSS custom properties (`--gantry-*`) for theming; dark mode via `.dark` class on `<html>`.
- `SchemaForm` component auto-generates forms from JSON Schema for entity creation/editing and action execution.

## Config

Env vars: `GANTRY_PORT`, `GANTRY_DB`, `GANTRY_DEV`, `GANTRY_ADMIN_PASSWORD`, `GANTRY_JWT_SECRET`, `GANTRY_DATA_DIR`. CLI flags override env vars. Default: port 8080, SQLite at `./data/gantry.db`, admin/changeme.

## API Patterns

REST endpoints: `/api/v1/entities`, `/api/v1/entities/{kind}`, `/api/v1/entities/{kind}/{name}`.
Actions: `/api/v1/actions/{name}/execute`, `/api/v1/actions/{name}/runs/{id}`.
Auth: `POST /api/v1/auth/login`, `GET /api/v1/auth/me`.
Search: `GET /api/v1/search?q=`.
WebSocket: `GET /api/v1/ws`.
Health: `/healthz`, `/readyz`.
