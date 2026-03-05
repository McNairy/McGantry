.PHONY: build run test lint clean dev frontend backend

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
