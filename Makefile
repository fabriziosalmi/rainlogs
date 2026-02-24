################################################################################
# RainLogs – Makefile
################################################################################

BINARY_API    := bin/rainlogs-api
BINARY_WORKER := bin/rainlogs-worker
MODULE        := github.com/fabriziosalmi/rainlogs
GO            := go
GOFLAGS       := -ldflags="-s -w"
VERSION       ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GOFLAGS_VER   := -ldflags="-s -w -X $(MODULE)/internal/config.Version=$(VERSION)"

.PHONY: all build api worker test test-verbose test-unit test-integration \
        lint vet vuln fmt fmt-check clean tidy \
        migrate-up migrate-down migrate-status migrate-create \
        docker-build docker-up docker-down docker-logs \
        garage-init garage-create-bucket \
        dev-api dev-worker

# ─── Build ──────────────────────────────────────────────────────────────────

all: build

build: api worker

api:
	@mkdir -p bin
	$(GO) build $(GOFLAGS_VER) -o $(BINARY_API) ./cmd/api

worker:
	@mkdir -p bin
	$(GO) build $(GOFLAGS_VER) -o $(BINARY_WORKER) ./cmd/worker

# ─── Run (local, requires .env loaded) ──────────────────────────────────────

dev-api: api
	@echo "→ Starting API (set env vars first or copy .env)"
	./$(BINARY_API)

dev-worker: worker
	@echo "→ Starting Worker"
	./$(BINARY_WORKER)

# ─── Test ────────────────────────────────────────────────────────────────────

test: test-unit

test-unit:
	$(GO) test -race -cover -coverprofile=coverage.out ./pkg/... ./internal/...

test-verbose:
	$(GO) test -race -v -cover -coverprofile=coverage.out ./...

test-integration:
	$(GO) test -race -v -timeout=120s ./tests/integration/...

cover: test-unit
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "→ Coverage report: coverage.html"

# ─── Code Quality ────────────────────────────────────────────────────────────

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...
	@command -v goimports >/dev/null 2>&1 && goimports -w -local $(MODULE) . || true

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Run 'make fmt' to fix formatting" && exit 1)

lint:
	@command -v golangci-lint >/dev/null 2>&1 || \
		(echo "golangci-lint not installed – run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...

vuln:
	@command -v govulncheck >/dev/null 2>&1 || \
		$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

# Full quality gate (CI-equivalent)
check: vet lint vuln test

# ─── Migrations ──────────────────────────────────────────────────────────────

DB_URL ?= postgres://rainlogs:changeme@localhost:5432/rainlogs?sslmode=disable

migrate-up:
	@command -v migrate >/dev/null 2>&1 || \
		(echo "migrate not installed – see https://github.com/golang-migrate/migrate" && exit 1)
	migrate -path=./migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path=./migrations -database "$(DB_URL)" down 1

migrate-status:
	migrate -path=./migrations -database "$(DB_URL)" version

migrate-create:
	@test -n "$(NAME)" || (echo "Usage: make migrate-create NAME=<migration_name>" && exit 1)
	migrate create -ext sql -dir migrations -seq $(NAME)

# ─── Docker ──────────────────────────────────────────────────────────────────

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f api worker

docker-ps:
	docker compose ps

# ─── Garage S3 setup (one-time after first docker-up) ────────────────────────

garage-init:
	@echo "Initialising Garage single-node cluster..."
	@NODE_ID=$$(docker compose exec garage /garage node id -q 2>/dev/null); \
	docker compose exec garage /garage layout assign -z dc1 -c 1G $$NODE_ID && \
	docker compose exec garage /garage layout apply --version 1 && \
	echo "Garage initialised – node $$NODE_ID"

garage-create-bucket:
	@echo "Creating key and bucket..."
	@KEY_INFO=$$(docker compose exec garage /garage key create rainlogs-key); \
	KEY_ID=$$(echo "$$KEY_INFO" | grep "Key ID" | awk '{print $$3}'); \
	SECRET=$$(echo "$$KEY_INFO" | grep "Secret key" | awk '{print $$3}'); \
	docker compose exec garage /garage bucket create rainlogs-logs && \
	docker compose exec garage /garage bucket allow rainlogs-logs --read --write --owner --key rainlogs-key && \
	echo "" && echo "S3_ACCESS_KEY_ID=$$KEY_ID" && echo "S3_SECRET_ACCESS_KEY=$$SECRET"

# ─── Maintenance ─────────────────────────────────────────────────────────────

tidy:
	$(GO) mod tidy

clean:
	rm -rf bin/ coverage.out coverage.html

# ─── Help ────────────────────────────────────────────────────────────────────

help:
	@echo "Available targets:"
	@echo "  build           Build API and Worker binaries"
	@echo "  test            Run unit tests with race detector"
	@echo "  test-integration Run integration tests (requires running infra)"
	@echo "  cover           Generate HTML coverage report"
	@echo "  lint            Run golangci-lint"
	@echo "  vet             Run go vet"
	@echo "  vuln            Run govulncheck"
	@echo "  fmt             Format code"
	@echo "  check           Full quality gate (vet + lint + vuln + test)"
	@echo "  migrate-up      Apply all pending migrations"
	@echo "  migrate-down    Roll back one migration"
	@echo "  migrate-create  Create new migration (NAME=<name>)"
	@echo "  docker-up       Start full stack with docker compose"
	@echo "  docker-down     Stop stack"
	@echo "  docker-logs     Tail API + Worker logs"
	@echo "  garage-init     Initialise Garage S3 cluster (one-time)"
	@echo "  clean           Remove build artefacts"
