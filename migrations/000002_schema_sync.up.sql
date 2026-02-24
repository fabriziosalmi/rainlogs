-- Migration: 000002_schema_sync.up.sql
-- Aligns column names between the initial schema and the Go models.
-- Adds missing columns and useful operational fields.

-- ── api_keys ─────────────────────────────────────────────────────────────────
-- Rename `name` → `label` to match the APIKey model.
ALTER TABLE api_keys RENAME COLUMN name TO label;

-- ── zones ────────────────────────────────────────────────────────────────────
-- Rename `cloudflare_zone_id` → `zone_id` to match the Zone model.
ALTER TABLE zones RENAME COLUMN cloudflare_zone_id TO zone_id;

-- Convert pull_interval_mins → pull_interval_secs (1 min = 60 secs).
ALTER TABLE zones ADD COLUMN pull_interval_secs INT NOT NULL DEFAULT 300;
UPDATE zones SET pull_interval_secs = pull_interval_mins * 60;
ALTER TABLE zones DROP COLUMN pull_interval_mins;
ALTER TABLE zones ADD CONSTRAINT zones_pull_interval_secs_check
    CHECK (pull_interval_secs >= 300);

-- Drop the encrypted CF key from zones (it lives on customers now).
ALTER TABLE zones DROP COLUMN IF EXISTS cloudflare_api_key;
-- Drop log_fields (handled by the Cloudflare client defaults).
ALTER TABLE zones DROP COLUMN IF EXISTS log_fields;

-- ── log_jobs ─────────────────────────────────────────────────────────────────
-- Add customer_id for fast tenant-scoped queries.
ALTER TABLE log_jobs ADD COLUMN customer_id UUID REFERENCES customers (id) ON DELETE CASCADE;

-- Backfill customer_id from the zones table.
UPDATE log_jobs lj
SET customer_id = z.customer_id
FROM zones z WHERE z.id = lj.zone_id;

ALTER TABLE log_jobs ALTER COLUMN customer_id SET NOT NULL;
CREATE INDEX idx_log_jobs_customer ON log_jobs (customer_id, created_at DESC);

-- Rename window_start/end → period_start/end to match the LogJob model.
ALTER TABLE log_jobs RENAME COLUMN window_start TO period_start;
ALTER TABLE log_jobs RENAME COLUMN window_end   TO period_end;

-- Rename storage_path → s3_key.
ALTER TABLE log_jobs RENAME COLUMN storage_path TO s3_key;

-- Add s3_provider (winning provider label from MultiStore failover).
ALTER TABLE log_jobs ADD COLUMN s3_provider TEXT NOT NULL DEFAULT '';

-- Rename bytes_pulled → byte_count.
ALTER TABLE log_jobs RENAME COLUMN bytes_pulled TO byte_count;

-- Rename error_message → err_msg.
ALTER TABLE log_jobs RENAME COLUMN error_message TO err_msg;

-- Add updated_at for optimistic-lock style tracking.
ALTER TABLE log_jobs ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Rename attempts → attempt_count in the model (keep column, just ensure it exists).
-- The column already exists as `attempts`; expose it via the model alias below.
-- (No rename needed – we'll update the model to use `attempts`.)

-- Update status check to match Go model constants.
ALTER TABLE log_jobs DROP CONSTRAINT IF EXISTS log_jobs_status_check;
ALTER TABLE log_jobs ADD CONSTRAINT log_jobs_status_check
    CHECK (status IN ('pending','running','done','failed','expired'));

-- ── log_objects ───────────────────────────────────────────────────────────────
-- Rename storage_path → s3_key to match the LogObject model.
ALTER TABLE log_objects RENAME COLUMN storage_path TO s3_key;
-- Drop zone_id (redundant – accessible via log_job).
ALTER TABLE log_objects DROP COLUMN IF EXISTS zone_id;
-- Rename log_job_id → job_id.
ALTER TABLE log_objects RENAME COLUMN log_job_id TO job_id;
-- Drop window_start/end and expires_at (managed at the job level).
ALTER TABLE log_objects DROP COLUMN IF EXISTS window_start;
ALTER TABLE log_objects DROP COLUMN IF EXISTS window_end;
ALTER TABLE log_objects DROP COLUMN IF EXISTS expires_at;
-- Rename size_bytes → byte_count.
ALTER TABLE log_objects RENAME COLUMN size_bytes TO byte_count;
