---
sidebar_position: 4
title: ArgoCD Plugin
description: Discover ArgoCD applications and trigger syncs from Gantry.
---

# ArgoCD Plugin

The ArgoCD plugin discovers ArgoCD Applications and represents them as `Service` entities in the catalog. It adds a live ArgoCD panel to entity detail pages and lets platform engineers trigger syncs and refreshes directly from Gantry.

## What it Does

- **Application discovery** — Syncs ArgoCD Applications as `Service` entities
- **Smart correlation** — Merges with Kubernetes-synced entities of the same name (no duplicates)
- **Live status panel** — Shows sync health, health status, and managed resources
- **Sync/refresh actions** — Trigger ArgoCD operations from the entity detail page

## Installation

```bash
curl -X POST http://localhost:8080/api/v1/plugins/argocd/install \
  -H "Authorization: Bearer <token>"
```

## Configuration

```bash
curl -X PUT http://localhost:8080/api/v1/plugins/argocd/config \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://argocd.your-org.com",
    "token": "your-argocd-api-token",
    "insecure": "false"
  }'
```

### Config Schema

| Field | Type | Description |
|---|---|---|
| `url` | string | ArgoCD server URL (e.g., `https://argocd.your-org.com`) |
| `token` | string | ArgoCD API token (see below for how to create) |
| `insecure` | `"true"` \| `"false"` | Skip TLS verification (not recommended for production) |

### Creating an ArgoCD API Token

```bash
# Create a read-only service account in ArgoCD
argocd account generate-token --account gantry-reader

# Or with the ArgoCD API:
curl -X POST https://argocd.your-org.com/api/v1/session \
  -d '{"username": "admin", "password": "..."}'
```

For sync/refresh operations, the token needs `applications, sync` permissions. A read-only token is sufficient for discovery only.

## Synced Entity Format

Each ArgoCD Application becomes a `Service` entity:

```yaml
apiVersion: gantry.io/v1
kind: Service
metadata:
  name: payment-api              # ArgoCD app name
  annotations:
    argocd.io/appName: payment-api
    argocd.io/namespace: argocd
    argocd.io/syncStatus: Synced
    argocd.io/healthStatus: Healthy
    argocd.io/repoURL: https://github.com/acme/payment-api
    argocd.io/destNamespace: payments
    argocd.io/destServer: https://kubernetes.default.svc
spec:
  lifecycle: production
  type: backend
  repoUrl: https://github.com/acme/payment-api   # from ArgoCD Git source
  deployedIn:
    - kind: Environment
      name: payments             # from destNamespace
```

## Smart Correlation with Kubernetes

If both the Kubernetes plugin and ArgoCD plugin are active, entities synced by both plugins are **merged** rather than duplicated:

- ArgoCD app name = Kubernetes deployment `app` label → same entity
- Annotations from both plugins are merged onto the entity
- `spec.deployedIn` (from ArgoCD `destNamespace`) aligns with the Environment entity synced by the k8s plugin
- `spec.repoUrl` (from ArgoCD Git source) enables the GitHub tab to activate automatically

This three-way correlation (k8s + ArgoCD + GitHub) means a single entity shows live Kubernetes pod data, ArgoCD sync status, and GitHub PR/commit info — all on the same page.

## Entity Panel (ArgoCD Tab)

The ArgoCD tab appears automatically on `Service` entities with `argocd.io/appName` annotation.

The panel shows:
- **Status card** — Sync status (Synced/OutOfSync), health status (Healthy/Degraded/Progressing), last sync time
- **Action buttons** — Sync, Hard Sync, Refresh
- **Managed resources table** — All k8s resources managed by the ArgoCD app (group, kind, name, namespace, status)

### Overview Sidebar Card

The entity Overview tab shows a compact ArgoCD card with sync and health status pulled from `argocd.io/*` annotations — visible at a glance without needing to open the ArgoCD tab.

## API Reference

### Get App Detail

```
GET /api/v1/plugins/argocd/apps/{appName}
```

Returns full ArgoCD application state including resources.

### Sync Application

```
POST /api/v1/plugins/argocd/apps/{appName}/sync
```

Requires `developer` role. Triggers a normal ArgoCD sync.

### Hard Sync

```
POST /api/v1/plugins/argocd/apps/{appName}/sync
```

With body `{"force": true}` — prunes resources and forces re-apply.

### Refresh

```
POST /api/v1/plugins/argocd/apps/{appName}/refresh
```

Requests ArgoCD to re-fetch the Git source without syncing.

### Get Apps for Entity

```
GET /api/v1/plugins/argocd/entity-apps?name=payment-api
```

Returns all ArgoCD apps matching an entity name.

## Linking Existing Entities to ArgoCD

Add annotations to manually-created entities to enable the ArgoCD tab:

```yaml
metadata:
  name: payment-api
  annotations:
    argocd.io/appName: payment-api-production    # ArgoCD application name
```

The plugin will fetch live status from ArgoCD using this app name.

## Sync Trigger

```bash
curl -X POST http://localhost:8080/api/v1/plugins/argocd/sync \
  -H "Authorization: Bearer <token>"
```

Re-discovers all ArgoCD applications and upserts them as entities.

## Troubleshooting

**ArgoCD tab not appearing:**
- Verify `argocd.io/appName` annotation is set on the entity
- Check that the plugin is enabled and configured correctly

**Sync button fails with 403:**
- The ArgoCD token needs `applications, sync` permission
- Check the Gantry user has `developer` role or higher

**Entity not created after sync:**
- Check server logs for ArgoCD connection errors
- Verify the ArgoCD URL is reachable from the Gantry server
- Test the token: `curl -H "Authorization: Bearer <token>" https://argocd.your-org.com/api/v1/applications`
