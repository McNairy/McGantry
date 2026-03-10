---
sidebar_position: 1
title: Plugin Overview
description: How Gantry's plugin system works.
---

# Plugin System

Plugins extend Gantry's capabilities by connecting external systems. They can sync entities from those systems into the catalog, add context panels to entity detail pages, and contribute new action types.

## Built-in Plugins

Gantry ships with a bundled registry of official plugins:

| Plugin | Category | Description |
|---|---|---|
| [kubernetes](./kubernetes) | integration | Sync namespaces, deployments, services. View pod logs. |
| [github](./github) | integration | Sync repos as Service entities. Live PR/commit data. OAuth SSO. |
| [argocd](./argocd) | integration | Discover ArgoCD apps. Trigger syncs. |
| pagerduty | integration | *(Planned)* On-call schedules, incident status |
| datadog | integration | *(Planned)* Service metrics, dashboards |
| grafana | integration | *(Planned)* Dashboard links and status |
| slack | integration | *(Planned)* Channel info, notifications |

## Installing a Plugin

### Via the UI

1. Go to **Plugins** in the sidebar (requires `developer` role or higher)
2. Browse the **Marketplace** tab
3. Click **Install** on the plugin you want
4. Configure it in the **Configure** modal
5. Click **Enable** to activate

### Via the API

```bash
# Install
curl -X POST http://localhost:8080/api/v1/plugins/kubernetes/install \
  -H "Authorization: Bearer <token>"

# Configure
curl -X PUT http://localhost:8080/api/v1/plugins/kubernetes/config \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"clusters": "[{\"name\":\"prod\",\"kubeconfig\":\"...\"}]"}'

# Enable
curl -X PUT http://localhost:8080/api/v1/plugins/kubernetes/enable \
  -H "Authorization: Bearer <token>" \
  -d '{"enabled": true}'

# Trigger sync
curl -X POST http://localhost:8080/api/v1/plugins/kubernetes/sync \
  -H "Authorization: Bearer <token>"
```

## Plugin Lifecycle

```
Install → Configure → Enable → [Sync] → Disable → Uninstall
```

- **Install** — Registers the plugin in the `plugins` DB table
- **Configure** — Sets plugin-specific config (credentials, endpoints, etc.). Sensitive values are encrypted with AES-256-GCM.
- **Enable** — Activates the plugin; triggers initial sync if applicable
- **Sync** — Manually triggers a re-sync of entities from the external system
- **Disable** — Stops the plugin from running but keeps config intact
- **Uninstall** — Removes the plugin and all its config

## Plugin Configuration Storage

Plugin configs are stored as JSON in the `plugins.config` column. Sensitive fields (API tokens, kubeconfigs) are encrypted with AES-256-GCM using the key from `GANTRY_ENCRYPTION_KEY` / `$GANTRY_DATA_DIR/encryption.key`.

## Entity Panels

Plugins can contribute panels that appear as tabs on entity detail pages:

- **Kubernetes plugin** → Adds a "Kubernetes" tab to Service entities with a `kubernetes.io/cluster` annotation
- **GitHub plugin** → Adds a "GitHub" tab to entities with `spec.repoUrl` containing `github.com`
- **ArgoCD plugin** → Adds an "ArgoCD" tab to Service entities with `argocd.io/appName` annotation

Panels appear automatically when the entity has the relevant annotations or spec fields — no manual configuration needed.

## Plugin Permissions

Managing plugins (install, configure, enable, sync) requires `developer` role or higher.

Viewing plugin data (entity panels, tabs) is available to all authenticated users.

## Syncing Entities

When a plugin syncs, it **upserts** entities into the catalog using `kind`, `name`, and `namespace` as the unique key. Plugins typically:

1. Fetch data from the external system
2. Transform it into Gantry entity format
3. Call the entity CRUD API to create/update
4. Log results including any errors to the server log

Entity updates from plugin sync are recorded in the audit log with `source: plugin`.

:::tip Sync Behavior
Plugins use upsert semantics — they will never delete manually-created entities. If a resource disappears from the external system, the corresponding entity remains in the catalog until manually removed.
:::
