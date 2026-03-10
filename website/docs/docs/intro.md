---
slug: /
sidebar_position: 1
title: Introduction
description: What is Gantry and why should you use it?
---

# Gantry

**Gantry** is an open-source internal developer platform (IDP) that ships as a **single Go binary** with zero external dependencies. It gives engineering teams a unified service catalog, self-service actions, GitOps-native configuration, and a plugin ecosystem — without the operational complexity of alternatives like Backstage.

## What Gantry Does

```
┌─────────────────────────────────────────────────────┐
│                     Gantry IDP                       │
│                                                     │
│  Service Catalog  │  Actions  │  Plugins  │  Search │
│─────────────────────────────────────────────────────│
│              Single Go Binary                        │
│         Embedded SQLite  ·  JWT Auth                │
│    Kubernetes  ·  GitHub  ·  ArgoCD  ·  More        │
└─────────────────────────────────────────────────────┘
```

- **Service Catalog** — A single source of truth for every service, API, team, infrastructure component, and environment in your org. Entities are typed, validated, and searchable.
- **Self-Service Actions** — Define runbooks and workflows as executable actions with schema-driven forms. Developers can trigger deployments, provision environments, or run scripts without opening a ticket.
- **Plugin Ecosystem** — Connect Kubernetes clusters, GitHub repositories, ArgoCD applications, and more. Plugins sync entities automatically and add live context (pod logs, PRs, sync status) into the catalog.
- **GitOps Native** — Manage your catalog with YAML files and `gantry apply`. Your catalog becomes a Git-versioned artifact you can review, diff, and roll back.
- **Full-Text Search** — Find any entity in milliseconds via SQLite FTS5. No Elasticsearch required.

## Why Not Backstage?

Backstage is powerful, but operating it is a full-time job:

| | Gantry | Backstage |
|---|---|---|
| Setup time | ~5 minutes | Hours to days |
| Runtime dependencies | None — single binary | Node.js, PostgreSQL, often Kubernetes |
| Hosting | Any server or Docker | Kubernetes recommended |
| Embedded database | SQLite, zero config | External DB required |
| GitOps apply | `gantry apply` built-in | YAML ingestion via plugins |

Gantry is built for teams that want the *benefits* of an IDP without dedicated platform engineering headcount to maintain it.

## Quick Start

```bash
# Download and run (Linux/macOS)
curl -sSL https://github.com/go2engle/gantry/releases/latest/download/install.sh | sh
gantry serve

# Or with Docker
docker run -p 8080:8080 ghcr.io/go2engle/gantry:latest
```

Open [http://localhost:8080](http://localhost:8080). Default credentials: `admin` / `changeme`.

See the [Installation guide](./getting-started/installation) for more options.

## Key Concepts

| Concept | Description |
|---|---|
| [Entity](./concepts/entity-model) | The core unit — a typed YAML object with `kind`, `metadata`, and `spec` |
| [Kind](./concepts/entity-model#kinds) | The type of entity: Service, API, Team, Environment, Infrastructure, Action, Documentation |
| [Action](./concepts/actions) | An executable workflow with schema-driven inputs |
| [Plugin](./plugins/overview) | A first-party or community extension that connects external systems |

## Project Status

Gantry is actively developed and approaching its first stable release. The core API (entities, actions, auth, plugins) is stable. GitOps reconciliation (push/pull paths, GitHub webhooks) is in progress.

- Apache 2.0 licensed
- Contributions welcome — see [Contributing](./contributing/overview)
- Issues and feature requests: [github.com/go2engle/gantry/issues](https://github.com/go2engle/gantry/issues)
