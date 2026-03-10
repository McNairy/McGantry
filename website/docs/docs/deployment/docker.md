---
sidebar_position: 2
title: Docker
description: Run Gantry with Docker or Docker Compose.
---

# Docker

Gantry publishes multi-platform Docker images to GitHub Container Registry (`ghcr.io`).

## Image Tags

| Tag | Description |
|---|---|
| `ghcr.io/go2engle/gantry:latest` | Latest stable release |
| `ghcr.io/go2engle/gantry:v0.1.0` | Specific version |
| `ghcr.io/go2engle/gantry:main` | Latest build from `main` branch |

Supported platforms: `linux/amd64`, `linux/arm64`

## Quick Start

```bash
docker run \
  -p 8080:8080 \
  -v gantry-data:/data \
  -e GANTRY_DATA_DIR=/data \
  -e GANTRY_ADMIN_PASSWORD=changeme \
  ghcr.io/go2engle/gantry:latest
```

Open [http://localhost:8080](http://localhost:8080).

## Docker Compose

### Minimal (SQLite)

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
      GANTRY_ADMIN_PASSWORD: ${GANTRY_ADMIN_PASSWORD:-changeme}
      GANTRY_JWT_SECRET: ${GANTRY_JWT_SECRET}
      GANTRY_ENCRYPTION_KEY: ${GANTRY_ENCRYPTION_KEY}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 5s
      retries: 3

volumes:
  gantry-data:
```

### With PostgreSQL

```yaml
# docker-compose.yml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: gantry
      POSTGRES_USER: gantry
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "gantry"]
      interval: 10s
      timeout: 5s
      retries: 5

  gantry:
    image: ghcr.io/go2engle/gantry:latest
    ports:
      - "8080:8080"
    volumes:
      - gantry-data:/data
    environment:
      GANTRY_DB: postgres://gantry:${POSTGRES_PASSWORD}@postgres:5432/gantry?sslmode=disable
      GANTRY_DATA_DIR: /data
      GANTRY_ADMIN_PASSWORD: ${GANTRY_ADMIN_PASSWORD}
      GANTRY_JWT_SECRET: ${GANTRY_JWT_SECRET}
      GANTRY_ENCRYPTION_KEY: ${GANTRY_ENCRYPTION_KEY}
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

volumes:
  postgres-data:
  gantry-data:
```

### Environment File

Create a `.env` file alongside `docker-compose.yml`:

```bash
# .env
GANTRY_ADMIN_PASSWORD=my-secure-password
GANTRY_JWT_SECRET=a64characterrandomhexstringhere
GANTRY_ENCRYPTION_KEY=another64characterrandomhexstring
POSTGRES_PASSWORD=db-password
```

```bash
docker compose up -d
```

## Kubernetes (Helm — Planned)

A Helm chart is planned. Until then, use this manifest as a starting point:

```yaml
# gantry-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gantry
  namespace: platform
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gantry
  template:
    metadata:
      labels:
        app: gantry
    spec:
      containers:
        - name: gantry
          image: ghcr.io/go2engle/gantry:latest
          ports:
            - containerPort: 8080
          env:
            - name: GANTRY_DB
              value: /data/gantry.db
            - name: GANTRY_DATA_DIR
              value: /data
            - name: GANTRY_ADMIN_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: gantry-secrets
                  key: admin-password
            - name: GANTRY_JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: gantry-secrets
                  key: jwt-secret
            - name: GANTRY_ENCRYPTION_KEY
              valueFrom:
                secretKeyRef:
                  name: gantry-secrets
                  key: encryption-key
          volumeMounts:
            - name: data
              mountPath: /data
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8080
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8080
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: gantry-pvc
```

## Data Persistence

Mount a volume to `GANTRY_DATA_DIR` (default `/data` in the container). This directory contains:
- `gantry.db` — SQLite database
- `encryption.key` — Encryption key for plugin configs

:::danger Back Up Your Data
Back up the data volume regularly, especially `encryption.key`. Losing the encryption key makes all plugin credentials unrecoverable.
:::

## Building Your Own Image

```dockerfile
# Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# Build frontend first
COPY web/ ./web/
RUN apk add --no-cache nodejs npm
RUN cd web && npm ci && npm run build

# Build binary
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /gantry ./cmd/gantry

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /gantry /usr/local/bin/gantry

EXPOSE 8080
VOLUME ["/data"]
ENV GANTRY_DATA_DIR=/data

CMD ["gantry", "serve"]
```

```bash
docker build -t my-gantry .
docker run -p 8080:8080 my-gantry
```
