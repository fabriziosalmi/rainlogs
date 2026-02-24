# API Reference

RainLogs exposes a REST API on port `8080` (configurable via `RAINLOGS_APP_PORT`).

All authenticated endpoints require a Bearer API key issued via `POST /api/v1/api-keys`.

## Authentication

```http
Authorization: Bearer rl_<your-api-key>
```

Keys are generated once and shown **only once**. Store them securely — only the bcrypt hash is saved server-side.

---

## Public Endpoints

### `GET /health`

Returns the health status of the service and its dependencies.

**Response `200 OK`**
```json
{
  "status": "ok",
  "version": "0.1.0",
  "deps": {
    "postgres": { "status": "ok" }
  }
}
```

**Response `503 Service Unavailable`** when a dependency is down:
```json
{
  "status": "degraded",
  "version": "0.1.0",
  "deps": {
    "postgres": { "status": "error", "error": "connection refused" }
  }
}
```

---

### `POST /customers`

Register a new customer (tenant). Each customer maps to one Cloudflare account.

**Request body**
```json
{
  "name": "Acme GmbH",
  "email": "ops@acme.de",
  "cf_account_id": "abc123",
  "cf_api_key": "v1.0-...",
  "retention_days": 395
}
```

| Field | Type | Description |
|---|---|---|
| `name` | string | Display name |
| `email` | string | Unique contact email |
| `cf_account_id` | string | Cloudflare Account ID |
| `cf_api_key` | string | Cloudflare API token (encrypted at rest with AES-256-GCM) |
| `retention_days` | int | Log retention period in days (NIS2 minimum: 395) |

**Response `201 Created`**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Acme GmbH",
  "email": "ops@acme.de",
  "cf_account_id": "abc123",
  "retention_days": 395,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

> ⚠️ Note: `cf_api_key` is never returned in responses.

---

### `GET /customers/:id`

Retrieve a customer by UUID.

**Response `200 OK`** — same shape as `POST /customers` response.

---

## Authenticated Endpoints (`/api/v1`)

All routes below require `Authorization: Bearer <api-key>`.

### Zones

#### `POST /api/v1/zones`

Register a Cloudflare zone for log archival.

**Request body**
```json
{
  "zone_id": "d41d8cd98f00b204e9800998ecf8427e",
  "name": "example.com",
  "plan": "enterprise",
  "pull_interval_secs": 300
}
```

| Field | Type | Constraints | Description |
|---|---|---|---|
| `zone_id` | string | required | Cloudflare Zone ID |
| `name` | string | required | Human-readable zone name |
| `plan` | string | optional | One of `enterprise` (default), `business`, `free_pro` |
| `pull_interval_secs` | int | min 300 | Pull frequency in seconds |

**Response `201 Created`**

#### `GET /api/v1/zones`

List all zones for the authenticated customer.

**Response `200 OK`**
```json
[
  {
    "id": "...",
    "customer_id": "...",
    "zone_id": "d41d8cd9...",
    "name": "example.com",
    "plan": "enterprise",
    "pull_interval_secs": 300,
    "last_pulled_at": "2024-01-15T10:25:00Z",
    "active": true,
    "created_at": "2024-01-10T08:00:00Z"
  }
]
```

#### `PATCH /api/v1/zones/:zone_id`

Update zone configuration. All fields are optional.

**Request body**
```json
{
  "name": "New Name",
  "plan": "business",
  "pull_interval_secs": 600,
  "active": true
}
```

**Response `200 OK`**

#### `DELETE /api/v1/zones/:zone_id`

Remove a zone and stop log archival. Existing log archives are preserved.

**Response `204 No Content`**

#### `POST /api/v1/zones/:zone_id/pull`

Trigger an immediate log pull outside the scheduled interval.

**Response `202 Accepted`**
```json
{
  "task_id": "asynq:job:abc123",
  "status": "pending"
}
```

---

### API Keys

#### `POST /api/v1/api-keys`

Generate a new API key for the authenticated customer.

**Request body**
```json
{ "label": "CI pipeline" }
```

