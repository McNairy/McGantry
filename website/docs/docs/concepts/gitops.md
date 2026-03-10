---
sidebar_position: 4
title: GitOps
description: Manage your catalog as code using gantry apply.
---

# GitOps

Gantry is GitOps-native. Your entire service catalog can be defined as YAML files, checked into Git, and applied via `gantry apply`. This means your catalog is version-controlled, reviewable via pull requests, and reproducible.

## The Apply Command

`gantry apply` reads one or more YAML files and creates or updates entities in Gantry:

```bash
gantry apply -f catalog.yaml
gantry apply -f teams.yaml -f services.yaml
```

**Behavior:**
- If the entity doesn't exist → **create**
- If the entity already exists → **update** (upsert by kind+name+namespace)
- Entities not mentioned in the file are **not deleted** (non-destructive by default)

## Catalog Repository Layout

A recommended structure for a GitOps catalog repository:

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
