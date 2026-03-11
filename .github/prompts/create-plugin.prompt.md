---
agent: agent
description: Create a new Gantry plugin end-to-end (backend handler, registry, routes, frontend page, sidebar, dashboard widget)
---

# Create a new Gantry plugin

You are creating a new plugin for the Gantry IDP. Follow this checklist exactly — every plugin touches up to 11 files. Use the `status-monitor` plugin (`internal/api/handlers/status_monitor.go`, `web/src/pages/StatusMonitor.tsx`) as a complete reference implementation.

## Plugin description from user

{{{ input }}}

## Step-by-step instructions

Work through these steps in order. After each file, verify it compiles/type-checks before moving on.

### Step 1: Backend Handler — `internal/api/handlers/<plugin_name>.go`

- Package: `handlers`
- All handlers are methods on `*Handlers` struct: `func (h *Handlers) GetMyData(w http.ResponseWriter, r *http.Request)`
- First check plugin is installed and enabled:
  ```go
  p, err := h.DB.GetPlugin(r.Context(), "<plugin-name>")
  if err != nil || p == nil { writeError(w, http.StatusNotFound, "plugin not installed"); return }
  if !p.Enabled { writeError(w, http.StatusBadRequest, "plugin not enabled"); return }
  ```
- Read custom config from `p.Config` (`map[string]any`)
- For external HTTP calls: use `http.NewRequest` + `client.Do(req)`, always set `User-Agent: Gantry/1.0 <PluginName>` header
- For expensive operations: use in-memory cache with `sync.RWMutex` and a TTL (e.g., 60s)
- For concurrent work: `sync.WaitGroup` + goroutines with `sync.Mutex` to protect shared state
- Use `writeJSON(w, statusCode, data)` and `writeError(w, statusCode, message)` helpers

### Step 2: Registry Entry — `internal/plugins/bundled/registry.json`

Append a new entry with ALL required fields:

```json
{
  "name": "<plugin-name>",
  "title": "<Plugin Title>",
  "description": "Short description for plugin list.",
  "longDescription": "Detailed description for plugin detail view.",
  "features": ["Feature 1", "Feature 2"],
  "version": "1.0.0",
  "author": "Gantry",
  "category": "integration|widget|action-type",
  "homepage": "https://github.com/go2engle/gantry",
  "configSchema": { "type": "object", "properties": { ... } },
  "entityPanels": [],
  "actionTypes": []
}
```

### Step 3: Route Registration — `internal/api/server.go`

Add routes in the `protected` group, BEFORE wildcard/SPA routes. Look for where other plugin routes are registered (search for `/plugins/`):

```go
protected.Get("/plugins/<plugin-name>/<endpoint>", h.HandlerMethod)
```

### Step 4: Frontend Types — `web/src/lib/types.ts`

Add exported TypeScript interfaces for API response types.

### Step 5: API Client — `web/src/lib/api.ts`

Add methods to the `api` object:

```typescript
getMyPluginData: () => request<MyType>('GET', '/plugins/<plugin-name>/data'),
```

### Step 6: Plugin Page — `web/src/pages/<PluginPage>.tsx`

If the plugin has a sidebar entry, create a full-page component:

- Use loading state, error handling, auto-refresh with `setInterval`
- CSS: ONLY use `--gantry-*` custom properties: `bg-primary`, `bg-secondary`, `bg-tertiary`, `text-primary`, `text-secondary`, `border`, `accent`, `accent-hover`, `danger`
- For text on accent backgrounds: use `text-[var(--gantry-bg-primary)]` NOT `text-white`
- Tailwind opacity: `bg-[var(--gantry-accent)]/10` NOT `bg-opacity-10`
- Icons: import from `lucide-react`

### Step 7: Route — `web/src/App.tsx`

Add a `<Route>` inside the authenticated routes section.

### Step 8: Sidebar Entry — `web/src/components/Sidebar.tsx`

- Add state to track if plugin is enabled
- In the existing `useEffect` that checks plugins, add check for this plugin
- Conditionally render a nav item with a lucide-react icon

### Step 9: Dashboard Widget — `web/src/pages/Dashboard.tsx`

1. Add to `WIDGET_LABELS`: `'<widget_id>': '<Widget Label>'`
2. Add to `DEFAULT_WIDGETS`: `{ id: '<widget_id>', visible: true, order: <next>, width: 'full' }`
3. Add state + fetch call in the main `useEffect` / `Promise.all`
4. Add `case '<widget_id>':` in `renderWidget()` — return `null` if no data

### Step 10: Backend Widget Validation — `internal/api/handlers/dashboard.go`

Add widget ID to `knownWidgetIDs` map:

```go
"<widget_id>": true,
```

### Step 11: Entity Detail Tab (if applicable)

Create `web/src/components/<PluginTab>.tsx` for entity detail panel tabs.

## Verification

After all files are created/edited, run:

```bash
go build ./...
cd web && npx tsc --noEmit
```

## Common Pitfalls

- **CSS variables:** only 9 `--gantry-*` properties exist — undefined ones render invisible
- **Dark mode contrast:** `text-[var(--gantry-bg-primary)]` on accent backgrounds, never `text-white`
- **Tailwind opacity:** `bg-[var(--gantry-accent)]/10` not `bg-opacity-10`
- **Entity schemas:** `additionalProperties: false` — check allowed spec fields
- **External HTTP:** always set User-Agent header
- **Widget ID:** must be in `knownWidgetIDs` in `dashboard.go` or save fails
- **Route ordering:** plugin routes must come BEFORE wildcard routes in `server.go`
- **Registry JSON:** must include `configSchema`, `entityPanels`, `actionTypes`
