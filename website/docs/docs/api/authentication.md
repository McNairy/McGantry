---
sidebar_position: 1
title: Authentication
description: JWT tokens, API keys, and role-based access control.
---

# Authentication

Gantry uses JWT Bearer tokens for user sessions and API keys for programmatic access. All API endpoints (except health checks, login, and GitHub OAuth config) require authentication.

## User Login

```
POST /api/v1/auth/login
```

```json
{
  "username": "admin",
  "password": "changeme"
}
```

Response:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid",
    "username": "admin",
    "displayName": "Administrator",
    "email": "",
    "role": "admin"
  }
}
```

JWT tokens expire after **24 hours**. Re-login to get a new token.

## Using Tokens

Include the token in all API requests:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

```bash
# Example
curl http://localhost:8080/api/v1/entities \
  -H "Authorization: Bearer <your-token>"
```

## API Keys

API keys are long-lived credentials for CI/CD and automation. They use the same `Authorization: Bearer` header but with a different format: `gantry_<64-hex-chars>`.

### Creating an API Key

**Via the UI:** Settings → API Keys → Create New Key

**Via the API:**

```bash
curl -X POST http://localhost:8080/api/v1/auth/apikeys \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ci-catalog-sync",
    "role": "developer"
  }'
```

Response:

```json
{
  "id": "uuid",
  "name": "ci-catalog-sync",
  "key": "gantry_a1b2c3d4e5f6...",
  "prefix": "gantry_a1b2c3",
  "role": "developer",
  "createdAt": "2025-01-01T00:00:00Z"
}
```

:::warning
The raw API key is only shown once at creation. Store it securely — Gantry only stores a SHA-256 hash.
:::

### Using API Keys

```bash
curl http://localhost:8080/api/v1/entities \
  -H "Authorization: Bearer gantry_yourkeyhere"
```

### Managing API Keys

```bash
# List your keys
curl http://localhost:8080/api/v1/auth/apikeys \
  -H "Authorization: Bearer <token>"

# Revoke a key
curl -X DELETE http://localhost:8080/api/v1/auth/apikeys/{id} \
  -H "Authorization: Bearer <token>"
```

## Roles and Permissions

Gantry has four roles in ascending order of privilege:

| Role | Level | Capabilities |
|---|---|---|
| `viewer` | 1 | Read entities, search, view schemas and audit log |
| `developer` | 2 | All viewer capabilities + create/update/delete entities, execute actions, manage plugins |
| `platform-engineer` | 3 | All developer capabilities + reserved for future use |
| `admin` | 4 | All capabilities + user management, dashboard configuration |

The default `admin` user has the `admin` role. Newly registered users get `viewer` by default.

### Role Enforcement

Role checks use a hierarchical model — any role with a **higher or equal level** passes:

```
endpoint requires developer (level 2)
  → viewer (1)  → DENIED
  → developer (2) → ALLOWED
  → platform-engineer (3) → ALLOWED
  → admin (4) → ALLOWED
```

## User Management

### Register a New User (Admin Only)

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "password": "secure-password",
    "displayName": "Alice Kim",
    "email": "alice@example.com",
    "role": "developer"
  }'
```

### List Users (Admin Only)

```bash
curl http://localhost:8080/api/v1/auth/users \
  -H "Authorization: Bearer <admin-token>"
```

### Update User Role (Admin Only)

```bash
curl -X PUT http://localhost:8080/api/v1/auth/users/{id} \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"role": "platform-engineer"}'
```

### Delete User (Admin Only)

```bash
curl -X DELETE http://localhost:8080/api/v1/auth/users/{id} \
  -H "Authorization: Bearer <admin-token>"
```

### Change Your Own Password

```bash
curl -X PUT http://localhost:8080/api/v1/auth/me/password \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "currentPassword": "old-password",
    "newPassword": "new-secure-password"
  }'
```

## Get Current User

```bash
curl http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer <token>"
```

Response:

```json
{
  "id": "uuid",
  "username": "alice",
  "displayName": "Alice Kim",
  "email": "alice@example.com",
  "role": "developer"
}
```

## Public Endpoints (No Auth Required)

These endpoints are accessible without authentication:

| Endpoint | Description |
|---|---|
| `GET /healthz` | Health check |
| `GET /readyz` | Readiness check |
| `GET /metrics` | Prometheus metrics |
| `POST /api/v1/auth/login` | User login |
| `GET /api/v1/auth/github/config` | GitHub SSO status |
| `GET /api/v1/auth/github` | GitHub OAuth redirect |
| `GET /api/v1/auth/github/callback` | GitHub OAuth callback |

## GitHub OAuth SSO

When the GitHub plugin is configured with `oauthEnabled: "true"`, users can sign in with GitHub. See [GitHub Plugin](../plugins/github#github-oauth-sso) for setup instructions.
