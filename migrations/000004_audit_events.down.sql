-- 000004_audit_events.down.sql
DROP INDEX IF EXISTS idx_audit_events_customer;
DROP TABLE IF EXISTS audit_events;