**Response `201 Created`**
```json
{
  "id": "...",
  "label": "CI pipeline",
  "prefix": "aBcDeFgH",
  "created_at": "2024-01-15T10:30:00Z",
  "api_key": "rl_aBcDeFgH..."
}
```

> ⚠️ `api_key` is shown **exactly once**. Store it immediately — it cannot be recovered.

#### `GET /api/v1/api-keys`

List all API keys for the authenticated customer (plaintext never returned).

**Response `200 OK`**
```json
[
  {
    "id": "...",
    "customer_id": "...",
    "prefix": "aBcDeFgH",
    "label": "CI pipeline",
    "created_at": "2024-01-10T08:00:00Z",
    "last_used_at": "2024-01-15T09:45:00Z"
  }
]
```

#### `DELETE /api/v1/api-keys/:key_id`

Revoke an API key immediately. All subsequent requests using that key will be rejected.

**Response `204 No Content`**

---

### Log Jobs

#### `GET /api/v1/logs/jobs`

List log archival jobs for the authenticated customer.

**Query parameters**

| Parameter | Default | Description |
|---|---|---|
| `limit` | 50 | Max results (max 500) |
| `offset` | 0 | Pagination offset |

**Response `200 OK`**
```json
[
  {
    "id": "...",
    "zone_id": "...",
    "customer_id": "...",
    "period_start": "2024-01-15T09:00:00Z",
    "period_end": "2024-01-15T09:05:00Z",
    "status": "done",
    "sha256": "abc123...",
    "chain_hash": "def456...",
    "byte_count": 102400,
    "log_count": 1523,
    "attempts": 1,
    "created_at": "2024-01-15T09:06:00Z",
    "updated_at": "2024-01-15T09:06:15Z"
  }
]
```

Job `status` values:

| Status | Meaning |
|---|---|
| `pending` | Queued, not yet started |
| `running` | Currently pulling from Cloudflare |
| `done` | Successfully archived |
| `failed` | Permanently failed |
| `expired` | Archived data deleted per retention policy (GDPR art.17) |

#### `GET /api/v1/logs/jobs/:job_id`

Get a single log job by ID.

**Response `200 OK`** — same shape as list item above.

#### `GET /api/v1/logs/jobs/:job_id/download`

Download the raw NDJSON log archive for a completed job.

**Response `200 OK`**
- `Content-Type: application/x-ndjson`
- `Content-Disposition: attachment; filename="rainlogs_20240115T090000Z_20240115T090500Z.ndjson"`
- `X-SHA256: <hex>` — SHA-256 of the returned bytes for client-side integrity verification
- `X-Chain-Hash: <hex>` — WORM chain hash for tamper evidence

---

## Error Responses

All errors return a consistent JSON envelope:

```json
{
  "code": 400,
  "message": "missing required fields",
  "request_id": "550e8400-e29b-41d4-a716-446655440001"
}
```

Include the `request_id` in bug reports or support tickets.

| HTTP Status | Meaning |
|---|---|
| `400` | Bad request / validation error |
| `401` | Missing or invalid API key |
| `403` | Access denied (wrong tenant) |
| `404` | Resource not found |
| `429` | Rate limit exceeded (60 req/s per IP) |
| `500` | Internal server error |
| `503` | Service unavailable (dependency down) |

---

## Rate Limiting

The API enforces **60 requests/second per IP** with a burst of 120. Exceeding the limit returns `429 Too Many Requests`.

---

## WORM Integrity Verification

Every completed job includes `sha256` and `chain_hash` fields.

To verify a downloaded archive:

```bash
# Verify object integrity
sha256sum rainlogs_*.ndjson | awk '{print $1}'
# Must match the X-SHA256 response header.

# Verify the chain hash
# chain_hash = SHA256(prev_chain_hash || sha256 || job_id)
echo -n "${prev_chain_hash}${sha256}${job_id}" | sha256sum
```

The genesis hash (first job in a zone's chain) is:
```
0000000000000000000000000000000000000000000000000000000000000000
```
