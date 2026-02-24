################################################################################
# RainLogs – Makefile
################################################################################

BINARY_API    := bin/rainlogs-api
BINARY_WORKER := bin/rainlogs-worker
MODULE        := github.com/fabriziosalmi/rainlogs
GO            := go
GOFLAGS       := -ldflags="-s -w"

.PHONY: all build api worker test lint clean migrate-up migrate-down \
        docker-build docker-up docker-down dev

# ─── Build ──────────────────────────────────────────────────────────────────

all: build

build: api worker

api:
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -o $(BINARY_API) ./cmd/api

worker:
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -o $(BINARY_WORKER) ./cmd/worker

# ─── Run (local, requires .env loaded) ──────────────────────────────────────

dev-api: api
	@echo "→ Starting API (set env vars first or copy .env)"
	./$(BINARY_API)

dev-worker: worker
	@echo "→ Starting Worker"
	./$(BINARY_WORKER)

# ─── Test ────────────────────────────────────────────────────────────────────

test:
	$(GO) test -race -cover ./...

test-verbose:
	$(GO) test -race -v -cover ./...

# ─── Lint ────────────────────────────────────────────────────────────────────

lint:
	@command -v golangci-lint >/dev/null 2>&1 || \
		(echo "golangci-lint not installed – run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...

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

# ─── Docker ──────────────────────────────────────────────────────────────────

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f api worker

# ─── Garage S3 setup (one-time after first docker-up) ────────────────────────
# Run these commands after `make docker-up` to init a single-node Garage cluster.

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

# ─── Clean ───────────────────────────────────────────────────────────────────

clean:
	rm -rf bin/

# ─── tidy ────────────────────────────────────────────────────────────────────

tidy:
	$(GO) mod tidy
