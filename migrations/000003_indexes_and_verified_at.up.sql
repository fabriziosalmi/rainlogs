-- Add verified_at so operators can tell which jobs have been integrity-checked.
ALTER TABLE log_jobs ADD COLUMN IF NOT EXISTS verified_at TIMESTAMPTZ NULL;

-- Composite index for LogJobRepository.ListExpired – avoids full table scan.
CREATE INDEX IF NOT EXISTS idx_log_jobs_expiry
  ON log_jobs (customer_id, period_end DESC)
  WHERE status = 'done';

-- Deterministic ordering for GetLastJob – tiebreaker on id prevents
-- non-deterministic WORM chain linkage when two jobs share a created_at timestamp.
CREATE INDEX IF NOT EXISTS idx_log_jobs_zone_last
  ON log_jobs (zone_id, created_at DESC, id DESC)
  WHERE status = 'done';
