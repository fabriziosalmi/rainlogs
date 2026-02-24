DROP INDEX IF EXISTS idx_log_jobs_zone_last;
DROP INDEX IF EXISTS idx_log_jobs_expiry;
ALTER TABLE log_jobs DROP COLUMN IF EXISTS verified_at;
