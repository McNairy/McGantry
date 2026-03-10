---
sidebar_position: 2
title: Development Setup
description: Get a full Gantry development environment running locally.
---

# Development Setup

## Prerequisites

| Tool | Version | Notes |
|---|---|---|
| Go | 1.22+ | `go version` |
| Node.js | 20+ | `node --version` |
| npm | 9+ | Comes with Node |
| Make | any | Optional, see Makefile |
| golangci-lint | 1.57+ | For Go linting |
| air | latest | Optional, for hot reload |

### Installing Prerequisites

```bash
# Go — https://go.dev/dl/
# macOS with Homebrew:
brew install go

# Node.js — https://nodejs.org/
# or with nvm:
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
nvm install 20

# golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.57.2

# air (live reload for Go)
go install github.com/air-verse/air@latest
```

## Clone and Build

```bash
git clone https://github.com/go2engle/gantry.git
cd gantry

# Install frontend dependencies
cd web && npm install && cd ..

# Build frontend (required for backend to serve the SPA)
cd web && npm run build && cd ..

# Build Go binary
go build -o bin/gantry ./cmd/gantry

# Verify
./bin/gantry version
```

## Running in Development

### Option A: Backend + Pre-built Frontend

```bash
./bin/gantry serve --dev
```

Opens at [http://localhost:8080](http://localhost:8080).

### Option B: Hot Reload (Recommended)

Terminal 1 — Backend with live reload:

```bash
# Using air (watches Go files and rebuilds automatically)
air
# or without air:
go run ./cmd/gantry serve --dev
```

Terminal 2 — Frontend dev server:

```bash
cd web
npm run dev
```

Frontend dev server runs at [http://localhost:5173](http://localhost:5173) and proxies API calls to `:8080`. Changes to React files hot-reload instantly.

### Option C: Full Rebuild

```bash
make build
bin/gantry serve --dev
```

The `Makefile` runs frontend build then Go build.

## Project Structure

```
gantry/
├── cmd/gantry/           # CLI entry point (cobra commands)
│   ├── main.go
│   ├── serve.go          # gantry serve
│   ├── apply.go          # gantry apply
│   ├── get.go            # gantry get
│   ├── describe.go       # gantry describe
│   ├── export.go         # gantry export
│   ├── run.go            # gantry run
│   └── version.go        # gantry version
│
├── internal/
│   ├── api/              # HTTP server + routes
│   │   ├── server.go     # chi router setup, middleware wiring
│   │   ├── handlers/     # HTTP request handlers
│   │   │   ├── entities.go
│   │   │   ├── auth.go
│   │   │   ├── actions.go
│   │   │   ├── audit.go
│   │   │   ├── graph.go
│   │   │   ├── plugins.go
│   │   │   ├── search.go
│   │   │   ├── schemas.go
│   │   │   ├── history.go
│   │   │   └── dashboard.go
│   │   ├── middleware/   # JWT/API key auth, request logging
│   │   └── websocket/    # WebSocket hub, channel subscriptions
│   │
│   ├── auth/             # Password hashing (bcrypt) + JWT lifecycle
│   ├── config/           # Config loading (env + YAML + defaults)
│   ├── crypto/           # AES-256-GCM encryption helpers
│   ├── db/               # Database layer
│   │   ├── db.go         # DB connection + helper methods
│   │   ├── migrations.go # Schema migrations (idempotent)
│   │   └── queries.go    # Entity CRUD + all DB queries
│   ├── entity/           # Entity types + JSON Schema validation
│   │   ├── entity.go     # Entity struct, marshaling
│   │   ├── validator.go  # JSON Schema validator (go:embed)
│   │   └── schemas/      # JSON Schema files for each kind
│   │       ├── service.json
│   │       ├── api.json
│   │       ├── team.json
│   │       ├── environment.json
│   │       ├── infrastructure.json
│   │       ├── action.json
│   │       └── documentation.json
│   ├── events/           # In-process pub/sub event bus
│   ├── metrics/          # Custom Prometheus text format metrics
│   ├── plugins/          # Plugin manifest types + bundled registry
│   │   ├── manifest.go   # Plugin/RegistryEntry structs
│   │   ├── registry.go   # BundledRegistry() + registry helpers
│   │   ├── bundled/
│   │   │   └── registry.json
│   │   ├── kubernetes/   # Kubernetes plugin implementation
│   │   ├── github/       # GitHub plugin implementation
│   │   └── argocd/       # ArgoCD plugin implementation
│   └── search/           # FTS5 search service
│
├── web/                  # React SPA
│   ├── src/
│   │   ├── pages/        # Page components
│   │   ├── components/   # Shared UI components
│   │   ├── lib/
│   │   │   ├── api.ts    # API client (all endpoints)
│   │   │   ├── types.ts  # TypeScript types
│   │   │   └── plugin-runtime.ts  # Plugin extension registry
│   │   └── App.tsx       # Root component + routing
│   ├── package.json
│   └── vite.config.ts    # Vite config (proxies /api → :8080)
│
├── website/              # Marketing/docs website
│   ├── index.html
│   ├── assets/
│   └── docs/             # Docusaurus documentation (this site)
│
├── .github/
│   └── workflows/        # CI/CD workflows
│
├── .goreleaser.yaml      # Multi-platform binary + Docker release
├── Makefile
├── go.mod
└── CLAUDE.md             # Instructions for AI contributors
```

## Running Tests

```bash
# All Go tests
go test ./...

# With verbose output
go test ./... -v

# Specific test
go test ./internal/db/... -run TestEntityCreate -v

# With race detector (recommended before PRs)
go test -race ./...

# TypeScript type check
cd web && npx tsc --noEmit
```

## Linting

```bash
# Go
golangci-lint run ./...

# Frontend (if ESLint is configured)
cd web && npm run lint
```

## Database Reset

To start fresh during development:

```bash
rm -f ./data/gantry.db ./data/encryption.key
./bin/gantry serve --dev
```

## Adding a New Entity Kind

1. **Create the JSON Schema** in `internal/entity/schemas/{kind}.json` — follow the existing schemas as templates, use `"additionalProperties": false`
2. **Register the kind** in `internal/entity/validator.go` (add to the `knownKinds` or embed list)
3. **Test validation** — add test cases in `internal/entity/validator_test.go`
4. The frontend's `SchemaForm` component auto-generates forms from the JSON Schema — no frontend changes needed for basic kinds

## Adding a New API Endpoint

1. **Write the handler** in `internal/api/handlers/` as a method on `handlers.Handlers`
2. **Register the route** in `internal/api/server.go` under the appropriate middleware group
3. **Add role check** if the endpoint requires a specific role — use the `requireRole(role)` middleware
4. **Audit log** — write an audit entry for any mutations using `h.db.InsertAuditEntry()`
5. **Write a test** in `internal/api/handlers/`

## Environment for Tests

Tests use a real SQLite in-memory database. No mocking of the DB layer.

```go
func TestEntityCreate(t *testing.T) {
    db := setupTestDB(t) // creates in-memory SQLite
    h := handlers.New(db, ...)
    // ...
}
```
