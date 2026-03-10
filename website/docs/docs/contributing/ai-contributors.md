---
sidebar_position: 5
title: AI Contributors Guide
description: A complete reference for AI agents (Claude, GPT, etc.) working on this codebase.
---

# AI Contributors Guide

This guide is specifically written for AI coding agents (Claude Code, GitHub Copilot, etc.) working on the Gantry codebase. It distills the most important knowledge needed to make correct, safe changes without breaking existing behavior.

:::tip For Human Reviewers
If you're a human reviewing AI-generated contributions, this document describes the standards the AI was following. Deviations from these standards are worth flagging in code review.
:::

## Start Here: CLAUDE.md

The repository root contains [`CLAUDE.md`](https://github.com/go2engle/gantry/blob/main/CLAUDE.md) — authoritative instructions that **override** general defaults. Always read it at the start of any session.

## Codebase Mental Model

```
Everything is an entity. Entities have kind + metadata + spec.
Spec is validated against a JSON Schema per kind.
Handlers write to DB + audit log + event bus.
Plugins sync external systems into the entity catalog.
Single binary: CGO_ENABLED=0 required always.
```

## Top Pitfalls (Read Before Making Any Change)

### 1. CSS Variables — Only Use Defined Ones

Only these CSS variables exist in Gantry's theme system:

```
--gantry-bg-primary      ✓
--gantry-bg-secondary    ✓
--gantry-bg-tertiary     ✓
--gantry-text-primary    ✓
--gantry-text-secondary  ✓
--gantry-border          ✓
--gantry-accent          ✓
--gantry-accent-hover    ✓
--gantry-danger          ✓
```

**These do NOT exist** — using them renders as transparent/invisible:
- ~~`--gantry-surface`~~
- ~~`--gantry-bg`~~
- ~~`--gantry-text`~~
- ~~`--gantry-hover`~~
- ~~`--gantry-text-muted`~~

### 2. Transparent Accent Backgrounds in Tailwind

```tsx
// Correct
className="bg-[var(--gantry-accent)]/10"

// Wrong — bg-opacity does not work with CSS variables
className="bg-[var(--gantry-accent)] bg-opacity-10"
```

### 3. Entity Schemas Have `additionalProperties: false`

All built-in schemas reject unknown spec fields. If a plugin sync sets an unknown field, it will fail schema validation. Always check `internal/entity/schemas/{kind}.json` before adding new fields in sync code.

### 4. DB Helper Receivers Are `d *DB`, Not `db.sql.*`

```go
// Correct
d.exec(query, args...)
d.queryRow(dest, query, args...)
d.queryRows(query, args...)

// Wrong — don't call sql methods directly
d.sql.ExecContext(ctx, query, args...)
```

### 5. SQLite Booleans Use `boolToInt()`

SQLite has no native boolean. The plugin `enabled` field is stored as INTEGER:

```go
// Write
boolToInt(plugin.Enabled)    // returns 0 or 1

// Read
plugin.Enabled = enabledInt != 0
```

### 6. The `cmd/gantry/` Package Was in .gitignore

The `.gitignore` previously had a bare `gantry` entry that matched the `cmd/gantry/` directory. This was fixed to `/gantry` (root-anchored). If CLI files disappear from git, check `.gitignore`.

### 7. Plugin RegistryEntry Must Include All Fields

When adding entries to `internal/plugins/bundled/registry.json`, include `ConfigSchema`, `EntityPanels`, and `ActionTypes` or the frontend marketplace won't render them correctly.

## Required Reading Before Each Task Type

### Modifying Entity Schemas

Read:
- `internal/entity/schemas/` — existing schemas for conventions
- `internal/entity/validator.go` — how schemas are loaded and validated
- The [Entity Model](../concepts/entity-model) doc

Rules:
- `additionalProperties: false` on the `spec` object — always
- New fields must be **optional** (never add required fields to existing kinds)
- Use `description` on every field (used in auto-generated UI forms)
- Test with: `go test ./internal/entity/...`

### Adding a New API Endpoint

Read:
- `internal/api/server.go` — route registration and middleware groups
- `internal/api/handlers/entities.go` — canonical handler pattern
- `internal/api/middleware/auth.go` — how auth context works

Checklist:
- [ ] Handler is a method on `handlers.Handlers`
- [ ] Route registered in `server.go` under correct auth group
- [ ] Auth check: `requireAuth` or `requireRole(role)` middleware
- [ ] Audit entry: `h.db.InsertAuditEntry()` for mutations
- [ ] Client IP: `clientIP(r)` in audit entry
- [ ] Event published: `h.events.Publish()` for entity changes
- [ ] JSON response with correct status code

### Adding a Frontend Feature

Read:
- `web/src/lib/api.ts` — add new API calls here first
- `web/src/lib/types.ts` — add TypeScript types
- `web/src/pages/` or `web/src/components/` — where to put new UI

Checklist:
- [ ] New API calls go through `api.ts`, never raw `fetch()`
- [ ] Types added to `types.ts`
- [ ] Only defined CSS variables used
- [ ] `npx tsc --noEmit` passes with zero errors

### Adding a Plugin

Read:
- `internal/plugins/manifest.go` — Manifest and RegistryEntry structs
- `internal/plugins/kubernetes/` — canonical plugin implementation
- `internal/plugins/bundled/registry.json` — bundled registry format
- `internal/api/server.go` — where plugin routes are registered

Steps:
1. Create `internal/plugins/{name}/plugin.go` with sync and handler logic
2. Add to `bundled/registry.json` with all required fields
3. Register plugin routes in `internal/api/server.go`
4. Add entity panel logic if the plugin contributes UI tabs
5. Test with manual install → configure → enable → sync flow

### Database Schema Changes

Read:
- `internal/db/migrations.go` — existing migration pattern
- `internal/db/db.go` — helper methods

Rules:
- ALL migrations must be idempotent (`CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`)
- Never use bare `CREATE TABLE` — it fails on restart
- Support both SQLite and PostgreSQL dialects where they differ
- Test: restart the server multiple times; migrations must not fail

## Key File Locations

| What | Where |
|---|---|
| CLI commands | `cmd/gantry/{serve,apply,get,describe,export,run,version}.go` |
| Route registration | `internal/api/server.go` |
| All handlers | `internal/api/handlers/*.go` |
| Auth middleware | `internal/api/middleware/auth.go` |
| DB queries | `internal/db/queries.go` |
| DB migrations | `internal/db/migrations.go` |
| Entity schemas (JSON) | `internal/entity/schemas/*.json` |
| Schema validator | `internal/entity/validator.go` |
| Plugin manifests | `internal/plugins/manifest.go` |
| Bundled registry | `internal/plugins/bundled/registry.json` |
| Config | `internal/config/config.go` |
| Frontend API client | `web/src/lib/api.ts` |
| Frontend types | `web/src/lib/types.ts` |
| Frontend routing | `web/src/App.tsx` |
| CSS variables | `web/src/index.css` |
| Plugin runtime hooks | `web/src/lib/plugin-runtime.ts` |

## Common Patterns

### Reading Auth Claims in a Handler

```go
claims := auth.ClaimsFromContext(r.Context())
// claims.UserID, claims.Username, claims.Role
```

### Checking Minimum Role

In `server.go`, wrap routes with the role middleware:

```go
r.With(requireRole("developer")).Post("/my-endpoint", h.MyHandler)
```

Or check inline in the handler for dynamic role checks.

### Writing an Audit Entry

```go
h.db.InsertAuditEntry(audit.Entry{
    UserName:     claims.Username,
    Action:       "create",    // create | update | delete | login | execute | ...
    ResourceType: entity.Kind,
    ResourceName: entity.Metadata.Name,
    Source:       "api",       // api | cli | plugin | system
    IPAddress:    clientIP(r),
    BeforeState:  beforeJSON,  // nil for creates
    AfterState:   afterJSON,   // nil for deletes
})
```

### Publishing an Event

```go
h.events.Publish("entity.created", entity)
h.events.Publish("entity.updated", entity)
h.events.Publish("entity.deleted", map[string]string{"kind": kind, "name": name})
```

### Encrypting Plugin Config

Sensitive plugin config values (tokens, keys) are encrypted before DB storage:

```go
// The plugin manager handles this automatically via config PUT endpoint.
// If writing directly, use the crypto package:
encrypted, err := crypto.Encrypt(plaintext, h.encryptionKey)
decrypted, err := crypto.Decrypt(encrypted, h.encryptionKey)
```

## Build Verification

Before submitting any change, verify these all pass:

```bash
# Go build (must succeed with CGO_ENABLED=0)
CGO_ENABLED=0 go build ./...

# Tests
go test ./...

# Race condition check
go test -race ./...

# Frontend type check
cd web && npx tsc --noEmit

# Frontend build
cd web && npm run build

# Lint
golangci-lint run ./...
```

## What NOT To Do

- **Don't add CGO dependencies** — the binary must build with `CGO_ENABLED=0`
- **Don't add external dependencies without discussion** — check go.mod first
- **Don't use undefined CSS variables** — causes invisible/broken UI
- **Don't add `additionalProperties: true` to entity schemas** — this breaks the strict validation model
- **Don't use `db.sql.*` directly** — use the `d.exec()` / `d.queryRow()` helpers
- **Don't skip audit entries on mutations** — every create/update/delete needs an audit record
- **Don't hardcode colors in React components** — use `var(--gantry-*)` CSS variables
- **Don't create new npm packages for functionality already in the standard library or existing deps**
- **Don't modify migrations to be non-idempotent** — every migration runs on every server start

## Asking for Clarification

If a task is ambiguous or you're unsure whether a change aligns with project goals, the best signals are:

1. **CLAUDE.md** — authoritative project instructions
2. **Existing patterns in the codebase** — follow what's already there
3. **This document** — for Gantry-specific rules
4. **Open a GitHub Discussion** or issue for architectural questions

When in doubt, choose the simpler solution, match existing patterns, and note any trade-offs in the PR description.
