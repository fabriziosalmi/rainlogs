-- 000004_audit_events.up.sql
-- GDPR Art. 30 / NIS2 Art. 21 – persistent audit trail

CREATE TABLE IF NOT EXISTS audit_events (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id  UUID        NULL REFERENCES customers(id) ON DELETE SET NULL,
    request_id   TEXT        NOT NULL,
    action       TEXT        NOT NULL,
    resource_id  TEXT        NULL,
    ip_address   TEXT        NOT NULL,
    status_code  INT         NOT NULL,
    error_detail TEXT        NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_events_customer
    ON audit_events(customer_id, created_at DESC);
