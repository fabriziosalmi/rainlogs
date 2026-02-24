-- Migration: 000001_initial_schema.up.sql
-- Creates the core RainLogs schema.
--
-- Design notes:
--   - All IDs are UUID v4 (gen_random_uuid() via pgcrypto).
--   - Soft deletes via deleted_at for customers and zones.
--   - log_jobs uses hard deletes (retention enforced at S3 level).
--   - The chain_hash column on log_jobs implements the append-only WORM chain.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- customers
-- ============================================================
CREATE TABLE customers (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT        NOT NULL,
    email           TEXT        NOT NULL UNIQUE,
    cf_account_id   TEXT        NOT NULL,
    cf_api_key_enc  TEXT        NOT NULL,
    retention_days  INT         NOT NULL DEFAULT 30,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_customers_email ON customers (email) WHERE deleted_at IS NULL;

-- ============================================================
-- api_keys
-- ============================================================
CREATE TABLE api_keys (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id  UUID        NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    name         TEXT        NOT NULL,
    key_hash     TEXT        NOT NULL,       -- bcrypt hash of the full key
    prefix       TEXT        NOT NULL,       -- first 8 chars for fast lookup
    last_used_at TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at   TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_prefix     ON api_keys (prefix)      WHERE revoked_at IS NULL;
CREATE INDEX idx_api_keys_customer   ON api_keys (customer_id) WHERE revoked_at IS NULL;

-- ============================================================
-- zones
-- ============================================================
CREATE TABLE zones (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id          UUID        NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    cloudflare_zone_id   TEXT        NOT NULL,
    -- Cloudflare API token encrypted at rest (AES-256-GCM via application layer).
    cloudflare_api_key   TEXT        NOT NULL,
    name                 TEXT        NOT NULL,
    active               BOOLEAN     NOT NULL DEFAULT TRUE,
    pull_interval_mins   INT         NOT NULL DEFAULT 60
                                     CHECK (pull_interval_mins >= 5),
    -- High watermark: next pull starts from here.
    last_pulled_at       TIMESTAMPTZ,
    -- Comma-separated Cloudflare log fields.
    log_fields           TEXT        NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ,
    UNIQUE (customer_id, cloudflare_zone_id)
);

CREATE INDEX idx_zones_customer ON zones (customer_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_zones_due      ON zones (last_pulled_at NULLS FIRST)
    WHERE active = TRUE AND deleted_at IS NULL;

-- ============================================================
-- log_jobs
-- ============================================================
CREATE TABLE log_jobs (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    zone_id       UUID        NOT NULL REFERENCES zones (id) ON DELETE CASCADE,
    status        TEXT        NOT NULL DEFAULT 'pending'
                              CHECK (status IN ('pending','running','completed','failed','retrying')),
    window_start  TIMESTAMPTZ NOT NULL,
    window_end    TIMESTAMPTZ NOT NULL,
    -- Raw bytes received from Cloudflare (before compression).
    bytes_pulled  BIGINT      NOT NULL DEFAULT 0,
    -- Number of NDJSON log lines.
    log_count     BIGINT      NOT NULL DEFAULT 0,
    -- S3 object key where the compressed+WORM object resides.
    storage_path  TEXT        NOT NULL DEFAULT '',
    -- SHA-256 hex of the stored gzip object (WORM integrity anchor).
    sha256        TEXT        NOT NULL DEFAULT '',
    -- Append-only hash chain: SHA-256(prev_chain_hash || sha256 || window_start || window_end)
    chain_hash    TEXT        NOT NULL DEFAULT '',
    error_message TEXT        NOT NULL DEFAULT '',
    attempts      INT         NOT NULL DEFAULT 0,
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_log_jobs_zone_window ON log_jobs (zone_id, window_start DESC);
CREATE INDEX idx_log_jobs_status      ON log_jobs (status) WHERE status IN ('pending','running','retrying');

-- ============================================================
-- log_objects (catalogue of stored S3 objects)
-- ============================================================
CREATE TABLE log_objects (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    zone_id       UUID        NOT NULL REFERENCES zones (id) ON DELETE CASCADE,
    log_job_id    UUID        NOT NULL REFERENCES log_jobs (id) ON DELETE CASCADE,
    storage_path  TEXT        NOT NULL,
    -- S3 provider label (e.g. 'garage-eu', 'hetzner-fsn1').
    provider      TEXT        NOT NULL DEFAULT 'primary',
    sha256        TEXT        NOT NULL,
    size_bytes    BIGINT      NOT NULL DEFAULT 0,
    log_count     BIGINT      NOT NULL DEFAULT 0,
    window_start  TIMESTAMPTZ NOT NULL,
    window_end    TIMESTAMPTZ NOT NULL,
    -- NIS2 retention: expires_at = created_at + configured retention period.
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_log_objects_zone    ON log_objects (zone_id, window_start DESC);
CREATE INDEX idx_log_objects_expires ON log_objects (expires_at);
