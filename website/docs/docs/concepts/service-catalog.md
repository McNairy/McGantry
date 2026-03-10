---
sidebar_position: 2
title: Service Catalog
description: How the Gantry service catalog works.
---

# Service Catalog

The service catalog is the core of Gantry — a searchable, filterable, and navigable registry of every software component, team, and infrastructure piece in your organization.

## What Goes in the Catalog?

| Entity Kind | Examples |
|---|---|
| Service | Payment API, Auth service, Frontend app, Background worker |
| API | REST API, gRPC interface, GraphQL schema, Event topic |
| Team | Platform team, Backend team, Frontend team |
| Environment | Production, Staging, Development |
| Infrastructure | PostgreSQL database, Redis cache, S3 bucket, SQS queue |
| Action | Deploy workflow, Database migration, Rollback script |
| Documentation | Runbook, Architecture diagram, API docs |

## Browsing the Catalog

The **Catalog** page in the UI shows all entities in a filterable table. You can filter by:
- **Kind** — Services, APIs, Teams, etc.
- **Owner** — Filter by owning team
- **Tag** — Filter by any tag
- **Lifecycle** — Filter by development stage
- **Namespace** — Filter by namespace (if using multiple)

## Entity Detail View

Click any entity to see its detail page with tabs:

- **Overview** — Spec fields, links sidebar, metadata
- **Relations** — Dependency graph showing connected entities
- **Activity** — Audit log filtered to this entity
- **Plugin tabs** — Kubernetes, GitHub, ArgoCD tabs appear automatically when the entity has relevant annotations or spec fields

## Relationships

The catalog tracks relationships between entities. These are defined in entity specs:

```
payment-api (Service)
    ├── deployedIn → production (Environment)
    ├── deployedIn → staging (Environment)
    ├── dependsOn → postgres-payments (Infrastructure)
    ├── dependsOn → auth-service (Service)
    ├── providesApis → payment-api-v2 (API)
    └── owned by → platform-team (Team)
```

The **relationship graph** (`/api/v1/graph/{kind}/{name}`) traverses these connections and returns a graph suitable for visualization.

## Search

Press `/` in the UI to open the command palette. Search works across:
- Entity names
- Titles and descriptions
- Tags
- Owner names

Search is powered by SQLite FTS5 full-text search — results appear in milliseconds without any external search infrastructure.

```bash
# API search
curl "http://localhost:8080/api/v1/search?q=payment" \
  -H "Authorization: Bearer <token>"
```

## Ownership

Every entity should have an `owner` field pointing to a `Team` entity name. This enables:
- Filtering the catalog by team
- Understanding who to contact for a service
- Tracking which teams own which infrastructure

```yaml
metadata:
  name: payment-api
  owner: platform-team    # References Team/platform-team
```

## Real-Time Updates

When any entity is created, updated, or deleted, Gantry broadcasts the change over WebSocket to all connected clients. The UI updates in real-time without requiring a page refresh.

## Annotations

Annotations are arbitrary key-value metadata. They're used by plugins to store cross-system identifiers:

```yaml
metadata:
  annotations:
    github.com/repo: acme/payment-api
    argocd.io/appName: payment-api-production
    kubernetes.io/cluster: prod-k8s
    kubernetes.io/namespace: payments
```

:::note Plugin Annotations
Plugin annotations are filtered from the generic "Annotations" card in the UI. Each plugin renders its own card with the relevant annotations displayed in a human-readable format.
:::
