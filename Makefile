.PHONY: build run test lint clean dev frontend backend proto

# Build the entire project (frontend + backend)
build: frontend backend

# Build only the Go binary (embeds frontend assets)
backend:
	go build -o bin/gantry ./cmd/gantry

# Build the frontend
frontend:
	cd web && npm install && npm run build

# Run the server in development mode
dev:
	go run ./cmd/gantry serve --dev

# Live reload dev (requires: go install github.com/air-verse/air@latest)
# Backend auto-reloads on .go changes; frontend has HMR at http://localhost:3000
AIR := $(shell go env GOPATH)/bin/air

dev-watch:
	@test -f $(AIR) || (echo "air not found — run: go install github.com/air-verse/air@latest" && exit 1)
	@echo "Starting backend (air) on :8080 and frontend (vite) on :8080..."
	@echo "Open http://localhost:3000 in your browser"
	@trap 'kill 0' INT TERM; \
		$(AIR) & \
		(cd web && npm run dev) & \
		wait

# Run all tests
test:
	go test ./...

# Run a single test (usage: make test-one TEST=TestEntityCreate)
test-one:
	go test ./... -run $(TEST) -v

# Run frontend tests
test-frontend:
	cd web && npm test

# Lint Go code
lint:
	golangci-lint run ./...

# Lint frontend
lint-frontend:
	cd web && npm run lint

# Clean build artifacts
clean:
	rm -rf bin/ web/dist/

# Run frontend dev server (with hot reload)
dev-frontend:
	cd web && npm run dev

# Format Go code
fmt:
	gofmt -w .

# Generate protobuf/gRPC code for the external plugin system.
# Requires: protoc, protoc-gen-go, protoc-gen-go-grpc
proto:
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		internal/plugins/external/proto/plugin.proto
