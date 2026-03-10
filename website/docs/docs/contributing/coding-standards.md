---
sidebar_position: 4
title: Coding Standards
description: Code style, patterns, and rules for Gantry contributors.
---

# Coding Standards

These standards apply to all code in this repository. Understanding them before contributing saves time in code review.

## Go

### General

- Follow standard Go formatting — run `gofmt` or `goimports` before committing
- `go vet ./...` must pass
- `golangci-lint run ./...` must pass (configuration in `.golangci.yml`)
- No unused imports; no unused variables

### Error Handling

Return errors — don't silently ignore them.

```go
// Good
result, err := d.queryRow(...)
if err != nil {
    return fmt.Errorf("get entity: %w", err)
}

// Bad
result, _ := d.queryRow(...)
```

Wrap errors with context using `fmt.Errorf("context: %w", err)`. Don't re-wrap already-wrapped errors.

### Database Queries

Use the `db.go` helper methods — not `db.sql.*` directly:

```go
// Good
if err := d.exec(query, args...); err != nil { ... }
row, err := d.queryRow(dest, query, args...)

// Bad
_, err := d.sql.ExecContext(ctx, query, args...)
```

**SQLite booleans** — SQLite has no native boolean type. Use `boolToInt()` for writes and `int-to-bool` conversion for reads:

```go
// Write
_, err := d.exec(`UPDATE plugins SET enabled = ? WHERE name = ?`, boolToInt(enabled), name)

// Read
var enabledInt int
// scan into enabledInt, then:
plugin.Enabled = enabledInt != 0
```

**JSON fields** — Entity `tags`, `annotations`, `labels`, `spec`, and plugin `config` are stored as TEXT JSON:

```go
// Marshal before write
specJSON, err := json.Marshal(entity.Spec)

// Unmarshal after read
if err := json.Unmarshal([]byte(specJSON), &entity.Spec); err != nil { ... }
```

### Handlers

All handlers are methods on `handlers.Handlers`. Pattern:

```go
func (h *Handlers) CreateEntity(w http.ResponseWriter, r *http.Request) {
    // 1. Parse and validate request
    var req EntityRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }

    // 2. Validate schema
    if err := h.validator.Validate(req.Kind, req.Spec); err != nil {
        http.Error(w, err.Error(), http.StatusUnprocessableEntity)
        return
    }

    // 3. DB operation
    entity, err := h.db.CreateEntity(req)
    if err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }

    // 4. Audit log — always capture IP
    h.db.InsertAuditEntry(audit.Entry{
        Action:       "create",
        ResourceType: req.Kind,
        ResourceName: req.Metadata.Name,
        AfterState:   entity,
        IPAddress:    clientIP(r),   // always use this helper
        UserName:     claimsFromCtx(r.Context()).Username,
    })

    // 5. Publish event
    h.events.Publish("entity.created", entity)

    // 6. Respond
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(entity)
}
```

### Auth Context

Claims are stored in request context by the auth middleware. Access them:

```go
claims := auth.ClaimsFromContext(r.Context())
if claims == nil {
    http.Error(w, "unauthorized", http.StatusUnauthorized)
    return
}
// claims.UserID, claims.Username, claims.Role
```

### Client IP

Always use the `clientIP(r)` helper for audit entries — it respects `X-Real-IP` and `X-Forwarded-For`:

```go
ip := clientIP(r)
```

### No New External Dependencies Without Discussion

Every new Go dependency must be justified in the PR. We deliberately keep the dependency tree small:
- Check `go.mod` before `go get`-ing
- Prefer standard library solutions
- Never add CGO-dependent packages

## Entity Schemas

### Rules for `internal/entity/schemas/*.json`

- All schemas MUST have `"additionalProperties": false` on the `spec` object
- All schemas MUST validate with JSON Schema draft 2020-12
- Keep enum values as `snake_case` or `kebab-case` strings (no camelCase)
- Use `description` on every field for frontend form generation
- New spec fields MUST NOT break existing entities (add as optional, never required on existing kinds)

```json
{
  "properties": {
    "spec": {
      "type": "object",
      "additionalProperties": false,   // Required
      "properties": {
        "myNewField": {
          "type": "string",
          "description": "What this field does"   // Required for good UX
        }
      }
    }
  }
}
```

