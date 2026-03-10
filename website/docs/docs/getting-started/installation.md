---
sidebar_position: 1
title: Installation
description: Install Gantry via binary, Docker, or from source.
---

# Installation

Gantry ships as a single self-contained binary. There is no Node.js, no PostgreSQL, no Kubernetes — just one executable.

## Option 1: Install Script (Linux / macOS)

```bash
curl -sSL https://github.com/go2engle/gantry/releases/latest/download/install.sh | sh
```

This downloads the latest binary for your OS/arch to `/usr/local/bin/gantry`.

Verify:

```bash
gantry version
# gantry v0.1.0 (commit abc1234, built 2025-01-01)
```

## Option 2: Manual Download

Download the binary for your platform from [GitHub Releases](https://github.com/go2engle/gantry/releases/latest):

| Platform | File |
|---|---|
| Linux amd64 | `gantry_linux_amd64.tar.gz` |
| Linux arm64 | `gantry_linux_arm64.tar.gz` |
| macOS amd64 | `gantry_darwin_amd64.tar.gz` |
| macOS arm64 (Apple Silicon) | `gantry_darwin_arm64.tar.gz` |
| Windows amd64 | `gantry_windows_amd64.zip` |

Extract and move to your `PATH`:

```bash
tar -xzf gantry_linux_amd64.tar.gz
sudo mv gantry /usr/local/bin/
```

## Option 3: Docker

```bash
docker run -p 8080:8080 ghcr.io/go2engle/gantry:latest
```

With a persistent data volume:

```bash
docker run \
  -p 8080:8080 \
  -v /path/to/data:/data \
  -e GANTRY_DB=/data/gantry.db \
  -e GANTRY_DATA_DIR=/data \
  ghcr.io/go2engle/gantry:latest
```

## Option 4: Docker Compose

```yaml
# docker-compose.yml
version: '3.8'
services:
  gantry:
    image: ghcr.io/go2engle/gantry:latest
    ports:
      - "8080:8080"
    volumes:
      - gantry-data:/data
    environment:
      GANTRY_DB: /data/gantry.db
      GANTRY_DATA_DIR: /data
      GANTRY_ADMIN_PASSWORD: your-secure-password
      GANTRY_JWT_SECRET: your-jwt-secret

volumes:
  gantry-data:
```

```bash
docker compose up -d
```

## Option 5: Build from Source

Requirements: Go 1.22+, Node.js 20+

```bash
git clone https://github.com/go2engle/gantry.git
cd gantry

# Build frontend
cd web && npm install && npm run build && cd ..

# Build binary
go build -o bin/gantry ./cmd/gantry

# Run
./bin/gantry serve --dev
```

See [Development Setup](../contributing/development-setup) for a full local dev environment.

## Starting the Server

```bash
# Start with defaults (port 8080, SQLite at ./data/gantry.db)
gantry serve

# Start in development mode (verbose logging, open CORS)
gantry serve --dev

# Custom port and database
gantry serve --port 9000 --db ./mydata/gantry.db

# Change the default admin password
gantry serve --admin-password my-secure-password
```

Open [http://localhost:8080](http://localhost:8080) in your browser. Log in with:
- **Username:** `admin`
- **Password:** `changeme` (or your `--admin-password` value)

:::info First Run
On first start, Gantry automatically creates the SQLite database, runs migrations, creates the default `admin` user, and loads built-in entity schemas. No setup required.
:::

## Next Steps

- [Quick Start](./quick-start) — Create your first entities and apply a catalog
- [Configuration](./configuration) — Configure via env vars or config file
