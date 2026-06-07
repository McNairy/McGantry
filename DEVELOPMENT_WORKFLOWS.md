# Development Workflows

## Repository Overview

| Layer | Repo | Purpose |
|---|---|---|
| Core | `McNairy/McGantry` (GitHub) | The application; external plugin system baked in |
| Contract | `mcxample/gantry-plugin-sdk` (Forgejo) | Interface between Gantry and plugins |
| Implementations | `mcxample/gantry-plugin-*` (Forgejo) | Individual plugins, one binary each |
| Tooling | `mcxample/gantry-plugin-tools` (Forgejo) | `gantry-new-plugin`, `gantry-migrate-plugin` |

At startup Gantry scans the plugins directory and spawns each binary as a subprocess over gRPC.

---

## New integration (greenfield)

```bash
gantry-new-plugin my-thing
cd gantry-plugin-my-thing
# implement plugin logic
go mod tidy && go mod vendor
go build -o gantry-plugin-my-thing .
cp gantry-plugin-my-thing /path/to/plugins/
# enable in Gantry UI
```

## Migrate an old-style plugin to external

Point the tool at any plugin source directory written for the old internal plugin system:

```bash
gantry-migrate-plugin --src ./my-plugin
cd gantry-plugin-my-plugin
go mod tidy && go mod vendor
go build -o gantry-plugin-my-plugin .
cp gantry-plugin-my-plugin /path/to/plugins/
# enable in Gantry UI
```

The tool auto-detects the plugin name, entity import path, and SyncResult field mappings from the source. Optional flags:

```
--name        override plugin name (default: detected from package declaration)
--title       display title shown in UI
--description plugin description
--category    plugin category (default: integration)
--module      Go module path (default: mcxample/gantry-plugin-<name>)
--output      output directory (default: ./gantry-plugin-<name>)
--entity-src  path to entity package if not using embedded snapshot
```

To migrate a plugin that is part of the McGantry source (e.g. `internal/plugins/argocd`):

```bash
gantry-migrate-plugin --src ./internal/plugins/argocd --title "ArgoCD"
# then remove internal/plugins/argocd/ from McGantry source
```

## Changing the plugin interface

```bash
# 1. Edit mcxample/gantry-plugin-sdk
# 2. Edit McGantry host code (internal/api/handlers/plugins.go, etc.)
# 3. Rebuild affected plugins — the replace directive picks up SDK changes automatically
go build -o gantry-plugin-<name> .
```

## Iterating on an existing plugin

```bash
# edit plugin source in mcxample/gantry-plugin-<name>
go build -o gantry-plugin-<name> .
cp gantry-plugin-<name> /path/to/plugins/
# restart Gantry
```

## Shipping a plugin release

```bash
git tag v1.2.3 && git push origin v1.2.3
# Forgejo CI builds the binary and uploads it as a release asset
```

---

## Notes

- The SDK `replace` directive in each plugin's `go.mod` points to the local SDK path, so SDK changes are picked up automatically during development — no version bumping needed.
- If you change what Gantry expects from plugins, update the SDK first, then the McGantry host, then affected plugins.