## TypeScript / React

### TypeScript

- All new code must be typed — no `any` unless absolutely unavoidable
- `npx tsc --noEmit` must pass with zero errors
- Use interfaces for data structures, types for unions/intersections

### API Client

All HTTP calls go through the `api` object in `web/src/lib/api.ts`. Never use `fetch()` directly in components:

```typescript
// Good
const entities = await api.getEntities();

// Bad
const res = await fetch('/api/v1/entities', {
  headers: { Authorization: `Bearer ${token}` }
});
```

### Types

Add types to `web/src/lib/types.ts`. Mirror the Go structs:

```typescript
// types.ts
export interface MyNewType {
  id: string;
  name: string;
  createdAt: string;  // ISO 8601 string (Go time.Time serializes this way)
}
```

### CSS / Theming

Use CSS custom properties for all colors — no hardcoded hex values:

```tsx
// Good
<div style={{ background: 'var(--gantry-bg-secondary)' }}>

// Bad
<div style={{ background: '#1c1c1e' }}>
```

For Tailwind + CSS variables, use the bracket notation:

```tsx
// Good — transparent accent background
<div className="bg-[var(--gantry-accent)]/10">

// Bad — bg-opacity doesn't work with CSS variables
<div className="bg-[var(--gantry-accent)] bg-opacity-10">
```

Only use these defined CSS variables (others do not exist):
- `--gantry-bg-primary`, `--gantry-bg-secondary`, `--gantry-bg-tertiary`
- `--gantry-text-primary`, `--gantry-text-secondary`
- `--gantry-border`
- `--gantry-accent`, `--gantry-accent-hover`
- `--gantry-danger`

### Component Patterns

- Use `ErrorBoundary` around new route-level components
- Put new pages in `web/src/pages/`, shared components in `web/src/components/`
- New API calls go in `api.ts` first, then used in components
- Use `lucide-react` for icons — do not add new icon libraries

## Migrations

Database migrations in `internal/db/migrations.go`:

```go
// Good — idempotent
db.Exec(`CREATE TABLE IF NOT EXISTS my_table (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
)`)

// Good — idempotent index
db.Exec(`CREATE INDEX IF NOT EXISTS idx_my_table_name ON my_table(name)`)

// Never use bare CREATE TABLE — it fails if table exists
db.Exec(`CREATE TABLE my_table (...)`) // Bad
```

Migrations run on every startup. They must be **safe to re-run** at any time.

Support both SQLite and PostgreSQL dialects if the query differs:

```go
if d.dialect == "postgres" {
    db.Exec(`CREATE TABLE IF NOT EXISTS ... (id UUID DEFAULT gen_random_uuid() PRIMARY KEY, ...)`)
} else {
    db.Exec(`CREATE TABLE IF NOT EXISTS ... (id TEXT PRIMARY KEY, ...)`)
}
```

## Testing

- Write tests for any new handler, DB query, or validation logic
- Use real SQLite in-memory DB for handler tests — no mocks of the DB layer
- Table-driven tests for schema validation:

```go
func TestServiceSchemaValidation(t *testing.T) {
    tests := []struct {
        name    string
        spec    map[string]any
        wantErr bool
    }{
        {"valid spec", map[string]any{"type": "backend", "lifecycle": "production"}, false},
        {"invalid type", map[string]any{"type": "invalid"}, true},
        {"unknown field", map[string]any{"type": "backend", "unknown": "field"}, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validator.Validate("Service", tt.spec)
            if (err != nil) != tt.wantErr {
                t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
            }
        })
    }
}
```

## Security Checklist

Before submitting any PR, verify:

- [ ] No SQL injection — use parameterized queries (`?` placeholders), never string concatenation
- [ ] No hardcoded secrets or credentials in code or tests
- [ ] Auth checks on all new endpoints (`requireAuth`, `requireRole`)
- [ ] Audit log entry for any mutation (create, update, delete)
- [ ] Client IP captured in audit entries via `clientIP(r)`
- [ ] No user-controlled values interpolated into file paths or shell commands
- [ ] Plugin config encryption — any new sensitive config fields must use the encryption helpers
