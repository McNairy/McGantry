---
sidebar_position: 3
title: Architecture
description: How Gantry is structured internally.
---

# Architecture

Gantry is a **modular monolith** — all functionality runs in a single Go process with clear internal package boundaries.

## High-Level Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                         gantry serve                          │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │                    HTTP Server (chi)                    │  │
│  │  /api/v1/*  ·  /healthz  ·  /readyz  ·  /metrics      │  │
│  │  SPA fallback (web/dist/**)                            │  │
│  └────────────────────────────────────────────────────────┘  │
│            │               │               │                 │
│         Handlers        WebSocket        Metrics             │
│            │               │                                 │
│  ┌─────────▼───────────────▼──────────────────────────────┐  │
│  │                    handlers.Handlers                    │  │
│  │  (entities, auth, actions, plugins, search, graph ...)  │  │
│  └──────────┬──────────────────────────┬───────────────────┘  │
│             │                          │                       │
│  ┌──────────▼──────────┐   ┌───────────▼───────────────────┐  │
│  │    db.DB (SQLite/   │   │       Event Bus (pub/sub)      │  │
│  │      Postgres)      │   │   entity.created / .updated   │  │
│  └─────────────────────┘   └───────────────────────────────┘  │
│                                        │                       │
│                            ┌───────────▼───────────────────┐  │
│                            │    WebSocket Hub              │  │
│                            │  (broadcast to subscribers)   │  │
│                            └───────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

## Request Lifecycle

A typical entity create request flows through:

```
HTTP POST /api/v1/entities
    ↓
chi router
    ↓
RequestID middleware (adds X-Request-ID)
    ↓
RealIP middleware (sets client IP from X-Forwarded-For)
    ↓
RequestLogger middleware (structured log with latency)
    ↓
Auth middleware (validates JWT or API key, injects claims into ctx)
    ↓
RequireRole middleware (checks minimum role)
    ↓
handlers.CreateEntity()
    ↓
entity.SchemaValidator.Validate(kind, spec JSON)
    ↓
db.CreateEntity() (INSERT with conflict check)
    ↓
db.InsertAuditEntry() (audit log)
    ↓
events.Bus.Publish("entity.created", entity)
    ↓
WebSocket hub broadcasts to subscribers
    ↓
JSON response 201 Created
```

## Package Responsibilities

### `cmd/gantry/`

CLI entry point using [cobra](https://github.com/spf13/cobra). Each subcommand is a separate file. Commands that talk to the server (`get`, `apply`, `export`, `describe`, `run`) use shared helpers: `getToken()`, `doRequest()`, `readBody()`, `printYAML()`, `printJSON()`.

**Key rule:** CLI commands do NOT import `internal/` packages directly. They communicate via HTTP to a running server. This keeps the CLI portable (no CGO, no SQLite dependency in the CLI binary).

### `internal/api/`

Sets up the chi router, wires all middleware, registers routes. The `server.go` file is the composition root — it instantiates `handlers.Handlers` and connects everything.

**Routes are grouped by auth level:**
- Public routes (no auth middleware)
- Authenticated routes (auth middleware only)
- Role-specific sub-groups (auth + role check)

### `internal/api/handlers/`

HTTP handlers are methods on the `handlers.Handlers` struct, which holds all service dependencies (DB, validator, event bus, plugin manager, etc.).

Pattern:
```go
type Handlers struct {
    db        *db.DB
    validator *entity.SchemaValidator
    events    *events.Bus
    plugins   *plugins.Manager
    // ...
}

func (h *Handlers) CreateEntity(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request body
    // 2. Validate schema
    // 3. Write to DB
    // 4. Write audit entry
    // 5. Publish event
    // 6. Write JSON response
}
```

### `internal/api/middleware/`

- **auth.go** — Extracts and validates JWT or API key from `Authorization` header. Injects `auth.Claims` into `r.Context()`. Checks `strings.HasPrefix(token, "gantry_")` to distinguish API keys from JWTs.
- **logging.go** — Structured JSON request logs with method, path, status, latency, request ID.

### `internal/auth/`

Stateless auth utilities:
- `HashPassword(plain)` → bcrypt hash
- `CheckPassword(hash, plain)` → error or nil
- `GenerateToken(claims, secret)` → JWT string
- `ValidateToken(token, secret)` → claims or error

### `internal/db/`

Direct `database/sql` queries — no ORM. The `DB` struct wraps `*sql.DB` with helper methods:

```go
func (d *DB) exec(query string, args ...any) error
func (d *DB) queryRow(dest any, query string, args ...any) error
func (d *DB) queryRows(query string, args ...any) (*sql.Rows, error)
```

JSON fields (`tags`, `annotations`, `labels`, `spec`) are stored as `TEXT` and serialized/deserialized with `encoding/json`. The boolean `enabled` in the plugins table uses `boolToInt()` / integer-to-bool conversion (SQLite doesn't have native booleans).

**Migrations** in `migrations.go` are idempotent (`CREATE TABLE IF NOT EXISTS`, `INSERT OR IGNORE`). They run on every server startup.

### `internal/entity/`

The `SchemaValidator` uses `go:embed` to load JSON Schemas from `internal/entity/schemas/` into memory at startup. Validation uses [santhosh-tekuri/jsonschema/v5](https://github.com/santhosh-tekuri/jsonschema).

All built-in schemas have `"additionalProperties": false` — any unknown field in `spec` causes a validation error. This is intentional.

### `internal/events/`

Simple in-process pub/sub. Publishers call `bus.Publish(topic, data)`. Subscribers receive via channel. The WebSocket hub subscribes to entity events and broadcasts to connected clients.

No external message broker. Events are ephemeral — they're not persisted to the DB.

### `internal/plugins/`

Manages the plugin lifecycle. The `Manager` struct:
- Loads the bundled registry (`bundled/registry.json` via `go:embed`)
- Reads installed plugins from the `plugins` DB table
- Handles config encryption/decryption
- Dispatches HTTP requests to plugin-specific handlers

Each plugin is a sub-package with its own handler registrations:
- `plugins/kubernetes/` — sync, workload, log streaming handlers
- `plugins/github/` — repo sync, live info, OAuth handlers
- `plugins/argocd/` — app discovery, sync, refresh handlers

### `internal/metrics/`

Custom Prometheus text format — no external Prometheus client library. Uses atomic counters and gauges exported via the `/metrics` endpoint. Keeps the binary lean.

### `web/`

React 18 + Vite + Tailwind CSS. Key conventions:

- All API calls go through the `api` object in `web/src/lib/api.ts`
- TypeScript types mirror Go structs in `web/src/lib/types.ts`
- Theming uses CSS custom properties (`var(--gantry-*)`) — see below
- Dark mode via `.dark` class on `<html>` (toggled by `ThemeToggle`)
- Plugin extension points registered in `plugin-runtime.ts`

**CSS Custom Properties (all defined variables):**

```css
--gantry-bg-primary      /* main background */
--gantry-bg-secondary    /* card/panel background */
--gantry-bg-tertiary     /* input/hover background */
--gantry-text-primary    /* primary text */
--gantry-text-secondary  /* secondary/muted text */
--gantry-border          /* border color */
--gantry-accent          /* blue accent */
--gantry-accent-hover    /* blue accent hover state */
--gantry-danger          /* red for destructive actions */
```

:::warning Undefined Variables
Do NOT use `--gantry-surface`, `--gantry-bg`, `--gantry-text`, `--gantry-hover`, or `--gantry-text-muted` — these are not defined and will render as transparent/invisible.
:::

## Database Schema Overview

```
entities         — all catalog entities (kind/name/namespace + JSON fields)
users            — user accounts (username, bcrypt hash, role)
audit_log        — immutable audit trail (who, what, when, before/after)
action_runs      — action execution history
api_keys         — API key hashes (never stored raw)
plugins          — installed plugin state and config
user_history     — per-user recently viewed entities
dashboard_config — single-row dashboard widget config
```

## Key Invariants

1. **Every mutation** creates an audit log entry with `before_state` and `after_state`
2. **API keys** are stored only as SHA-256 hashes — raw keys are never persisted
3. **Plugin configs** are encrypted at rest with AES-256-GCM
4. **Entity schema validation** happens in the handler, before any DB write
5. **Entity names** are unique per `(kind, namespace)` — enforced by DB UNIQUE constraint
6. **Binary size** — no CGO, no external Prometheus, no ORM. Dependencies are chosen for necessity.
