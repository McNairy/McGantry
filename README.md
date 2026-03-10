# Gantry

**The Developer Platform That Just Works**

Gantry is an open-source internal developer platform (IDP) that provides a unified service catalog, self-service portal, and GitOps-native configuration management — all in a single binary.

## Quick Start

```bash
# Build everything (frontend + backend)
make build

# Start the server
bin/gantry serve --dev

# Open http://localhost:8080
# Login: admin / changeme
```

## Prerequisites

- **Go 1.22+**
- **Node.js 18+** and npm

## Development

### Backend

```bash
# Build the Go binary
go build -o bin/gantry ./cmd/gantry

# Run the server in dev mode
bin/gantry serve --dev

# Run all Go tests
go test ./...

# Run a specific test
go test ./internal/db -run TestCreateEntity -v

# Lint (requires golangci-lint)
golangci-lint run ./...

# Format
gofmt -w .
```

### Frontend

The React app lives in `web/`. In dev mode, Vite proxies API requests to the Go server on port 8080.

```bash
cd web

# Install dependencies
npm install

# Dev server with hot reload (port 3000, proxies /api to :8080)
npm run dev

# Type check
npx tsc --noEmit

# Production build (outputs to web/dist/)
npm run build
```

### Live Reload (recommended for development)

For the fastest feedback loop, use `air` for Go hot-reload combined with Vite's HMR for the frontend — no manual rebuilds needed.

**One-time setup:**

```bash
go install github.com/air-verse/air@latest
```

**Then just run:**

```bash
make dev-watch
```

This starts both processes together:
- **Backend** (`air`) — watches `*.go` files, rebuilds and restarts automatically on save
- **Frontend** (Vite) — hot module replacement at `http://localhost:3000`, proxies `/api` to `:8080`

Open `http://localhost:3000` in your browser. `Ctrl+C` stops both processes.

### Full Build

```bash
# Frontend + backend in one step
make build

# Clean build artifacts
make clean
```

## CLI

The `gantry` binary doubles as both server and client (like `kubectl`):

```bash
# Start the server
gantry serve
gantry serve --port 9090 --dev
gantry serve --db postgres://user:pass@host/gantry

# Apply entities from YAML
gantry apply -f services.yaml

# List entities
gantry get services
gantry get Service payments-api -o yaml

# Describe an entity
gantry describe Service payments-api

# Version
gantry version
```

## Configuration

| Env Var | Flag | Default | Description |
|---------|------|---------|-------------|
| `GANTRY_PORT` | `--port` | `8080` | HTTP listen port |
| `GANTRY_DB` | `--db` | `sqlite (./data/gantry.db)` | Database connection string |
| `GANTRY_DEV` | `--dev` | `false` | Development mode (permissive CORS, verbose logging) |
| `GANTRY_ADMIN_PASSWORD` | `--admin-password` | `changeme` | Initial admin password |
| `GANTRY_JWT_SECRET` | — | auto-generated | JWT signing secret |
| `GANTRY_DATA_DIR` | — | `./data` | Data directory for SQLite |

Prefix a DB string with `postgres://` for PostgreSQL; otherwise it's treated as a SQLite file path.

## API

All endpoints under `/api/v1/` require a Bearer token (obtained from login), except `/healthz`, `/readyz`, and `/api/v1/auth/login`.

```bash
# Login
curl -X POST localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"changeme"}'

# Create an entity
curl -X POST localhost:8080/api/v1/entities \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "kind": "Service",
    "metadata": {
      "name": "payments-api",
      "title": "Payments API",
      "owner": "team-payments",
      "tags": ["backend", "go"]
    },
    "spec": {
      "type": "backend",
      "lifecycle": "production"
    }
  }'

# List entities
curl localhost:8080/api/v1/entities -H "Authorization: Bearer $TOKEN"

# Search
curl "localhost:8080/api/v1/search?q=payments" -H "Authorization: Bearer $TOKEN"
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Health check |
| `GET` | `/readyz` | Readiness check |
| `POST` | `/api/v1/auth/login` | Login, returns JWT |
| `GET` | `/api/v1/auth/me` | Current user info |
| `GET` | `/api/v1/entities` | List all entities |
| `GET` | `/api/v1/entities/{kind}` | List by kind |
| `GET` | `/api/v1/entities/{kind}/{name}` | Get entity |
| `POST` | `/api/v1/entities` | Create entity |
| `PUT` | `/api/v1/entities/{kind}/{name}` | Update entity |
| `DELETE` | `/api/v1/entities/{kind}/{name}` | Delete entity |
| `GET` | `/api/v1/search?q=` | Full-text search |
| `GET` | `/api/v1/schemas` | List all JSON schemas |
| `GET` | `/api/v1/schemas/{kind}` | Get schema for kind |
| `POST` | `/api/v1/actions/{name}/execute` | Execute an action |
| `GET` | `/api/v1/actions/{name}/runs` | List action runs |
| `GET` | `/api/v1/audit` | Audit log |
| `GET` | `/api/v1/ws` | WebSocket (real-time events) |

## Project Structure

```
cmd/gantry/          CLI entry point (serve, apply, get, describe, version)
internal/
  api/               HTTP server and routes
    handlers/        REST handler functions
    middleware/      Auth (JWT) and logging middleware
    websocket/       WebSocket hub
  auth/              Password hashing and JWT tokens
  config/            Configuration loading
  db/                Database layer, migrations, queries
  entity/            Entity types, built-in kinds, JSON Schema validation
    schemas/         JSON Schema files for each entity kind
  events/            In-process event bus
  search/            Full-text search (SQLite FTS5)
web/                 React frontend (Vite + Tailwind)
  src/
    components/      Sidebar, CommandPalette, EntityCard, EntityTable, SchemaForm
    hooks/           useAuth, useTheme
    lib/             API client, types
    pages/           Dashboard, Catalog, EntityDetail, Actions, Settings, Login
```

## License

Apache 2.0
