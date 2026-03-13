---
sidebar_position: 3
title: GitHub Plugin
description: Sync GitHub repositories and enable OAuth SSO.
---

# GitHub Plugin

The GitHub plugin syncs repositories from a GitHub organization as `Service` entities, adds live repository context (commits, PRs, topics) to entity detail pages, and optionally enables GitHub OAuth for single sign-on.

## What it Does

- **Repo sync** — Syncs all repos in a GitHub org as `Service` entities
- **Live repo info** — Shows recent commits, open PRs, stars, forks, topics on entity detail pages
- **OAuth SSO** — "Sign in with GitHub" on the Gantry login page (optional)

## Installation

The GitHub plugin is bundled with Gantry. Open **Plugins**, select **GitHub**, configure it, then enable it.

## Authentication Options

The plugin supports two authentication methods:

### Option A: Personal Access Token (PAT)

Simplest approach for most teams.

Required scopes:
- `repo` — Read repository info and sync
- `workflow` — Trigger GitHub Actions (if using `github-action` action type)

### Option B: GitHub App

For production environments — GitHub Apps have higher rate limits and more granular permissions.

Required permissions:
- `Contents: Read` — Read repo metadata
- `Metadata: Read` — Required by all GitHub Apps
- `Actions: Write` — (optional) trigger workflow dispatches

## Configuration

```bash
curl -X PUT http://localhost:8080/api/v1/plugins/github/config \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "authMode": "pat",
    "personalAccessToken": "ghp_yourpersonalaccesstoken",
    "orgName": "acme-corp",
    "ssoEnabled": false
  }'
```

### Config Schema

| Field | Type | Description |
|---|---|---|
| `authMode` | `pat` \| `app` | Authentication method |
| `personalAccessToken` | string | PAT token (if `authMode=pat`) |
| `appId` | string | GitHub App ID (if `authMode=app`) |
| `installationId` | string | GitHub App installation ID (if `authMode=app`) |
| `privateKey` | string | GitHub App RSA private key PEM (if `authMode=app`) |
| `ssoEnabled` | boolean | Enable GitHub OAuth SSO |
| `oauthClientId` | string | GitHub OAuth App client ID (required if `ssoEnabled=true`) |
| `oauthClientSecret` | string | GitHub OAuth App client secret (required if `ssoEnabled=true`) |
| `autoProvision` | boolean | Automatically create Gantry users for first-time SSO logins |
| `defaultRole` | string | Role assigned to auto-provisioned users and unmatched synced teams |
| `syncTeams` | boolean | Sync GitHub org teams to Gantry groups during SSO login |
| `orgName` | string | GitHub organization name for repo sync and optional team sync |
| `teamRoleMappings` | array | Optional team slug to Gantry role mappings |

## Synced Entity Format

Each synced repository becomes a `Service` entity:

```yaml
apiVersion: gantry.io/v1
kind: Service
metadata:
  name: payment-api              # repo name
  description: "Handles payment processing"  # repo description
  annotations:
    github.com/repo: acme-corp/payment-api
    github.com/url: https://github.com/acme-corp/payment-api
    github.com/defaultBranch: main
    github.com/language: Go
    github.com/stars: "42"
    github.com/topics: payments,api,go
spec:
  type: backend
  lifecycle: production
  repoUrl: https://github.com/acme-corp/payment-api
```

## Entity Panel (GitHub Tab)

The GitHub tab appears automatically on any entity that has:
- `spec.repoUrl` containing `github.com`, OR
- `annotations.github.com/repo` set

The panel shows:
- Repository metadata card (stars, forks, open issues, primary language, topics)
- Recent commits (up to 10 most recent)
- Open pull requests

### Live Repo Info API

```
GET /api/v1/plugins/github/repo?url=https://github.com/acme/payment-api
```

Returns:

```json
{
  "repo": {
    "name": "payment-api",
    "description": "...",
    "stars": 42,
    "forks": 7,
    "openIssues": 3,
    "defaultBranch": "main",
    "language": "Go",
    "topics": ["payments", "api"]
  },
  "commits": [
    {
      "sha": "abc1234",
      "message": "fix: handle nil pointer in payment processor",
      "author": "alice",
      "date": "2025-01-01T12:00:00Z",
      "url": "https://github.com/acme/payment-api/commit/abc1234"
    }
  ],
  "pullRequests": [
    {
      "number": 42,
      "title": "feat: add refund support",
      "state": "open",
      "author": "bob",
      "url": "https://github.com/acme/payment-api/pull/42"
    }
  ]
}
```

## GitHub OAuth SSO

When configured, Gantry shows a **"Sign in with GitHub"** button on the login page. Users authenticate with GitHub and receive a Gantry JWT token.

### Setup

1. Create a **GitHub OAuth App** at `github.com → Settings → Developer settings → OAuth Apps`
   - Homepage URL: `https://gantry.your-org.com`
   - Callback URL: `https://gantry.your-org.com/api/v1/auth/github/callback`

2. Configure the plugin with `ssoEnabled: true`, `oauthClientId`, and `oauthClientSecret`

3. Check that SSO is enabled:
   ```bash
   curl http://localhost:8080/api/v1/auth/github/config
   # {"ssoEnabled": true}
   ```

### OAuth Flow

```
User clicks "Sign in with GitHub"
    → GET /api/v1/auth/github
    → Redirect to github.com/login/oauth/authorize
    → User approves
    → GET /api/v1/auth/github/callback?code=...
    → Gantry exchanges code for GitHub user info
    → Creates/updates Gantry user
    → Sets gantry_session HttpOnly cookie
    → Redirects back to the SPA
```

### User Provisioning

On first OAuth login, Gantry can create a user with:
- Username: `github:<login>`
- Display name: GitHub display name
- Role: `viewer` by default, or the configured `defaultRole`

If `autoProvision` is disabled, users must already exist in Gantry before SSO login. Existing users are matched first by `github:<login>`, then by email address if GitHub returns one.

## Triggering GitHub Actions

When the GitHub plugin is configured, you can trigger workflow dispatches via Gantry Actions:

```yaml
apiVersion: gantry.io/v1
kind: Action
metadata:
  name: deploy-service
spec:
  type: github-action
  config:
    repository: acme/deployments
    workflow: deploy.yml
    ref: main
  inputs:
    - name: environment
      title: Environment
      type: select
      options: [staging, production]
```

```bash
gantry run deploy-service --input environment=staging
```

## Sync Trigger

```bash
curl -X POST http://localhost:8080/api/v1/plugins/github/sync \
  -H "Authorization: Bearer <token>"
```

Syncs all repositories from the configured GitHub org. Existing entities are updated; new repos create new entities. Archived and forked repositories are excluded by default. This endpoint requires `platform-engineer` role or higher.
