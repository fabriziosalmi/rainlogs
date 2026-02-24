# Getting Started

## Prerequisites

- [Docker Engine]((https://docs.docker.com/get-docker/)) (v24+)
- Docker Compose (v2+)
- [Go 1.24+](https://go.dev/dl/) (optional, for source builds)

## Docker Compose Deployment

The provided `docker-compose.yml` orchestrates the complete service mesh:

1. **Clone Repository**

   ```bash
   git clone https://github.com/fabriziosalmi/rainlogs.git
   cd rainlogs
   ```

2. **Initialize Services**

   ```bash
   docker compose up -d
   ```

   **Services Started:**
   - `rainlogs-api`: Application server
   - `rainlogs-worker`: Background log processor
   - `garage`: S3-compatible object storage
   - `postgres`: Primary data store
   - `redis`: Job queue backend
   - `asynqmon`: Queue monitoring UI

3. **Status Check**

   Verify the API health endpoint:

   ```bash
   curl http://localhost:8080/health
   # Expected: {"status":"ok", ...}
   ```

## Automated Provisioning

RainLogs simplifies initial setup by automatically provisioning the necessary S3 resources. Upon startup:

1. The API connects to the Garage management interface.
2. Creates required buckets (default: `rainlogs-data`).
3. Generates IAM credentials and stores S3 access keys.
4. Updates application configuration dynamically.

No manual S3 configuration is required for the default Garage deployment.

## Next Steps

- Review the [Architecture](./architecture.md) documentation.
- Configure [External Storage](./configuration.md) providers.
