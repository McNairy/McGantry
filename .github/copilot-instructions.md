# GitHub Copilot Instructions for Gantry

Gantry is an open-source internal developer platform (IDP) — a self-hostable alternative to Backstage that deploys as a single Go binary with an embedded React frontend. Apache 2.0 licensed.

## Tech Stack

- **Backend:** Go 1.22, chi router, modernc.org/sqlite (pure Go, no CGO required)
- **Frontend:** React 18, Vite, Tailwind CSS, lucide-react icons, TypeScript
- **Auth:** bcrypt + JWT (golang-jwt/jwt/v5), API keys (`gantry_<hex>` format), GitHub OAuth SSO
- **Real-time:** gorilla/websocket
- **Encryption:** AES-256-GCM via `internal/crypto/` for plugin config secrets
- **CLI:** spf13/cobra

## Project Structure

```
cmd/gantry/          CLI entry point (serve, apply, get, describe, export, run, version)
internal/
  api/
    handlers/        REST handlers — all methods on handlers.Handlers struct
    middleware/      JWT + API key auth, request logging, metrics
    websocket/       WebSocket hub with channel subscriptions
  auth/              bcrypt password hashing, JWT lifecycle
  config/            Env var + YAML config loading
  crypto/            AES-256-GCM encryption helpers
  db/                SQLite queries (no ORM), migrations
  dispatcher/        Action dispatchers (webhook, etc.)
  entity/            Entity types, JSON Schema validation (schemas in internal/entity/schemas/)
  events/            In-process pub/sub event bus
  metrics/           Custom Prometheus text format (no external dep)
  plugins/           Plugin system: bundled registry, Kubernetes/GitHub/ArgoCD integrations
  search/            SQLite FTS5 full-text search
web/src/
  pages/             Dashboard, Catalog, EntityDetail, Actions, AuditLog, Settings, Login, Plugins, Users
  components/        Sidebar, SchemaForm, EntityGraph, GitHubTab, KubernetesTab, ArgoCDTab, etc.
  hooks/             useAuth, useTheme
  lib/
    api.ts           Typed API client (all endpoints, Bearer token auth)
    types.ts         TypeScript types for all domain objects
    plugin-runtime.ts Plugin extension point registry
  styles/globals.css Global styles + markdown rendering
website/             Docusaurus documentation site (separate from the app)
```

## Key Conventions

### Go Backend

- **No ORM.** Use `database/sql` directly. JSON fields (`tags`, `annotations`, `labels`, `spec`) are stored as TEXT and marshaled with `encoding/json`.
- **Handlers** are methods on `handlers.Handlers` struct which holds DB, config, and service dependencies.
- **DB helper methods:** use `d.exec()`, `d.queryRow()`, `d.queryRows()` — receiver is `*DB`, not `db.sql.*`.
- **Audit logging:** always capture `IPAddress` (use `clientIP` helper) and `BeforeState`/`AfterState`.
- **Boolean in SQLite:** use `boolToInt()` helper since SQLite has no native bool.
- **Plugin configs** with secret fields are encrypted at rest. Use `internal/crypto/` encrypt/decrypt helpers.
- **Metrics:** add to `internal/metrics/` using the custom text format — do NOT add prometheus/client_golang.
- **No CGO.** The project builds with `CGO_ENABLED=0`. Do not add dependencies that require CGO.

### Entity Model

Everything is an entity: `kind`, `apiVersion`, `metadata` (name, namespace, labels, annotations, tags), `spec`.

Built-in kinds: `Service`, `API`, `Infrastructure`, `Team`, `Environment`, `Documentation`, `Action`.

Each kind has a JSON Schema in `internal/entity/schemas/`. All built-in schemas use `"additionalProperties": false` — always check allowed spec fields before adding new ones to sync code or entity creation.

Entity spec fields of interest:
- `repoUrl` (Service, API, Infrastructure) — used to auto-activate the GitHub tab
- `links` array — `{title, url, icon?}` where icon is one of: `dashboard/docs/runbook/github/slack/alert/monitor/ci/other`
- `deployedIn` — namespace string used by the Kubernetes tab to find pods

### Frontend

- **CSS variables:** only use defined properties: `--gantry-bg-primary`, `--gantry-bg-secondary`, `--gantry-bg-tertiary`, `--gantry-text-primary`, `--gantry-text-secondary`, `--gantry-border`, `--gantry-accent`, `--gantry-accent-hover`, `--gantry-danger`. Others will be invisible.
- **Tailwind opacity:** use `bg-[var(--gantry-accent)]/10` NOT `bg-[var(--gantry-accent)] bg-opacity-10`.
- **Dark mode:** toggled via `.dark` class on `<html>`.
- **API calls:** always go through the `api` object in `web/src/lib/api.ts`.
- **SchemaForm:** auto-generates forms from JSON Schema — used for entity create/edit and action execution.
- **Plugin tabs:** `GitHubTab` is shown when entity has `spec.repoUrl` containing `github.com`. `KubernetesTab` / `ArgoCDTab` are shown when the respective plugin is enabled and entity has relevant annotations.

