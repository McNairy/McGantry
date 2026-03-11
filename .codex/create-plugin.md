# Create a new Gantry plugin

This task creates a new plugin for the Gantry IDP end-to-end. Follow this checklist exactly — every plugin touches up to 11 files. Use the `status-monitor` plugin as a complete reference (`internal/api/handlers/status_monitor.go`, `web/src/pages/StatusMonitor.tsx`).

## Files to Create or Modify

| # | File | Action | Purpose |
|---|------|--------|---------|
| 1 | `internal/api/handlers/<plugin>.go` | Create | Backend handler(s) — business logic, external API calls, caching |
| 2 | `internal/plugins/bundled/registry.json` | Edit | Plugin metadata, config schema, entity panels, action types |
| 3 | `internal/api/server.go` | Edit | Register API routes (BEFORE wildcard/SPA routes) |
| 4 | `web/src/lib/types.ts` | Edit | TypeScript interfaces for API responses |
| 5 | `web/src/lib/api.ts` | Edit | API client methods |
| 6 | `web/src/pages/<PluginPage>.tsx` | Create | Full-page UI (if plugin has sidebar entry) |
| 7 | `web/src/App.tsx` | Edit | Add Route for new page |
| 8 | `web/src/components/Sidebar.tsx` | Edit | Conditional nav item (check plugin enabled via API) |
| 9 | `web/src/pages/Dashboard.tsx` | Edit | Dashboard widget (label, default config, render case) |
| 10 | `internal/api/handlers/dashboard.go` | Edit | Add widget ID to `knownWidgetIDs` map |
| 11 | `web/src/components/<PluginTab>.tsx` | Create | Entity detail tab (if plugin adds entity panels) |

## Backend Handler Pattern

```go
package handlers

func (h *Handlers) GetMyPluginData(w http.ResponseWriter, r *http.Request) {
    // 1. Check plugin installed and enabled
    p, err := h.DB.GetPlugin(r.Context(), "my-plugin")
    if err != nil || p == nil {
        writeError(w, http.StatusNotFound, "my-plugin not installed")
        return
    }
    if !p.Enabled {
        writeError(w, http.StatusBadRequest, "my-plugin not enabled")
        return
    }

    // 2. Read custom config from p.Config (map[string]any)
    // 3. Business logic
    // 4. For external HTTP: http.NewRequest + set User-Agent header
    // 5. For expensive ops: cache with sync.RWMutex + TTL
    // 6. writeJSON(w, http.StatusOK, result)
}
```

## Registry Entry Pattern

Each entry in `registry.json` must include ALL fields:

```json
{
  "name": "my-plugin",
  "title": "My Plugin",
  "description": "Short description.",
  "longDescription": "Detailed description.",
  "features": ["Feature 1", "Feature 2"],
  "version": "1.0.0",
  "author": "Gantry",
  "category": "integration|widget|action-type",
  "homepage": "https://github.com/go2engle/gantry",
  "configSchema": { "type": "object", "properties": {} },
  "entityPanels": [],
  "actionTypes": []
}
```

## Dashboard Widget Pattern

In `Dashboard.tsx`:
1. Add to `WIDGET_LABELS`: `my_widget: 'My Widget'`
2. Add to `DEFAULT_WIDGETS`: `{ id: 'my_widget', visible: true, order: N, width: 'full' }`
3. Add state + fetch in `useEffect` / `Promise.all`
4. Add `case 'my_widget':` in `renderWidget()` — return `null` if no data

In `dashboard.go`:
5. Add `"my_widget": true` to `knownWidgetIDs` map

## Frontend Rules

- CSS: ONLY use `--gantry-*` custom properties: `bg-primary`, `bg-secondary`, `bg-tertiary`, `text-primary`, `text-secondary`, `border`, `accent`, `accent-hover`, `danger`
- Text on accent backgrounds: `text-[var(--gantry-bg-primary)]` NOT `text-white`
- Tailwind opacity: `bg-[var(--gantry-accent)]/10` NOT `bg-opacity-10`
- Icons: import from `lucide-react`
- API calls: always through `api` object in `web/src/lib/api.ts`

## Common Pitfalls

- CSS variables: only 9 `--gantry-*` properties exist — undefined ones render invisible
- Entity schemas: `additionalProperties: false` — check allowed spec fields
- External HTTP: always set User-Agent header (some APIs block default Go client)
- Widget ID: must be in `knownWidgetIDs` in `dashboard.go` or dashboard save fails
- Route ordering: plugin routes must come BEFORE wildcard routes in `server.go`
- Registry JSON: missing `configSchema`/`entityPanels`/`actionTypes` causes parse errors

## Verification

After all changes:

```bash
go build ./...              # backend compiles
cd web && npx tsc --noEmit  # frontend type-checks
```
