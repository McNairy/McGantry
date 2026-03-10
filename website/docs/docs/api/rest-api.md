---
sidebar_position: 2
title: REST API
description: Complete Gantry REST API reference.
---

# REST API Reference

All API endpoints are prefixed with `/api/v1`. Requests must include `Authorization: Bearer <token>` unless noted as public.

## Entities

### List All Entities

```
GET /api/v1/entities
```

Query parameters:

| Param | Description |
|---|---|
| `namespace` | Filter by namespace (default: all) |
| `owner` | Filter by owner name |
| `tag` | Filter by tag |

```bash
curl "http://localhost:8080/api/v1/entities?tag=critical" \
  -H "Authorization: Bearer <token>"
```

### List Entities by Kind

```
GET /api/v1/entities/{kind}
```

`{kind}` is case-insensitive: `service`, `Service`, `SERVICE` all work.

```bash
curl http://localhost:8080/api/v1/entities/service \
  -H "Authorization: Bearer <token>"
```

### Get Entity

```
GET /api/v1/entities/{kind}/{name}?namespace=default
```

```bash
curl "http://localhost:8080/api/v1/entities/service/payment-api?namespace=default" \
  -H "Authorization: Bearer <token>"
```

Response:

```json
{
  "kind": "Service",
  "apiVersion": "gantry.io/v1",
  "metadata": {
    "name": "payment-api",
    "namespace": "default",
    "title": "Payment API",
    "description": "Handles all payment processing",
    "owner": "platform-team",
    "tags": ["payments", "critical"],
    "labels": {"tier": "backend"},
    "annotations": {},
    "createdAt": "2025-01-01T00:00:00Z",
    "updatedAt": "2025-01-01T00:00:00Z",
    "createdBy": "alice"
  },
  "spec": {
    "type": "backend",
    "lifecycle": "production",
    "repoUrl": "https://github.com/acme/payment-api"
  }
}
```

### Create Entity

```
POST /api/v1/entities
```

Requires `developer` role. Body is the full entity JSON:

```bash
curl -X POST http://localhost:8080/api/v1/entities \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "kind": "Service",
    "apiVersion": "gantry.io/v1",
    "metadata": {
      "name": "my-new-service",
      "description": "A new service"
    },
    "spec": {
      "type": "backend",
      "lifecycle": "development"
    }
  }'
```

Returns `201 Created` with the created entity.

### Update Entity

```
PUT /api/v1/entities/{kind}/{name}
```

Requires `developer` role. Replaces the full entity spec.

```bash
curl -X PUT http://localhost:8080/api/v1/entities/service/payment-api \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{...full entity JSON...}'
```

### Delete Entity

```
DELETE /api/v1/entities/{kind}/{name}?namespace=default
```

Requires `developer` role.

```bash
curl -X DELETE "http://localhost:8080/api/v1/entities/service/payment-api" \
  -H "Authorization: Bearer <token>"
```

## Search

```
GET /api/v1/search?q={query}
```

Full-text search across all entities. Searches name, title, description, tags, and owner fields.

```bash
curl "http://localhost:8080/api/v1/search?q=payment" \
  -H "Authorization: Bearer <token>"
```

Response:

```json
[
  {
    "kind": "Service",
    "name": "payment-api",
    "title": "Payment API",
    "description": "Handles all payment processing",
    "owner": "platform-team",
    "tags": ["payments"]
  }
]
```

## Relationship Graph

```
GET /api/v1/graph/{kind}/{name}?namespace=default
```

Returns a graph of entities related to the given entity (dependencies, APIs, environments).

```bash
curl "http://localhost:8080/api/v1/graph/service/payment-api" \
  -H "Authorization: Bearer <token>"
```

Response:

```json
{
  "nodes": [
    {"id": "service/payment-api", "kind": "Service", "name": "payment-api"},
    {"id": "infrastructure/postgres-payments", "kind": "Infrastructure", "name": "postgres-payments"}
  ],
  "edges": [
    {"from": "service/payment-api", "to": "infrastructure/postgres-payments", "type": "dependsOn"}
  ]
}
```

## Schemas

### List All Schemas

```
GET /api/v1/schemas
```

Returns JSON Schema for all entity kinds.

### Get Schema for Kind

```
GET /api/v1/schemas/{kind}
```

