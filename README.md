# RainLogs

> Cloudflare log archiving for NIS2-compliant European businesses.

RainLogs collects logs from Cloudflare zones via the **Logpull API** (available on Free, Pro, and Business plans) and stores them in **EU-sovereign object storage** (Garage S3-compatible, Hetzner, Contabo) with **WORM integrity guarantees** suitable for NIS2 / D.Lgs. 138/2024 incident forensics.

[![CI](https://github.com/fabriziosalmi/rainlogs/actions/workflows/ci.yml/badge.svg)](https://github.com/fabriziosalmi/rainlogs/actions/workflows/ci.yml)
[![Go 1.24](https://img.shields.io/badge/go-1.24-blue)](https://go.dev)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

## Why

| Problem | RainLogs solution |
|---------|-------------------|
| Cloudflare retains Logpull data for **7 days only** | Pulls every 5 min, archives for **13+ months** (configurable) |
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
                    │  (zone scheduler   │
                    │   + task processor) │
                    └──────────┬──────────┘
                               │ compress + SHA-256 + WORM chain
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
                    │  rainlogs-api       │◄── Bearer API Key / JWT
                    │  (Echo HTTP server) │    Rate-limited · WORM headers
                    └─────────────────────┘
```

### Key Components

| Component | Tech | Notes |
|-----------|------|-------|
| API server | Go 1.24 + Echo v4 | REST, API-key + JWT auth, rate limiting, security headers |
| Worker | Go 1.24 + asynq | Pulls CF logs, stores WORM objects, verifies integrity |
| Queue | Redis 7 (asynq) | Reliable at-least-once delivery, retry with exponential backoff |
| Database | PostgreSQL 16 | Customers, zones, log jobs, WORM chain hashes |
| Object store | Garage / S3-compatible | EU-sovereign, partitioned by zone/date/hour, multi-provider failover |
| Integrity | SHA-256 + WORM hash chain | NIS2/forensic-grade tamper evidence |

### Engineering Standards

- **Idempotency**: Deterministic S3 keys prevent duplicate artifacts on job retries.
- **CQRS**: API and Worker services scale independently — reads vs writes.
- **Exponential Backoff with Jitter**: `asynq` handles transient Cloudflare failures automatically.
- **Hexagonal Architecture**: Core logic decoupled from DB, storage, and queue; easy to unit test.
- **Graceful Degradation**: Multi-provider S3 failover — if primary is unreachable, secondary providers are tried in order.
- **Dependency Injection**: All components wired explicitly at startup; no global state.
- **WORM Chain**: `ChainHash = SHA256(prevHash ∥ objectSHA256 ∥ jobID)` — tamper-evident, forensic-grade.
- **Graceful Shutdown**: SIGTERM drains connections cleanly, preventing data loss during rolling updates.

---

## Quick Start (Production)

### Option 1: Docker Compose (Single Node)

Includes HTTPS (Traefik), PostgreSQL, Redis, Garage S3, and Asynqmon dashboard.

```bash
curl -fsSL https://raw.githubusercontent.com/fabriziosalmi/rainlogs/main/install.sh | bash
```

### Option 2: Kubernetes (K3s / K8s)

```bash
kubectl apply -f k8s/
```

---

## Quick Start (Development)

### Prerequisites

- Go ≥ 1.24
- Docker + Docker Compose v2
- `make`

### 1. Clone and configure

```bash
git clone https://github.com/fabriziosalmi/rainlogs.git
cd rainlogs
cp .env.example .env
# Set required secrets:
openssl rand -hex 32   # → RAINLOGS_JWT_SECRET
openssl rand -hex 32   # → RAINLOGS_KMS_KEY
```

### 2. Start infrastructure

```bash
make docker-up
# Queue UI: http://localhost:8383
```

### 3. Initialise Garage (first run only)

```bash
make garage-init
make garage-create-bucket
# Copy the printed keys into .env
```

### 4. Run migrations

```bash
make migrate-up
```

### 5. Start API and Worker

```bash
make dev-api     # → :8080
make dev-worker  # → zone scheduler every 5 min
```

---

## API

All authenticated endpoints require `Authorization: Bearer rl_<token>`.
See the [full API reference](docs/guide/api-reference.md) for request/response shapes.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | Public | Health + dependency status |
| `POST` | `/customers` | Public | Register a new customer |
| `GET` | `/api/v1/customers/:id` | API Key | Get own customer profile |
| `POST` | `/api/v1/api-keys` | API Key | Issue a new API key |
| `GET` | `/api/v1/api-keys` | API Key | List API keys |
| `DELETE` | `/api/v1/api-keys/:key_id` | API Key | Revoke an API key |
| `POST` | `/api/v1/zones` | API Key | Add a Cloudflare zone |
| `GET` | `/api/v1/zones` | API Key | List zones (includes `health` field) |
| `DELETE` | `/api/v1/zones/:zone_id` | API Key | Remove a zone |
| `POST` | `/api/v1/zones/:zone_id/pull` | API Key | Trigger immediate pull |
| `GET` | `/api/v1/zones/:zone_id/logs` | API Key | List log jobs for a zone (paginated) |
| `GET` | `/api/v1/logs/jobs` | API Key | List all log jobs (paginated) |
| `GET` | `/api/v1/logs/jobs/:job_id` | API Key | Get single job + WORM hashes |
| `GET` | `/api/v1/logs/jobs/:job_id/download` | API Key | Download NDJSON archive |

### Rate Limiting

60 requests/second per IP, burst of 120. Returns `429 Too Many Requests` with `Retry-After` and `X-RateLimit-*` headers when exceeded.

### Error Responses

```json
{ "code": 400, "message": "missing required fields", "request_id": "550e8400-..." }
```

---

## Roadmap

- [ ] Log search API (query by IP, ray ID, time range)
- [ ] OpenAPI / Swagger docs
- [ ] NIS2 incident report export (PDF summary of events in window)
- [x] Asynq monitoring integration (Asynqmon)

---

## Development

```bash
make test        # Unit tests with race detector
make check       # Full quality gate: vet + lint + vuln + test
make cover       # HTML coverage report
make lint        # golangci-lint
make vuln        # govulncheck (supply chain security)
make fmt         # Format code
make migrate-create NAME=add_something  # New migration
make help        # All available targets
```

---

## Compliance Notes

| Regulation | How RainLogs addresses it |
|---|---|
| **NIS2 art. 21** | 13-month log retention (configurable), tamper-evident WORM chain |
| **NIS2 art. 23** | Structured NDJSON archives queryable by time window for 24h incident reporting |
| **GDPR art. 17** | Retention-based automatic deletion of S3 objects + DB records |
| **GDPR art. 32** | AES-256-GCM encryption at rest for Cloudflare API keys, bcrypt for API keys |
| **EU data sovereignty** | Storage exclusively on EU-based providers (Garage, Hetzner, Contabo) |

---

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `API did not become healthy` | Postgres/Redis not ready | `docker compose logs postgres redis` |
| `429 Too Many Requests` | Rate limit hit | Wait 1 s (see `Retry-After` header) or reduce request frequency |
| Worker shows no jobs in Asynqmon | No zones registered | `POST /api/v1/zones` to add a zone |
| `cloudflare: rate limited` | CF Logpull quota exceeded | Worker retries automatically with exponential backoff |
| `job missing s3 key or hash` | Zone had zero logs | Expected — empty windows are skipped, no archive created |
| `verified_at` is null | Job not yet verified | Verify task runs after pull; check worker logs |
| Garage bucket missing | First-run init skipped | Run `make garage-init && make garage-create-bucket` |
| `dial tcp: connection refused` on Redis | Redis not started | `docker compose up -d redis` |

**Useful commands:**

```bash
docker compose logs -f api          # API structured JSON logs
docker compose logs -f worker       # Worker structured JSON logs
docker compose ps                   # Service health status
curl http://localhost:8080/health   # API + dependency health
curl http://localhost:8081/health/worker  # Worker + queue depth (internal port)
open http://localhost:8383          # Asynqmon queue UI
```

---

## License

Apache License 2.0 — see [LICENSE](LICENSE).
