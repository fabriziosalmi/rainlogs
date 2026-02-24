CREATE TYPE export_status AS ENUM ('pending', 'processing', 'completed', 'failed');

CREATE TABLE log_exports (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    s3_config_enc TEXT NOT NULL,
    filter_start TIMESTAMPTZ NOT NULL,
    filter_end TIMESTAMPTZ NOT NULL,
    status export_status NOT NULL DEFAULT 'pending',
    log_count BIGINT DEFAULT 0,
    byte_count BIGINT DEFAULT 0,
    error_msg TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_log_exports_customer ON log_exports(customer_id);