```bash
curl http://localhost:8080/api/v1/schemas/service
```

## Actions

### List Actions

```
GET /api/v1/actions
```

Returns all `Action` entities.

### Execute Action

```
POST /api/v1/actions/{name}/execute
```

Requires `developer` role.

```bash
curl -X POST http://localhost:8080/api/v1/actions/deploy-service/execute \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"inputs": {"environment": "staging", "service": "payment-api"}}'
```

Response:

```json
{
  "id": "run-uuid",
  "actionName": "deploy-service",
  "status": "running",
  "inputs": {"environment": "staging"},
  "triggeredBy": "alice",
  "startedAt": "2025-01-01T12:00:00Z"
}
```

### List Action Runs

```
GET /api/v1/actions/{name}/runs
GET /api/v1/actions/runs           (all actions)
```

### Get Action Run

```
GET /api/v1/actions/{name}/runs/{id}
```

## Audit Log

```
GET /api/v1/audit?limit=50&offset=0
```

Returns audit log entries in reverse chronological order.

```bash
curl "http://localhost:8080/api/v1/audit?limit=20" \
  -H "Authorization: Bearer <token>"
```

Response entries:

```json
{
  "entries": [
    {
      "id": "uuid",
      "timestamp": "2025-01-01T12:00:00Z",
      "userName": "alice",
      "action": "create",
      "resourceType": "Service",
      "resourceName": "payment-api",
      "source": "api",
      "ipAddress": "10.0.0.1"
    }
  ],
  "total": 142
}
```

## User History

### Record Entity View

```
POST /api/v1/history
```

```json
{
  "kind": "Service",
  "name": "payment-api",
  "namespace": "default"
}
```

The UI calls this automatically when you navigate to an entity detail page.

### Get Browsing History

```
GET /api/v1/history?limit=10
```

Returns the current user's recently viewed entities (max 20 per user).

## Plugins

### List Plugins

```
GET /api/v1/plugins
```

Returns all plugins from the registry with their installation status.

### Install Plugin

```
POST /api/v1/plugins/{name}/install
```

Requires `developer` role.

### Get Plugin Config

```
GET /api/v1/plugins/{name}/config
```

### Update Plugin Config

```
PUT /api/v1/plugins/{name}/config
```

Requires `developer` role. Body is a flat JSON object of string values.

### Enable/Disable Plugin

```
PUT /api/v1/plugins/{name}/enable
```

Body: `{"enabled": true}` or `{"enabled": false}`

### Trigger Sync

```
POST /api/v1/plugins/{name}/sync
```

Requires `developer` role.

### Uninstall Plugin

```
DELETE /api/v1/plugins/{name}
```

Requires `developer` role.

## Dashboard

### Get Dashboard Config

```
GET /api/v1/dashboard/config
```

### Update Dashboard Config (Admin Only)

```
PUT /api/v1/dashboard/config
```

```json
{
  "widgets": [
    {"id": "entity_stats", "enabled": true},
    {"id": "recent_activity", "enabled": true}
  ],
  "announcements": [
    {
      "title": "Scheduled Maintenance",
      "message": "Downtime on Saturday 2am-4am UTC",
      "severity": "warning"
    }
  ]
}
```

## Health Checks

```
GET /healthz    → {"status": "ok"}
GET /readyz     → {"status": "ok"} or 503 if DB unreachable
GET /metrics    → Prometheus text format metrics
```

## WebSocket

```
GET /api/v1/ws
```

Real-time entity change notifications. Connect with a WebSocket client:

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws');

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  // msg.type: "entity.created" | "entity.updated" | "entity.deleted"
  // msg.payload: entity object
};
```

Subscribe to a specific channel:

```javascript
ws.send(JSON.stringify({
  type: "subscribe",
  channel: "entities"
}));
```

## Error Responses

All errors return JSON:

```json
{
  "error": "entity not found",
  "code": 404
}
```

Common status codes:

| Code | Meaning |
|---|---|
| `400` | Bad request — invalid JSON or failed schema validation |
| `401` | Unauthorized — missing or invalid token |
| `403` | Forbidden — insufficient role |
| `404` | Entity not found |
| `409` | Conflict — entity already exists (use PUT to update) |
| `422` | Unprocessable — entity failed JSON Schema validation |
| `500` | Internal server error |