### API Patterns

- REST: `GET/POST /api/v1/entities`, `GET/PUT/DELETE /api/v1/entities/{kind}/{name}`
- Auth: `POST /api/v1/auth/login`, `GET /api/v1/auth/me`, `GET /api/v1/auth/github` (OAuth)
- Search: `GET /api/v1/search?q=`
- WebSocket: `GET /api/v1/ws`
- Metrics: `GET /metrics`
- Health: `/healthz`, `/readyz`
- Plugins: `GET/POST /api/v1/plugins`, `GET/PUT/DELETE /api/v1/plugins/{name}`, `POST /api/v1/plugins/{name}/enable|disable|sync`
- API Keys: `GET/POST /api/v1/apikeys`, `DELETE /api/v1/apikeys/{id}`
- Graph: `GET /api/v1/graph/{kind}/{name}`
- History: `GET/POST /api/v1/history`

### Config

Env vars: `GANTRY_PORT`, `GANTRY_DB`, `GANTRY_DEV`, `GANTRY_ADMIN_PASSWORD`, `GANTRY_JWT_SECRET`, `GANTRY_DATA_DIR`, `GANTRY_ENCRYPTION_KEY`. Also configurable via YAML file with camelCase keys.

## Build Commands

```bash
make build          # full build (frontend + backend)
make dev            # run backend in dev mode
make dev-watch      # air (backend live reload) + vite HMR — requires air
go test ./...       # run all Go tests
cd web && npm run dev         # frontend dev server (proxies API to :8080)
cd web && npx tsc --noEmit    # TypeScript type check
golangci-lint run ./...       # lint Go
```

## Creating Plugins

Gantry plugins follow a consistent pattern across up to 11 files. Use `status-monitor` as a reference implementation.

### Files to Create or Modify

1. **`internal/api/handlers/<plugin>.go`** — Create backend handler(s) as methods on `*Handlers` struct. Check plugin installed+enabled via `h.DB.GetPlugin()`. For external HTTP calls, always set `User-Agent` header. Use in-memory cache (`sync.RWMutex` + TTL) for expensive operations.
2. **`internal/plugins/bundled/registry.json`** — Add entry with ALL fields: `name`, `title`, `description`, `longDescription`, `features`, `version`, `author`, `category`, `homepage`, `configSchema` (JSON Schema), optional `entityPanels` and `actionTypes`.
3. **`internal/api/server.go`** — Register routes in `protected` group, BEFORE wildcard/SPA routes.
4. **`web/src/lib/types.ts`** — Add TypeScript interfaces for new API response types.
5. **`web/src/lib/api.ts`** — Add API client methods to the `api` object.
6. **`web/src/pages/<PluginPage>.tsx`** — Full-page UI if plugin has sidebar entry.
7. **`web/src/App.tsx`** — Add `<Route>` for new page.
8. **`web/src/components/Sidebar.tsx`** — Conditional nav item (check plugin enabled via `api.listPlugins()`).
9. **`web/src/pages/Dashboard.tsx`** — Dashboard widget: add to `WIDGET_LABELS`, `DEFAULT_WIDGETS`, `renderWidget()` switch, and fetch data in `useEffect`.
10. **`internal/api/handlers/dashboard.go`** — Add widget ID to `knownWidgetIDs` map (required for save validation).
11. **`web/src/components/<PluginTab>.tsx`** — Entity detail tab if plugin adds entity panels.

### Common Pitfalls

- **CSS variables:** only 9 `--gantry-*` properties exist. Using undefined ones → invisible elements.
- **Tailwind opacity:** use `bg-[var(--gantry-accent)]/10` NOT `bg-[var(--gantry-accent)] bg-opacity-10`.
- **Dark mode contrast:** use `text-[var(--gantry-bg-primary)]` on accent backgrounds, NOT `text-white`.
- **Entity schemas:** all have `additionalProperties: false` — check allowed spec fields first.
- **External HTTP APIs:** set `User-Agent` header; some providers block default Go client.
- **Widget ID validation:** missing ID in `knownWidgetIDs` causes dashboard save failures.
- **Route ordering:** plugin routes must come before wildcard routes in `server.go`.
- **Registry JSON:** missing `configSchema`/`entityPanels`/`actionTypes` fields cause parse errors.

### Verify After Changes

```bash
go build ./...              # backend compiles
cd web && npx tsc --noEmit  # frontend type-checks
cd web && npm run build     # frontend builds
```
