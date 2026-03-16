---
sidebar_position: 1
title: Installation
description: Install Gantry via binary, Docker, or from source.
---

# Installation

Gantry ships as a single self-contained binary. There is no Node.js, no PostgreSQL, no Kubernetes — just one executable.

## Option 1: One-Line Installer (Linux / macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/go2engle/gantry/main/install.sh | sh
```

This script:

- Detects your OS and CPU architecture
- Downloads the matching Gantry release archive
- Verifies its SHA-256 checksum
- Extracts the binary and runs `gantry install`

`gantry install` then handles the actual machine setup: creating the Gantry service, writing the env file, copying the binary into `/usr/local/bin/gantry`, and starting the service.

Verify:

```bash
gantry version
```

Open [http://localhost:8080](http://localhost:8080) after the installer finishes.

Pass any `gantry install` flags through the script with `sh -s --`:

```bash
curl -fsSL https://raw.githubusercontent.com/go2engle/gantry/main/install.sh | sh -s -- --port 9090 --no-start
```

For non-interactive installs, set `GANTRY_ADMIN_PASSWORD` on the shell that runs the script:

```bash
curl -fsSL https://raw.githubusercontent.com/go2engle/gantry/main/install.sh | env GANTRY_ADMIN_PASSWORD='replace-me' sh -s -- --port 9090
```

To install a specific release instead of the latest one:

```bash
curl -fsSL https://raw.githubusercontent.com/go2engle/gantry/main/install.sh | env GANTRY_VERSION='v0.2.0' sh
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

Extract and run the built-in installer:

```bash
tar -xzf gantry_linux_amd64.tar.gz
sudo ./gantry install
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

After installation, open [http://localhost:8080](http://localhost:8080) in your browser. Log in with:
- **Username:** `admin`
- **Password:** the password you entered during install, or `changeme` if you skipped it

If you used `--no-start`, start the service manually:

```bash
sudo systemctl start gantry
```

On macOS:

```bash
sudo launchctl load -w /Library/LaunchDaemons/com.gantry.server.plist
```

:::info First Run
On first start, Gantry automatically creates the SQLite database, runs migrations, creates the default `admin` user, and loads built-in entity schemas. No setup required.
:::

## Next Steps

- [Quick Start](./quick-start) — Create your first entities and apply a catalog
- [Configuration](./configuration) — Configure via env vars or config file
