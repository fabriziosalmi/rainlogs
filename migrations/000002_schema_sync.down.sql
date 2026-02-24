-- Migration: 000002_schema_sync.down.sql
-- Reverts 000002_schema_sync.up.sql

-- log_objects
ALTER TABLE log_objects RENAME COLUMN byte_count TO size_bytes;
ALTER TABLE log_objects RENAME COLUMN job_id TO log_job_id;
ALTER TABLE log_objects RENAME COLUMN s3_key TO storage_path;

-- log_jobs
ALTER TABLE log_jobs DROP CONSTRAINT IF EXISTS log_jobs_status_check;
ALTER TABLE log_jobs ADD CONSTRAINT log_jobs_status_check
    CHECK (status IN ('pending','running','completed','failed','retrying'));
ALTER TABLE log_jobs DROP COLUMN IF EXISTS updated_at;
ALTER TABLE log_jobs DROP COLUMN IF EXISTS s3_provider;
ALTER TABLE log_jobs RENAME COLUMN byte_count TO bytes_pulled;
ALTER TABLE log_jobs RENAME COLUMN s3_key TO storage_path;
ALTER TABLE log_jobs RENAME COLUMN period_start TO window_start;
ALTER TABLE log_jobs RENAME COLUMN period_end   TO window_end;
DROP INDEX IF EXISTS idx_log_jobs_customer;
ALTER TABLE log_jobs DROP COLUMN IF EXISTS customer_id;

-- zones
ALTER TABLE zones DROP CONSTRAINT IF EXISTS zones_pull_interval_secs_check;
ALTER TABLE zones ADD COLUMN pull_interval_mins INT NOT NULL DEFAULT 60;
UPDATE zones SET pull_interval_mins = pull_interval_secs / 60;
ALTER TABLE zones DROP COLUMN pull_interval_secs;
ALTER TABLE zones RENAME COLUMN zone_id TO cloudflare_zone_id;

-- api_keys
ALTER TABLE api_keys RENAME COLUMN label TO name;
