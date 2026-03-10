---
sidebar_position: 3
title: Actions
description: Self-service runbooks and workflows with schema-driven forms.
---

# Actions

Actions are self-service workflows that developers can trigger from the Gantry UI or CLI without opening a ticket. They use JSON Schema forms for input, support multiple execution backends, and track run history with full audit logging.

## What is an Action?

An Action is a Gantry entity of `kind: Action`. It defines:

- **What to run** — `spec.type` determines the executor (GitHub Actions, webhook, ArgoCD, etc.)
- **What inputs it needs** — `spec.inputs` generates a validated form in the UI
- **Who can run it** — `spec.permissions` controls access by role or team
- **How it's configured** — `spec.config` holds executor-specific settings

## Defining an Action

```yaml
apiVersion: gantry.io/v1
kind: Action
metadata:
  name: deploy-service
  title: Deploy Service
  description: Deploy a service to a target environment
  owner: platform-team
  tags:
    - deployment
spec:
  type: github-action
  category: Deployment

  inputs:
    - name: service
      title: Service Name
      type: string
      description: The service to deploy
      required: true
    - name: environment
      title: Target Environment
      type: select
      options: [staging, production]
      required: true
    - name: dry_run
      title: Dry Run
      type: boolean
      default: false
    - name: notes
      title: Release Notes
      type: textarea
      required: false

  config:
    repository: acme/deployments
    workflow: deploy.yml
    ref: main

  permissions:
    allowedRoles:
      - developer
      - platform-engineer
      - admin
    requireApproval: false
```

## Input Types

| Type | UI Element | Notes |
|---|---|---|
| `string` | Text input | Single-line text |
| `textarea` | Multi-line textarea | For long content, scripts, etc. |
| `number` | Number input | Integer or float |
| `boolean` | Checkbox | True/false toggle |
| `select` | Dropdown | Requires `options: [...]` array |

## Action Types

### `github-action`

Triggers a GitHub Actions workflow dispatch.

```yaml
spec:
  type: github-action
  config:
    repository: org/repo      # GitHub repository (owner/name)
    workflow: deploy.yml      # Workflow file name or ID
    ref: main                 # Branch, tag, or SHA to run on
```

Requires the GitHub plugin to be installed and configured with a Personal Access Token that has `actions:write` permission.

### `webhook`

Sends an HTTP POST request to a URL with the action inputs as JSON body.

```yaml
spec:
  type: webhook
  config:
    url: https://hooks.example.com/deploy
    method: POST              # POST (default)
    headers:
      Authorization: Bearer ${SECRET_TOKEN}
      Content-Type: application/json
```

### `argocd-sync`

Triggers an ArgoCD application sync.

```yaml
spec:
  type: argocd-sync
  config:
    appName: my-app           # ArgoCD application name
```

Requires the ArgoCD plugin to be installed and configured.

### `internal`

Runs built-in Gantry operations (e.g., entity management automation).

### `flux-update`

*(Planned)* Triggers a Flux reconciliation or image update.

## Permissions

```yaml
spec:
  permissions:
    allowedRoles:
      - developer             # Minimum role required
      - platform-engineer
      - admin
    allowedTeams:
      - platform-team         # Team entity names (any member can run)
    requireApproval: true     # Require approval before execution
    approvers:
      - alice                 # Username or team name
```

If `allowedRoles` is empty, any authenticated user can run the action. `requireApproval: true` (not yet implemented, planned) holds the run in `pending` state until approved.

## Running Actions

### From the UI

Navigate to **Actions** in the sidebar. Click on an action to see its form, fill in the inputs, and click **Execute**. The run appears in the Runs tab with real-time status.

### From the CLI

```bash
# Basic execution
gantry run deploy-service --input service=payment-api --input environment=staging

# Pass inputs as JSON
gantry run deploy-service --inputs '{"service": "payment-api", "environment": "staging"}'

# Wait for completion
gantry run deploy-service \
  --input service=payment-api \
  --input environment=staging \
  --wait \
  --timeout 120
```

### From the API

```bash
curl -X POST http://localhost:8080/api/v1/actions/deploy-service/execute \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"inputs": {"service": "payment-api", "environment": "staging"}}'
```

Response:

```json
{
  "id": "run-uuid",
  "actionName": "deploy-service",
  "status": "running",
  "inputs": {"service": "payment-api", "environment": "staging"},
  "triggeredBy": "alice",
  "startedAt": "2025-01-01T12:00:00Z"
}
```

## Run Status

| Status | Description |
|---|---|
| `pending` | Run created, waiting to execute |
| `running` | Currently executing |
| `success` | Completed successfully |
| `failed` | Execution failed |

## Action Runs

All runs are stored in the `action_runs` table and visible in the UI under **Actions → Runs**.

```bash
# List runs for a specific action
curl http://localhost:8080/api/v1/actions/deploy-service/runs \
  -H "Authorization: Bearer <token>"

# Get a specific run
curl http://localhost:8080/api/v1/actions/deploy-service/runs/run-uuid \
  -H "Authorization: Bearer <token>"

# List all runs across all actions
curl http://localhost:8080/api/v1/actions/runs \
  -H "Authorization: Bearer <token>"
```

## Audit Log

Every action execution is recorded in the audit log with:
- Who triggered the run
- What inputs were provided
- The IP address of the request
- Before/after state if the action modified entities
