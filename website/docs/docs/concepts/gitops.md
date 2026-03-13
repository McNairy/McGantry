---
sidebar_position: 4
title: GitOps
description: Bidirectional Git sync and catalog-as-code with the GitOps plugin and gantry apply.
---

# GitOps

Gantry is GitOps-native. Your entire service catalog can be defined as YAML files, checked into Git, and synced bidirectionally between Gantry and a Git repository. This means your catalog is version-controlled, reviewable via pull requests, and reproducible.

There are two complementary approaches:

| Approach | Direction | How it works |
|---|---|---|
| **GitOps Plugin** | Bidirectional (push + pull) | Built-in plugin that syncs entity changes to/from a Git repository in real-time |
| **`gantry apply` CLI** | One-way (push to Gantry) | CLI command that reads YAML files and creates/updates entities via the API |

## GitOps Plugin

The GitOps plugin provides automatic bidirectional synchronization between your Gantry catalog and a Git repository. Entity changes made in the UI or API are pushed to Git, and changes made directly in Git (via pull requests) are pulled back into Gantry.

### Setup

1. Go to **Plugins** in the sidebar
2. Find **GitOps**, open it, and configure the plugin:

| Setting | Required | Description |
|---|---|---|
| **Repository URL** | Yes | HTTPS Git URL (e.g., `https://github.com/org/gantry-catalog.git`) |
| **Branch** | No | Branch to sync with (default: `main`) |
| **Personal Access Token** | No | PAT for private repos (HTTPS auth) |
| **Base Path** | No | Subdirectory within the repo for entity files (default: repo root) |
| **Auto-Push** | No | Automatically push entity changes to remote (default: `true`) |
| **Commit Author Name** | No | Git commit author name (default: `Gantry GitOps`) |
| **Commit Author Email** | No | Git commit author email (default: `gantry@localhost`) |
| **Pull Sync Interval** | No | How often to auto-pull from Git (e.g., `5m`, `1h`). Leave empty for manual-only |

3. Click **Enable** to activate the plugin

### How It Works

#### Push (Gantry → Git)

When an entity is created, updated, or deleted through the Gantry UI or API:

1. The change is queued in a 2-second debounce window
2. All queued changes are batched into a single Git commit
3. The commit is pushed to the remote repository (if auto-push is enabled)

This means rapid edits (e.g., bulk imports via `gantry apply`) result in a single commit rather than one per entity.

#### Pull (Git → Gantry)

When changes are made directly in the Git repository (e.g., via a pull request):

1. Gantry fetches the latest changes from the remote
2. YAML files are parsed and compared against the database
3. New entities are created; existing entities are updated with the repo's values
4. Changes made during a pull are suppressed from triggering a push back (no feedback loops)

Pull can be triggered manually or run on a configurable interval.

#### Full Sync

A full sync exports **all** entities from the database to the Git repository. This is useful for:

- Initial setup — populating a new repo with your existing catalog
- Recovery — ensuring the repo reflects the current database state
- Bulk updates — after importing entities via the API

### Repository Layout

The plugin organizes entity files by kind, namespace, and name:

```
<basePath>/
├── Service/
│   └── default/
│       ├── api-gateway.yaml
│       ├── user-service.yaml
│       └── payment-processor.yaml
├── Team/
│   └── default/
│       ├── platform-engineering.yaml
│       └── backend-services.yaml
├── Infrastructure/
│   └── default/
│       ├── postgres-primary.yaml
│       └── redis-cache.yaml
└── Environment/
    └── default/
        ├── k8s-prod-primary.yaml
        └── k8s-staging.yaml
```

Each YAML file contains a single entity in the standard Gantry format:

```yaml
kind: Service
apiVersion: gantry.io/v1
metadata:
  name: user-service
  title: User Service
  description: Manages user accounts and profiles
  owner: backend-services
  tags:
    - users
    - accounts
  labels:
    tier: backend
spec:
  type: backend
  lifecycle: production
  system: commerce
```

Server-managed fields (`createdAt`, `updatedAt`, `createdBy`) are stripped from the YAML output. The `default` namespace is omitted for cleaner files.

### API Endpoints

All endpoints require authentication. GitOps status/history/files and push/pull operations are `admin`-only.

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/plugins/gitops/status` | Current sync status (connected, last commit, errors) |
| `GET` | `/api/v1/plugins/gitops/history` | Recent sync operations (push/pull log) |
| `GET` | `/api/v1/plugins/gitops/files` | List all entity files tracked in the repo |
| `POST` | `/api/v1/plugins/gitops/sync` | Trigger a full sync (push all entities to Git) |
| `POST` | `/api/v1/plugins/gitops/pull` | Trigger a pull (fetch Git changes into Gantry) |

Sync and pull operations run asynchronously and return `202 Accepted` immediately. Check the status endpoint for progress.

**Status response:**

```json
{
  "connected": true,
  "repoUrl": "https://github.com/org/gantry-catalog.git",
  "branch": "main",
  "lastCommit": "a1b2c3d4e5f6",
  "lastCommitAt": "2026-03-12T10:30:00Z",
  "lastPushAt": "2026-03-12T10:30:00Z",
  "lastPullAt": "2026-03-12T10:25:00Z",
  "lastError": "",
  "pendingFiles": 0
}
```

**History entry:**

```json
{
  "id": "1741234567890",
  "timestamp": "2026-03-12T10:30:00Z",
  "direction": "push",
  "commit": "a1b2c3d4e5f6",
  "message": "gantry: update Service/user-service",
  "files": 1,
  "error": ""
}
```

For failed pulls, `error` contains a summary and the first few reconcile failures so the UI can show actionable details without flooding the history view.

## The Apply Command

`gantry apply` reads one or more YAML files and creates or updates entities in Gantry via the API. This is the simplest way to manage your catalog as code and works independently of the GitOps plugin.

```bash
gantry apply -f catalog.yaml
gantry apply -f teams.yaml -f services.yaml
gantry apply -f catalog/            # applies all YAML files in directory
```

**Behavior:**
- If the entity doesn't exist → **create**
- If the entity already exists → **update** (upsert by kind+name+namespace)
- Entities not mentioned in the file are **not deleted** (non-destructive by default)
- Supports multi-document YAML (separated by `---`)

### Catalog Repository Layout

A recommended structure for a `gantry apply`-based catalog repository:

```
catalog/
├── teams/
│   ├── platform-team.yaml
│   ├── backend-team.yaml
│   └── frontend-team.yaml
├── services/
│   ├── payment-api.yaml
│   ├── auth-service.yaml
│   └── notification-service.yaml
├── apis/
│   ├── payment-api-v2.yaml
│   └── auth-api-v1.yaml
├── infrastructure/
│   ├── postgres-payments.yaml
│   └── redis-sessions.yaml
├── environments/
│   ├── staging.yaml
│   └── production.yaml
└── actions/
    ├── deploy-service.yaml
    └── rollback-service.yaml
