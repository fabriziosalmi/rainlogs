# RainLogs

> Cloudflare log archiving for NIS2-compliant European businesses.

RainLogs collects logs from Cloudflare zones via the **Logpull API** (available on Free, Pro, and Business plans) and stores them in **EU-sovereign object storage** (Garage S3-compatible, Hetzner, Contabo) with **WORM integrity guarantees** suitable for NIS2 / D.Lgs. 138/2024 incident forensics.

## Why

| Problem | RainLogs solution |
|---------|-------------------|
| Cloudflare retains Logpull data for **7 days only** | Pulls every hour, archives for **13+ months** (configurable) |
| Logpush (real-time export) requires **Enterprise** plan | Works with **Free / Pro / Business** via Logpull |
| Log tampering risk undermines forensic value | SHA-256 WORM chain + append-only hash linking |
| US Cloud Act risk for EU data | Storage exclusively on **EU-based** providers; no US entity in chain |
| NIS2 art. 21 – incident reporting within 24h | Structured NDJSON archive queryable by time window |

---

## Architecture

```
                      ┌──────────────────┐
                      │  Cloudflare API  │
                      │  (Logpull API)   │
                      └────────┬─────────┘
                               │ HTTPS (zones/*/logs/received)
                               ▼
                    ┌─────────────────────┐
                    │   rainlogs-worker   │◄── Redis (asynq)
                    │  (Go, zone scheduler│
                    │   + task processor) │
                    └──────────┬──────────┘
                               │ compress + SHA-256
                               ▼
              ┌─────────────────────────────────┐
              │  S3-compatible EU object store  │
              │  (Garage dev / Hetzner prod)    │
              │  Key: logs/<zone>/<Y/M/D/H>/uuid│
              └────────────────┬────────────────┘
                               │ metadata + hash chain
                               ▼
                    ┌─────────────────────┐
                    │     PostgreSQL      │
                    │  customers zones    │
                    │  log_jobs log_objects│
                    └──────────┬──────────┘
                               │ REST API
                               ▼
                    ┌─────────────────────┐
                    │  rainlogs-api       │◄── Bearer API Key
                    │  (Echo HTTP server) │
                    └─────────────────────┘
```

### Key components

| Component | Tech | Notes |
|-----------|------|-------|
| API server | Go + Echo v4 | REST, API-key auth (bcrypt + prefix lookup) |
| Worker | Go + asynq | Pulls CF logs, stores WORM objects, verifies integrity |
| Queue | Redis (asynq) | Reliable at-least-once delivery, retry with backoff |
| Database | PostgreSQL 16 | Customers, zones, log jobs, WORM chain hashes |
| Object store | Garage / S3-compatible | EU-sovereign, partitioned by zone/date/hour |
| Integrity | SHA-256 + hash chain | NIS2/forensic-grade tamper evidence |

### Engineering Standards

RainLogs is built with a focus on reliability, data integrity, and long-term maintainability, adhering to several engineering patterns:

- **Idempotency (S3 Keys)**: Log archiving jobs utilize deterministic S3 keys based on time windows and zone IDs. Retrying a failed job is safe and will not produce duplicate data artifacts in storage, ensuring consistency even after network partitions.
- **CQRS (Command Query Responsibility Segregation)**: The system architecture separates the `api` service (optimized for reads/queries) from the `worker` service (optimized for writes/processing), allowing independent scaling of ingestion and retrieval workloads.
- **Exponential Backoff with Jitter**: The worker (via `asynq`) automatically handles transient failures (e.g., Cloudflare API rate limits) with randomized exponential backoff, preventing thundering herd issues during authorized outages.
- **Hexagonal Architecture (Ports and Adapters)**: Core business logic is isolated from external concerns like the database (PostgreSQL), storage (S3/FS), and queue (Redis), allowing for easier testing and component swapping without modifying the domain logic.
- **Graceful Degradation (Storage)**: The storage layer implements multi-provider failover strategies. If the primary S3 provider is unreachable, the system attempts to upload to secondary configured providers to ensure data durability.
- **Infrastructure as Code (IaC)**: All infrastructure components, including the storage layer and application services, are defined in version-controlled Kubernetes manifests and Docker Compose configurations, eliminating configuration drift.
- **Dependency Injection**: Components are explicitly wired at application startup (`main.go`), promoting loose coupling and making unit testing with mocks straightforward.
- **WORM Storage Compliance**: Implements a cryptographic hash chain (SHA-256) linking each log archive to the previous one, creating a tamper-evident sequence suitable for NIS2 compliance and audit trails.
- **Graceful Shutdown**: Services intercept termination signals (`SIGTERM`) to drain connections and complete in-flight jobs, preventing data loss or corruption during rolling updates.

---

## Quick start (Production)

### Option 1: Docker Compose (Single Node)

Includes HTTPS (Traefik), Database, Queue, and S3 Storage (Garage).

```bash
# 1. Download installer
curl -fsSL https://raw.githubusercontent.com/fabriziosalmi/rainlogs/main/install.sh | bash

# 2. Access Dashboard
# https://your-ip (Traefik handles self-signed certs automatically)
```

### Option 2: Kubernetes (K3s / K8s)

Deploy scalable stack on any cluster.

```bash
kubectl apply -f k8s/
```

---

## Quick start (Development)

### Prerequisites

- Go ≥ 1.23
- Docker + Docker Compose v2
- `make`

### 1. Clone and configure

```bash
git clone https://github.com/fabriziosalmi/rainlogs.git
cd rainlogs
cp .env.example .env
# Edit .env – at minimum set a JWT_SECRET:
openssl rand -hex 32   # paste into RAINLOGS_JWT_SECRET
```

### 2. Start infrastructure

```bash
make docker-up
```

This starts PostgreSQL, Redis, Garage S3, and Asynqmon (queue UI at `http://localhost:8383`).

### 3. Initialise Garage (first run only)

```bash
make garage-init
make garage-create-bucket
# Copy the printed S3_ACCESS_KEY_ID and S3_SECRET_ACCESS_KEY into your .env
```

### 4. Run migrations

```bash
make migrate-up
```

### 5. Start the API server

```bash
make dev-api
# → listening on :8080
```

### 6. Start the worker

```bash
make dev-worker
# → zone scheduler runs every 5 minutes
```

---

## API

All authenticated endpoints require `Authorization: Bearer rl_<token>`.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/v1/customers` | Register a new customer |
| `GET` | `/v1/me` | Get your customer profile |
| `POST` | `/v1/api-keys` | Issue a new API key |
| `DELETE` | `/v1/api-keys/:key_id` | Revoke an API key |
| `POST` | `/v1/zones` | Add a Cloudflare zone |
| `GET` | `/v1/zones` | List your zones |
| `GET` | `/v1/zones/:zone_id` | Get zone details |
| `DELETE` | `/v1/zones/:zone_id` | Remove a zone |
| `GET` | `/v1/zones/:zone_id/jobs` | List log jobs (paginated) |
| `POST` | `/v1/zones/:zone_id/pull` | Trigger an immediate pull |
| `GET` | `/v1/jobs/:job_id/download` | Download log archive (NDJSON) |

---

## License

This project is licensed under the Apache License 2.0.
See the [LICENSE](LICENSE) file for details.
