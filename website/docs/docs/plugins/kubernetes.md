---
sidebar_position: 2
title: Kubernetes Plugin
description: Sync Kubernetes resources into the Gantry catalog.
---

# Kubernetes Plugin

The Kubernetes plugin syncs resources from one or more Kubernetes clusters into the Gantry catalog and adds live workload context to entity detail pages.

## What it Does

**Entity Sync:**
- Namespaces → `Environment` entities
- Deployments → `Service` entities
- Kubernetes Services → `Infrastructure` entities

**Live Context (entity panels):**
- Deployment status and replica counts
- Pod list with individual pod status
- Container log streaming
- Live health indicator

## Installation

```bash
# Via API
curl -X POST http://localhost:8080/api/v1/plugins/kubernetes/install \
  -H "Authorization: Bearer <token>"
```

Or via the UI: **Plugins → Marketplace → Kubernetes → Install**

## Configuration

The Kubernetes plugin supports **multiple clusters**. Configure each cluster as a JSON array:

### Via the UI

Go to **Plugins → Installed → Kubernetes → Configure**. Add clusters using the multi-cluster form.

### Via the API

```bash
curl -X PUT http://localhost:8080/api/v1/plugins/kubernetes/config \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "clusters": "[{\"name\":\"prod\",\"kubeconfig\":\"<base64-kubeconfig>\",\"namespaceFilter\":\"payments,platform\",\"labelSelector\":\"app.kubernetes.io/managed-by=helm\"}]"
  }'
```

### Config Schema

Each cluster object in the `clusters` array:

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Display name for this cluster |
| `kubeconfig` | string | Yes | Full kubeconfig YAML (will be encrypted at rest) |
| `namespaceFilter` | string | No | Comma-separated namespace names to sync (empty = all) |
| `labelSelector` | string | No | Kubernetes label selector to filter resources (e.g., `app.kubernetes.io/managed-by=helm`) |

## Enabling and Syncing

```bash
# Enable the plugin (triggers initial sync)
curl -X PUT http://localhost:8080/api/v1/plugins/kubernetes/enable \
  -H "Authorization: Bearer <token>" \
  -d '{"enabled": true}'

# Trigger manual sync
curl -X POST http://localhost:8080/api/v1/plugins/kubernetes/sync \
  -H "Authorization: Bearer <token>"
```

## Synced Entity Format

### Namespace → Environment

```yaml
apiVersion: gantry.io/v1
kind: Environment
metadata:
  name: payments             # namespace name
  annotations:
    kubernetes.io/cluster: prod
    kubernetes.io/namespace: payments
spec:
  type: production           # inferred from namespace name patterns
  cluster: prod
```

### Deployment → Service

```yaml
apiVersion: gantry.io/v1
kind: Service
metadata:
  name: payment-api          # deployment name
  annotations:
    kubernetes.io/cluster: prod
    kubernetes.io/namespace: payments
    kubernetes.io/deployment: payment-api
spec:
  type: backend
  lifecycle: production
  deployedIn:
    - kind: Environment
      name: payments
```

### Kubernetes Service → Infrastructure

```yaml
apiVersion: gantry.io/v1
kind: Infrastructure
metadata:
  name: payment-api-svc
  annotations:
    kubernetes.io/cluster: prod
    kubernetes.io/namespace: payments
spec:
  type: cache               # inferred from service type/port
```

## Entity Panel (Kubernetes Tab)

The Kubernetes tab appears automatically on `Service` entities that have either:
- `annotations.kubernetes.io/cluster` set (from sync), OR
- The entity `name` matches a deployment's `app` label selector

The panel shows:
- Deployment status (ready replicas, desired replicas)
- Pod list with status and restarts
- **Log streaming** — click any container to stream its logs in real-time

### Log Streaming API

```
GET /api/v1/plugins/kubernetes/pods/{namespace}/{pod}/containers/{container}/logs
```

Returns chunked HTTP response with log lines. The UI polls this endpoint and displays logs in a terminal-style view.

### Workload API

```
GET /api/v1/plugins/kubernetes/workload/{appName}?cluster=prod&namespace=payments
```

Returns deployment info and pod list for a given `app` label value.

## Linking Existing Entities to Kubernetes

If you have manually-created Service entities, link them to their Kubernetes workloads by adding annotations:

```yaml
metadata:
  name: payment-api
  annotations:
    kubernetes.io/cluster: prod        # Cluster name from plugin config
    kubernetes.io/namespace: payments  # Kubernetes namespace
    # The entity name must match the deployment's app label
```

## Multi-Cluster Sync

All configured clusters are synced. Entity names are prefixed with the cluster name when there are naming conflicts across clusters:

```
prod-payment-api        (from cluster "prod")
staging-payment-api     (from cluster "staging")
```

Sync errors per cluster are logged with the prefix `[kubernetes-sync]` and visible in the server log.

## Troubleshooting

**Entities not appearing after sync:**
- Check the server log for `[kubernetes-sync]` errors
- Verify the kubeconfig has permission to `list` and `get` namespaces, deployments, and services
- Check `namespaceFilter` isn't accidentally excluding the namespace

**"Kubernetes tab" not showing on entity:**
- Ensure the entity has `annotations.kubernetes.io/cluster` set
- Or ensure the entity `name` matches the deployment's `app` label value

**Log streaming not working:**
- The kubeconfig needs `pods/log` permission
- Check the container name is correct