```

:::tip Apply order matters
Apply entities that are referenced by others first. A good order is: Teams → Environments → Infrastructure → Services → APIs → Documentation → Actions.
:::

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/catalog.yml
name: Sync Catalog

on:
  push:
    branches: [main]
    paths:
      - 'catalog/**'

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Gantry CLI
        run: |
          curl -sSL https://github.com/go2engle/gantry/releases/latest/download/install.sh | sh

      - name: Apply catalog
        env:
          GANTRY_SERVER: ${{ vars.GANTRY_SERVER }}
          GANTRY_TOKEN: ${{ secrets.GANTRY_TOKEN }}
        run: |
          for f in catalog/**/*.yaml; do
            gantry apply -f "$f"
          done
```

### GitLab CI

```yaml
# .gitlab-ci.yml
sync-catalog:
  stage: deploy
  only:
    - main
  script:
    - curl -sSL https://github.com/go2engle/gantry/releases/latest/download/install.sh | sh
    - |
      for f in catalog/**/*.yaml; do
        GANTRY_SERVER=$GANTRY_SERVER GANTRY_TOKEN=$GANTRY_TOKEN gantry apply -f "$f"
      done
```

## Entity Ownership via Git

Annotate entities with the Git file that owns them for traceability:

```yaml
metadata:
  name: payment-api
  annotations:
    git/file: catalog/services/payment-api.yaml
    git/repo: https://github.com/acme/platform-catalog
```

## Exporting the Current Catalog

Use `gantry export` to snapshot your current catalog as YAML — useful for initial migration or backup:

```bash
# Export everything
gantry export --format yaml > catalog-snapshot.yaml

# Export a specific namespace
gantry export --format yaml -n production > production-catalog.yaml

# Export as JSON
gantry export --format json > catalog.json
```

## Multi-Document YAML

A single YAML file can contain multiple entity documents separated by `---`:

```yaml
apiVersion: gantry.io/v1
kind: Team
metadata:
  name: platform-team
spec:
  members:
    - name: Alice Kim
      email: alice@example.com

---

apiVersion: gantry.io/v1
kind: Service
metadata:
  name: payment-api
  owner: platform-team
spec:
  type: backend
  lifecycle: production
```

## GitOps Plugin vs. `gantry apply`

| Feature | GitOps Plugin | `gantry apply` CLI |
|---|---|---|
| Direction | Bidirectional (push + pull) | One-way (files → Gantry) |
| Trigger | Automatic (on entity change) + manual | Manual (CLI or CI/CD) |
| Git integration | Built-in (clone, commit, push, pull) | External (you manage the Git workflow) |
| Conflict handling | Last-write-wins on pull | Upsert (create or update) |
| File layout | Auto-managed (`Kind/namespace/name.yaml`) | You define the layout |
| Best for | Live sync, GitOps-first workflows | CI/CD pipelines, initial seeding, scripting |

You can use both together. For example, use the GitOps plugin for live sync and `gantry apply` in CI for validation or bootstrapping a fresh instance.

## Best Practices

### Keep entity files focused

One entity per file, or logically grouped files (e.g., `team-and-services.yaml`). Avoid monolithic files with dozens of entities.

### Use owner references

Always set `metadata.owner` to a `Team` entity name. This enables the catalog to show team ownership clearly.

```yaml
metadata:
  owner: platform-team  # References kind: Team, name: platform-team
```

### Set `lifecycle` accurately

The `lifecycle` field on Service, API, and Infrastructure is the most important signal for catalog users:

- `experimental` — Not production-ready, may change
- `development` — In active development
- `production` — Live, stable, supported
- `deprecated` — Scheduled for removal

### Use tags consistently

Agree on a standard tag taxonomy in your org. Common patterns:
- By team: `payments`, `auth`, `platform`
- By criticality: `critical`, `pci-dss`, `soc2`
- By technology: `go`, `node`, `postgres`

### Pin API keys in CI

Don't use your admin password in CI/CD. Create a dedicated API key with `developer` role:

```bash
# In the UI: Settings → API Keys → Create New Key
# Or via API:
curl -X POST http://localhost:8080/api/v1/auth/apikeys \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "ci-catalog-sync", "role": "developer"}'
```

Store the key as a CI secret (`GANTRY_TOKEN`).
