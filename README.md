<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="web/public/logo-white.png">
  <source media="(prefers-color-scheme: light)" srcset="web/public/logo-black.png">
  <img alt="Gantry" src="web/public/logo-black.png" width="280">
</picture>

<br />
<br />

**The open-source internal developer platform that ships as a single binary.**

<p>
  <a href="https://github.com/Go2Engle/Gantry/releases/latest">
    <img alt="GitHub release" src="https://img.shields.io/github/v/release/Go2Engle/Gantry?style=flat-square&logo=github&color=6366f1&labelColor=1e1b4b">
  </a>
  <a href="https://github.com/Go2Engle/Gantry/blob/main/LICENSE">
    <img alt="License: Apache 2.0" src="https://img.shields.io/badge/license-Apache%202.0-6366f1?style=flat-square&labelColor=1e1b4b">
  </a>
  <a href="https://go.dev/">
    <img alt="Go 1.22+" src="https://img.shields.io/badge/go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white&labelColor=1e1b4b">
  </a>
  <a href="https://github.com/Go2Engle/Gantry/actions">
    <img alt="Build" src="https://img.shields.io/github/actions/workflow/status/Go2Engle/Gantry/release.yml?style=flat-square&logo=github-actions&logoColor=white&label=build&labelColor=1e1b4b">
  </a>
  <a href="https://github.com/Go2Engle/Gantry/stargazers">
    <img alt="GitHub Stars" src="https://img.shields.io/github/stars/Go2Engle/Gantry?style=flat-square&logo=github&color=f59e0b&labelColor=1e1b4b">
  </a>
  <a href="https://github.com/Go2Engle/Gantry/pkgs/container/gantry">
    <img alt="Docker" src="https://img.shields.io/badge/docker-ghcr.io-2496ED?style=flat-square&logo=docker&logoColor=white&labelColor=1e1b4b">
  </a>
</p>

<p>
  <a href="https://go2engle.com/Gantry/docs/">📖 Documentation</a>
  &nbsp;·&nbsp;
  <a href="https://github.com/Go2Engle/Gantry/releases">🚀 Releases</a>
  &nbsp;·&nbsp;
  <a href="https://github.com/Go2Engle/Gantry/issues/new">🐛 Report a Bug</a>
  &nbsp;·&nbsp;
  <a href="https://github.com/Go2Engle/Gantry/issues/new">✨ Request a Feature</a>
</p>

</div>

---

## What is Gantry?

Gantry is a self-hostable **internal developer platform (IDP)** — a lightweight alternative to Backstage that runs as a single Go binary with no external dependencies. It gives engineering teams a unified service catalog, self-service actions, GitOps-native configuration management, and a plugin ecosystem without the operational overhead.

| | Gantry | Backstage |
|---|---|---|
| Setup time | ~5 minutes | Hours to days |
| Runtime dependencies | **None** — single binary | Node.js, PostgreSQL, often Kubernetes |
| Hosting | Any server or Docker | Kubernetes recommended |
| Embedded database | SQLite, zero config | External DB required |
| GitOps apply | `gantry apply` built-in | YAML ingestion via plugins |

## Features

