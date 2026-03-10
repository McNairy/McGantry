---
sidebar_position: 2
title: Quick Start
description: Create your first Gantry catalog in under 5 minutes.
---

# Quick Start

This guide walks you through creating your first service catalog from scratch. You'll define entities in YAML, apply them to Gantry, and explore the result in the UI.

## 1. Start Gantry

```bash
gantry serve --dev
```

Gantry starts on port 8080. The `--dev` flag enables verbose logging and relaxed CORS for local development.

## 2. Create a Catalog File

Create a file called `catalog.yaml`. You can define multiple entities in a single file using YAML document separators (`---`):

```yaml
# catalog.yaml

apiVersion: gantry.io/v1
kind: Team
metadata:
  name: platform-team
  description: Platform engineering team
spec:
  members:
    - name: Alice Kim
      email: alice@example.com
      role: lead
    - name: Bob Chen
      email: bob@example.com
  slackChannel: "#platform-eng"

---

apiVersion: gantry.io/v1
kind: Environment
metadata:
  name: production
  description: Production environment (AWS us-east-1)
spec:
  type: production
  provider: AWS
  region: us-east-1
  cluster: prod-k8s

---

apiVersion: gantry.io/v1
kind: Service
metadata:
  name: payment-api
  description: Handles all payment processing
  owner: platform-team
  tags:
    - payments
    - critical
  labels:
    tier: "backend"
spec:
  type: backend
  lifecycle: production
  repoUrl: https://github.com/acme/payment-api
  deployedIn:
    - kind: Environment
      name: production
  links:
    - title: Runbook
      url: https://wiki.example.com/payment-api/runbook
      icon: runbook
    - title: Dashboard
      url: https://grafana.example.com/payment-api
      icon: dashboard

---

apiVersion: gantry.io/v1
kind: API
metadata:
  name: payment-api-v2
  description: REST API for payment processing (v2)
  owner: platform-team
  tags:
    - payments
spec:
  type: rest
  lifecycle: production
  definition: https://github.com/acme/payment-api/blob/main/openapi.yaml
```

## 3. Apply the Catalog

```bash
gantry apply -f catalog.yaml
```

Expected output:

```
✓ Created  Team/platform-team
✓ Created  Environment/production
✓ Created  Service/payment-api
✓ Created  API/payment-api-v2
```

If you run `apply` again, Gantry will update (upsert) existing entities. This makes it safe to run in CI/CD pipelines.

## 4. Explore in the UI

Open [http://localhost:8080](http://localhost:8080) and log in with `admin` / `changeme`.

You'll see your entities in the **Catalog** view. Click on `payment-api` to see:
- Entity details and spec fields
- Links sidebar (Runbook, Dashboard)
- Relationships to the Team and Environment
- Activity tab (audit log for this entity)

## 5. Use the CLI

```bash
# List all entities
gantry get

# List services
gantry get service

# Get a specific entity
gantry get service payment-api

# Export everything
gantry export --format yaml
```

## 6. Search

In the UI, press `/` to open the command palette and search for any entity by name, description, or tag. Or use the API:

```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/search?q=payment
```

## 7. Next Steps

- **Add more entities** — Infrastructure, Documentation, Actions
- **Install a plugin** — Connect [Kubernetes](../plugins/kubernetes), [GitHub](../plugins/github), or [ArgoCD](../plugins/argocd)
- **Create an Action** — Define a self-service runbook
- **Read about [Core Concepts](../concepts/entity-model)** to understand the entity model

## Common Patterns

### Apply multiple files

```bash
gantry apply -f teams.yaml -f services.yaml -f apis.yaml
```

### Apply a whole directory

```bash
# Apply all YAML files in the catalog/ directory
for f in catalog/*.yaml; do gantry apply -f "$f"; done
```

### Use from CI/CD

```bash
# Set credentials via environment
export GANTRY_SERVER=https://gantry.your-org.com
export GANTRY_TOKEN=gantry_yourapikeyhere

# Apply catalog on every push to main
gantry apply -f catalog.yaml
```
