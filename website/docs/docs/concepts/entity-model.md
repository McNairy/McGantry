---
sidebar_position: 1
title: Entity Model
description: Understanding Gantry's core entity structure.
---

# Entity Model

Everything in Gantry is an **entity** — a typed, versioned YAML object. Entities are the building blocks of your service catalog. They describe services, APIs, teams, environments, and more.

## Structure

Every entity has the same top-level shape:

```yaml
apiVersion: gantry.io/v1      # Always "gantry.io/v1"
kind: Service                  # Entity type (see Kinds below)
metadata:
  name: payment-api            # Required. Unique per kind+namespace.
  namespace: default           # Optional. Defaults to "default".
  title: Payment API           # Optional. Human-friendly display name.
  description: "..."           # Optional. Short description.
  owner: platform-team         # Optional. Owning team or person.
  tags:
    - payments
    - critical
  labels:
    tier: backend
  annotations:
    github.com/project: payment-api
spec:                          # Kind-specific configuration
  type: backend
  lifecycle: production
  # ... more kind-specific fields
```

## Universal Metadata Fields

These fields are available on **every** entity kind:

| Field | Type | Description |
|---|---|---|
| `name` | string | **Required.** Unique identifier within kind+namespace. Use `kebab-case`. |
| `namespace` | string | Logical grouping (default: `default`). |
| `title` | string | Human-friendly display name. |
| `description` | string | Short description of the entity. |
| `owner` | string | Owning team name or person. Should reference a `Team` entity name. |
| `tags` | string[] | Free-form tags for filtering and grouping. |
| `labels` | object | Key-value pairs for structured metadata. |
| `annotations` | object | Plugin-specific and tool metadata. |

## Kinds

Gantry ships with 7 built-in entity kinds. All schemas use JSON Schema draft 2020-12 with `additionalProperties: false`.

### Service

A deployable software component (microservice, frontend app, library, worker).

```yaml
kind: Service
spec:
  type: backend | frontend | fullstack | library | worker
  lifecycle: experimental | development | production | deprecated
  system: my-system             # Parent system/domain
  repoUrl: https://github.com/...
  deployedIn:
    - kind: Environment
      name: production
  dependsOn:
    - kind: Service
      name: auth-service
  providesApis:
    - payment-api-v2
  consumesApis:
    - notification-api
  links:
    - title: Dashboard
      url: https://grafana.example.com/payment-api
      icon: dashboard           # dashboard|docs|runbook|github|slack|alert|monitor|ci|other
```

### API

An interface provided by a service — REST, gRPC, GraphQL, or event-based.

```yaml
kind: API
spec:
  type: rest | grpc | graphql | event
  lifecycle: experimental | development | production | deprecated
  definition: https://github.com/.../openapi.yaml  # Spec URL
  owner: platform-team
  system: payments
  repoUrl: https://github.com/...
```

### Team

An engineering team and its membership.

```yaml
kind: Team
spec:
  members:
    - name: Alice Kim
      email: alice@example.com
      role: lead
  slackChannel: "#platform-eng"
  email: platform@example.com
  oncallSchedule: https://pagerduty.example.com/schedule
  manager: Bob Chen
```

### Environment

A deployment target — staging, production, development.

```yaml
kind: Environment
spec:
  type: staging | production | development
  provider: AWS            # or GCP, Azure, on-prem, etc.
  region: us-east-1
  cluster: prod-k8s        # Kubernetes cluster name (for k8s plugin)
  accountId: "123456789"   # Cloud account/project ID
```

### Infrastructure

A backing service — database, queue, cache, storage.

```yaml
kind: Infrastructure
spec:
  type: database | queue | cache | storage | bucket
  provider: AWS RDS
  lifecycle: production
  connectionString: postgres://...  # Sensitive — consider using annotations
  system: payments
  repoUrl: https://github.com/acme/infra  # IaC repository
```

### Action

A self-service workflow or runbook. See [Actions](./actions) for full detail.

```yaml
kind: Action
spec:
  type: github-action | argocd-sync | flux-update | webhook | internal
  category: Deployment
  inputs:
    - name: environment
      title: Target Environment
      type: select
      options: [staging, production]
      required: true
  config:
    # Type-dependent configuration
```

### Documentation

A link to external documentation for an entity.

```yaml
kind: Documentation
spec:
  type: runbook | wiki | api-docs | architecture
  url: https://wiki.example.com/payment-api
  associatedEntity: payment-api
  format: markdown
```

## Relationships

Entities reference each other using `{kind, name}` pairs in `spec` fields:

```yaml
# In a Service spec:
deployedIn:
  - kind: Environment
    name: production
dependsOn:
  - kind: Service
    name: auth-service
  - kind: Infrastructure
    name: postgres-payments
providesApis:
  - payment-api-v2          # API entity name
consumesApis:
  - notification-api        # API entity name
```

The **Relationship Graph** in the UI visualizes these connections. You can also query them via `GET /api/v1/graph/{kind}/{name}`.

## Namespaces

Namespaces let you logically separate entities — e.g., by team, environment type, or business unit. Most installations use the default namespace.

```yaml
metadata:
  name: payment-api
  namespace: payments-team   # Custom namespace
```

:::tip
Cross-namespace references are not yet supported. All entities in a `dependsOn` reference must be in the same namespace.
:::

## Links

Any entity can define `spec.links` — an array of clickable URLs shown in the entity's sidebar:

```yaml
spec:
  links:
    - title: Runbook
      url: https://wiki.example.com/runbook
      icon: runbook
    - title: Grafana Dashboard
      url: https://grafana.example.com/d/abc
      icon: dashboard
    - title: Slack Channel
      url: https://slack.com/app_redirect?channel=platform-eng
      icon: slack
```

Available icons: `dashboard`, `docs`, `runbook`, `github`, `slack`, `alert`, `monitor`, `ci`, `other`

## Validation

Every entity is validated against its kind's JSON Schema before being stored. Invalid entities are rejected with a descriptive error. Schemas are stored in `internal/entity/schemas/` and are embedded in the binary.

To view the schema for a kind:

```bash
curl http://localhost:8080/api/v1/schemas/service
```