- **Service Catalog** — Typed, validated, and searchable entities for every service, API, team, and infrastructure component in your org
- **Self-Service Actions** — Schema-driven forms so developers can trigger deployments or run workflows without opening a ticket
- **Plugin Ecosystem** — First-party integrations for [Kubernetes](https://go2engle.com/Gantry/docs/plugins/kubernetes), [GitHub](https://go2engle.com/Gantry/docs/plugins/github), [ArgoCD](https://go2engle.com/Gantry/docs/plugins/argocd), Status Monitor, and Microsoft Teams
- **GitOps Native** — Manage your catalog as YAML with `gantry apply`; diff, review, and roll back like code
- **Full-Text Search** — Find any entity in milliseconds via SQLite FTS5 — no Elasticsearch required
- **Single Binary** — CGO-free Go binary with an embedded React frontend, SQLite database, and all assets baked in
- **API Keys & JWT Auth** — Browser sessions + Bearer token auth for CLI and automation; GitHub OAuth SSO optional
- **Audit Log** — Every write operation is captured with before/after state and source IP
- **Prometheus Metrics** — Built-in `/metrics` endpoint, no extra instrumentation needed

---

## Quick Start

### Install Script (Linux / macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/go2engle/gantry/main/install.sh | sh
gantry serve
```

### Docker

```bash
docker run -p 8080:8080 ghcr.io/go2engle/gantry:latest
```

### Build from Source

```bash
git clone https://github.com/Go2Engle/Gantry.git && cd Gantry
make build
bin/gantry serve --dev
```

Open **http://localhost:8080** — default login: `admin` / `changeme`

> Full installation options: [go2engle.com/Gantry/docs/getting-started/installation](https://go2engle.com/Gantry/docs/getting-started/installation)

---

## CLI

The `gantry` binary is both server and client — think `kubectl` but for your developer platform.

```bash
# Start the server
gantry serve
gantry serve --port 9090 --dev
gantry serve --admin-password mysecret

# Apply entities from YAML — GitOps style
gantry apply -f services.yaml
gantry apply -f ./catalog/

# Query the catalog
gantry get services
gantry get Service payments-api -o yaml
gantry describe Service payments-api

# Check version / manage installation
gantry version
gantry upgrade
gantry uninstall
```

---

## Configuration

All options can be set via environment variable, CLI flag, or a YAML config file.

| Env Var | Flag | Default | Description |
|---------|------|---------|-------------|
| `GANTRY_PORT` | `--port` | `8080` | HTTP listen port |
| `GANTRY_DB` | `--db` | `./data/gantry.db` | SQLite path or `postgres://` URL |
| `GANTRY_DEV` | `--dev` | `false` | Permissive CORS + verbose logging |
| `GANTRY_ADMIN_PASSWORD` | `--admin-password` | `changeme` | Initial admin password |
| `GANTRY_JWT_SECRET` | — | auto-generated | JWT signing secret |
| `GANTRY_DATA_DIR` | — | `./data` | Data directory |
| `GANTRY_ENCRYPTION_KEY` | — | auto-generated | AES-256-GCM key for plugin secrets |

> Full configuration reference: [go2engle.com/Gantry/docs/getting-started/configuration](https://go2engle.com/Gantry/docs/getting-started/configuration)

---

## API Reference

Auth uses `Authorization: Bearer <token>` for CLI/API clients, or an HttpOnly session cookie for browsers.

```bash
# Authenticate
curl -X POST localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"changeme"}'

# Create a catalog entity
curl -X POST localhost:8080/api/v1/entities \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "kind": "Service",
    "metadata": { "name": "payments-api", "title": "Payments API", "owner": "team-payments" },
    "spec": { "type": "backend", "lifecycle": "production" }
  }'

# Search the catalog
curl "localhost:8080/api/v1/search?q=payments" -H "Authorization: Bearer $TOKEN"
```

<details>
<summary><strong>Full endpoint reference</strong></summary>

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Health check |
| `GET` | `/readyz` | Readiness check |
| `GET` | `/metrics` | Prometheus metrics |
| `POST` | `/api/v1/auth/login` | Login, returns JWT |
| `POST` | `/api/v1/auth/logout` | Clear session cookie |
| `GET` | `/api/v1/auth/me` | Current user info |
| `GET` | `/api/v1/apikeys` | List API keys |
| `POST` | `/api/v1/apikeys` | Create a scoped API key |
| `DELETE` | `/api/v1/apikeys/{id}` | Revoke an API key |
| `GET` | `/api/v1/entities` | List all entities |
| `GET` | `/api/v1/entities/{kind}` | List by kind |
| `GET` | `/api/v1/entities/{kind}/{name}` | Get entity |
| `POST` | `/api/v1/entities` | Create entity |
| `PUT` | `/api/v1/entities/{kind}/{name}` | Update entity |
| `DELETE` | `/api/v1/entities/{kind}/{name}` | Delete entity |
| `GET` | `/api/v1/search?q=` | Full-text search |
| `GET` | `/api/v1/schemas/{kind}` | JSON Schema for a kind |
| `GET` | `/api/v1/plugins` | List plugins and enabled state |
| `POST` | `/api/v1/actions/{name}/execute` | Execute an action |
| `GET` | `/api/v1/actions/{name}/runs` | List action runs |
| `GET` | `/api/v1/audit` | Audit log |
| `GET` | `/api/v1/graph/{kind}/{name}` | Entity dependency graph |
| `GET` | `/api/v1/ws` | WebSocket (real-time events) |

</details>

> Full API documentation: [go2engle.com/Gantry/docs/api](https://go2engle.com/Gantry/docs/api)

---

## Development

**Prerequisites:** Go 1.22+, Node.js 18+, npm

```bash
# Recommended: live reload for both backend and frontend
go install github.com/air-verse/air@latest
make dev-watch
# → Backend auto-rebuilds on .go changes (air)
# → Frontend hot-reloads at http://localhost:3000 (Vite HMR)

# Or run them separately:
bin/gantry serve --dev          # Go server on :8080
cd web && npm run dev           # Vite dev server on :3000

# Tests & checks
go test ./...
cd web && npx tsc --noEmit
golangci-lint run ./...
```

> Contributing guide: [go2engle.com/Gantry/docs/contributing](https://go2engle.com/Gantry/docs/contributing/overview)

---

## Star History

<div align="center">

[![Star History Chart](https://api.star-history.com/svg?repos=Go2Engle/Gantry&type=Date)](https://star-history.com/#Go2Engle/Gantry&Date)

</div>

---

## License

Apache 2.0 — see [LICENSE](LICENSE).

<div align="center">

If Gantry is useful to you, give it a ⭐ — it helps others find it.

</div>
