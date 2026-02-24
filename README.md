# RainLogs

> Cloudflare log archiving for NIS2-compliant European businesses.

RainLogs collects logs from Cloudflare zones and stores them in **EU-sovereign object storage** (Garage S3-compatible, Hetzner, Contabo) with **WORM integrity guarantees** suitable for NIS2 / D.Lgs. 138/2024 incident forensics.

## Features

### 🛡️ Compliance & Security
- **WORM Storage**: Logs are hash-chained (SHA-256) to detect tampering.
- **Verification Tool**: Built-in CLI `rainlogs-verify` to audit the cryptographic chain integrity.
- **EU Sovereignty**: Compatible with S3-compliant EU storage providers (Garage, Hetzner).
- **GDPR**: Automated retention policies (e.g., 395 days).

### 🚀 Performance & Reliability
- **Smart Polling**: Adapts to Cloudflare plans (Logpull for Enterprise, Instant Logs for Business, GraphQL for Free/Pro).
- **Resilience**: Automatic retries, exponential backoff, and circuit breakers.
- **Storage Failover**: Support for primary and secondary S3 buckets for high availability.
- **Quotas**: Configurable storage limits per customer to prevent abuse.

### 🔔 Observability
- **Notifications**: Alerts on critical failures (e.g., security event storms, improved instant log reliability).
- **Health Checks**: Built-in endpoints for K8s/Docker probing.

## Supported Plans

RainLogs adapts its collection strategy based on your Cloudflare plan:

| Plan | Method | Data Type | Notes |
|---|---|---|---|
| **Enterprise** | **Logpull API** | Full Access Logs | Historical backfill supported (7 days). |
| **Business** | **Instant Logs** (WebSocket) | Full Access Logs | Real-time stream only. No historical backfill. |
| **Pro / Free** | **Security Poller** (GraphQL) | Security Events (WAF) | Logs blocked requests only. No legitimate traffic logs. |

