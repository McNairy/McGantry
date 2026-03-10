---
sidebar_position: 1
title: Contributing Overview
description: How to contribute to Gantry — for humans and AI.
---

# Contributing to Gantry

Thank you for your interest in contributing! Gantry is Apache 2.0 licensed and welcomes contributions from developers and AI agents alike.

## Ways to Contribute

- **Bug reports** — [Open an issue](https://github.com/go2engle/gantry/issues/new?template=bug_report.md) with reproduction steps
- **Feature requests** — [Open a discussion](https://github.com/go2engle/gantry/discussions) or issue describing the use case
- **Pull requests** — Bug fixes, features, documentation improvements, tests
- **Documentation** — Improve or expand these docs
- **Plugins** — Build and share community plugins

## Before You Start

For significant changes (new features, architecture changes, new plugin integrations), **open an issue first** to discuss the approach. This avoids duplicate work and ensures the change aligns with the project direction.

Small changes (bug fixes, documentation improvements, test additions) can go straight to a PR.

## Pull Request Process

1. **Fork** the repository and create a branch from `main`
2. **Read** [Development Setup](./development-setup) and get your local environment running
3. **Make your changes** following the [Coding Standards](./coding-standards)
4. **Write or update tests** for any behavior you change
5. **Run the test suite** — `go test ./...` must pass
6. **Run the type checker** — `cd web && npx tsc --noEmit` must pass
7. **Open a PR** against `main` with a clear description of what and why

## PR Description Template

```markdown
## What

Brief description of what this PR changes.

## Why

Why is this change needed? Link to issue if applicable.

## How

Key implementation decisions or trade-offs.

## Testing

How you tested this. What test cases were added.
```

## Commit Style

Use conventional commits for easy changelog generation:

```
feat: add kubernetes log streaming endpoint
fix: correct JSON serialization of entity tags
docs: add argocd plugin configuration guide
refactor: extract clientIP helper to middleware package
test: add entity schema validation test cases
chore: update golangci-lint to v1.57
```

**Types:** `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`

## Code Review

All PRs require at least one review before merging. Reviewers will look for:

- Correctness and edge cases
- Test coverage for new behavior
- Adherence to [Coding Standards](./coding-standards)
- Impact on performance and binary size
- Security implications

## What We Don't Accept

- Breaking API changes without deprecation path
- New external Go dependencies without strong justification (keeping the binary lean is a core goal)
- Frontend changes that introduce new npm dependencies without discussion
- Changes that require CGO (the binary must build with `CGO_ENABLED=0`)

## License

By contributing, you agree that your contributions will be licensed under the [Apache 2.0 License](https://github.com/go2engle/gantry/blob/main/LICENSE).