[![CI](https://github.com/fabriziosalmi/rainlogs/actions/workflows/ci.yml/badge.svg)](https://github.com/fabriziosalmi/rainlogs/actions/workflows/ci.yml)
[![Go 1.24](https://img.shields.io/badge/go-1.24-blue)](https://go.dev)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

## Why

| Problem | RainLogs solution |
|---------|-------------------|
| Cloudflare retains Logpull data for **7 days only** | Pulls every 5 min, archives for **13+ months** (configurable) |
| Logpush (real-time export) requires **Enterprise** plan | Automates **Logpull** (Ent), **Instant Logs** (Biz), and **Security Events** (Free/Pro) |
| Log tampering risk undermines forensic value | SHA-256 WORM chain + append-only hash linking |
| US Cloud Act risk for EU data | Storage exclusively on **EU-based** providers; no US entity in chain |
| NIS2 art. 21 – incident reporting within 24h | Structured NDJSON archive queryable by time window |

---

## Architecture

```
                      ┌──────────────────────┐
                      │      Cloudflare      │
                      │ (Logpull/Instant/WAF)│
                      └──────────┬───────────┘
                                 │ HTTPS / WSS
                                 ▼
                      ┌─────────────────────┐
                      │   rainlogs-worker   │◄── Redis (asynq)
                      │  (zone scheduler    │
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
| API server | Go 1.24 + Echo v4 | REST, API-key + JWT auth, per-customer rate limiting, security headers, Prometheus metrics |
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

Deploy a production-ready stack with Ingress (nginx/Traefik), Autoscaling (HPA), and External Secrets (Vault, AWS, Azure).

```bash
# 1. Base deployment (ConfigMap, DB, Redis)
kubectl apply -f k8s/00-base.yaml
kubectl apply -f k8s/10-dependencies.yaml

# 2. Application (Migrations, API, Worker)
kubectl apply -f k8s/20-app.yaml

# 3. Networking (Ingress)
# Edit k8s/25-ingress.yaml to select nginx or Traefik
kubectl apply -f k8s/25-ingress.yaml

# 4. Scaling & Secrets (Optional)
kubectl apply -f k8s/30-hpa.yaml              # Horizontal Pod Autoscaler
kubectl apply -f k8s/35-external-secrets.yaml # External Secrets Operator
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
make dev-worker  # → zone scheduler runs every 1 min (configurable)
```

---

## API

All authenticated endpoints require `Authorization: Bearer rl_<token>`.
See the [full API reference](docs/guide/api-reference.md) for request/response shapes.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | Public | Health + dependency status |
| `GET` | `/metrics` | Public | Prometheus metrics (SRE) |
| `POST` | `/customers` | Public | Register a new customer |
| `GET` | `/api/v1/customers/:id` | API Key | Get own customer profile |
| `DELETE` | `/api/v1/customers/:id` | API Key | Erase account + all data (GDPR Art. 17) |
| `GET` | `/api/v1/export` | API Key | Export all data as JSON (GDPR Art. 20) |
| `GET` | `/api/v1/audit-log` | API Key | List own audit events (GDPR Art. 30) |
| `POST` | `/api/v1/api-keys` | API Key | Issue a new API key (optional `expires_in_days`) |
| `GET` | `/api/v1/api-keys` | API Key | List API keys |
| `DELETE` | `/api/v1/api-keys/:key_id` | API Key | Revoke an API key |
| `POST` | `/api/v1/zones` | API Key | Add a Cloudflare zone |
| `GET` | `/api/v1/zones` | API Key | List zones (includes `health` field) |
| `PATCH` | `/api/v1/zones/:zone_id` | API Key | Pause / resume / rename zone |
| `DELETE` | `/api/v1/zones/:zone_id` | API Key | Remove a zone (soft-delete) |
| `POST` | `/api/v1/zones/:zone_id/pull` | API Key | Trigger immediate pull |
| `GET` | `/api/v1/zones/:zone_id/logs` | API Key | List log jobs for a zone (paginated) |
| `GET` | `/api/v1/logs/jobs` | API Key | List all log jobs (paginated) |
| `GET` | `/api/v1/logs/jobs/:job_id` | API Key | Get single job + WORM hashes |
| `GET` | `/api/v1/logs/jobs/:job_id/download` | API Key | Download NDJSON archive |

All `/dashboard/*` routes mirror the above with JWT authentication instead of API keys.

### Rate Limiting

- **Global**: 60 req/s per IP, burst 120 (applied before auth)
- **Per-customer**: 30 req/s per authenticated customer, burst 60 (prevents tenant starvation)

Both layers return `429 Too Many Requests` with `Retry-After` and `X-RateLimit-*` headers.

### Error Responses

Machine-readable `error_code` field enables programmatic handling:

```json
{ "code": 409, "message": "email already registered", "error_code": "CUSTOMER_EMAIL_EXISTS", "request_id": "550e8400-..." }
```

Key error codes: `ZONE_NOT_FOUND`, `JOB_NOT_FOUND`, `ACCESS_DENIED`, `INVALID_REQUEST`, `API_KEY_EXPIRED`, `CUSTOMER_EMAIL_EXISTS`.

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
| **NIS2 art. 21** | 13-month log retention (configurable), tamper-evident WORM chain, persistent audit trail |
| **NIS2 art. 23** | Structured NDJSON archives queryable by time window for 24h incident reporting |
| **GDPR art. 17** | `DELETE /customers/:id` erases all S3 objects, zones, keys + soft-deletes account in one call |
| **GDPR art. 20** | `GET /export` returns a portable JSON snapshot of all customer data |
| **GDPR art. 30** | `audit_events` table + `GET /audit-log` — every mutating action recorded with IP, timestamp, result |
| **GDPR art. 32** | AES-256-GCM encryption at rest for Cloudflare API keys, bcrypt for API keys |
| **ISO 27001 A.9.4** | API key expiration (`expires_in_days`) with enforcement at auth time |
| **EU data sovereignty** | Storage exclusively on EU-based providers (Garage, Hetzner, Contabo) |
| **Supply chain** | SBOM (SPDX-JSON) generated and attached to every GitHub release via `anchore/sbom-action` |
| **Container security** | Trivy scans for CRITICAL/HIGH CVEs before every push; fails the build if found |
| **Network isolation** | Docker `backend` network (internal) + `frontend` network; DB/Redis never reachable from outside |

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
