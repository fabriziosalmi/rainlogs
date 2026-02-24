package models

import (
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	JobStatusPending JobStatus = "pending"
	JobStatusRunning JobStatus = "running"
	JobStatusDone    JobStatus = "done"
	JobStatusFailed  JobStatus = "failed"
	JobStatusExpired JobStatus = "expired"
)

// Customer is a tenant.
type Customer struct {
	ID            uuid.UUID `db:"id"             json:"id"`
	Name          string    `db:"name"           json:"name"`
	Email         string    `db:"email"          json:"email"`
	CFAccountID   string    `db:"cf_account_id"  json:"cf_account_id"`
	CFAPIKeyEnc   string    `db:"cf_api_key_enc" json:"-"`
	RetentionDays int       `db:"retention_days" json:"retention_days"`
	CreatedAt     time.Time `db:"created_at"     json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"     json:"updated_at"`
}

// APIKey is a hashed bearer token for a customer.
type APIKey struct {
	ID         uuid.UUID  `db:"id"           json:"id"`
	CustomerID uuid.UUID  `db:"customer_id"  json:"customer_id"`
	Prefix     string     `db:"prefix"       json:"prefix"`
	KeyHash    string     `db:"key_hash"     json:"-"`
	Label      string     `db:"label"        json:"label"`
	CreatedAt  time.Time  `db:"created_at"   json:"created_at"`
	LastUsedAt *time.Time `db:"last_used_at" json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `db:"revoked_at"   json:"revoked_at,omitempty"`
}

// Zone is a Cloudflare zone registered for a customer.
type Zone struct {
	ID               uuid.UUID  `db:"id"                 json:"id"`
	CustomerID       uuid.UUID  `db:"customer_id"        json:"customer_id"`
	ZoneID           string     `db:"zone_id"            json:"zone_id"`
	Name             string     `db:"name"               json:"name"`
	PullIntervalSecs int        `db:"pull_interval_secs" json:"pull_interval_secs"`
	LastPulledAt     *time.Time `db:"last_pulled_at"     json:"last_pulled_at,omitempty"`
	Active           bool       `db:"active"             json:"active"`
	CreatedAt        time.Time  `db:"created_at"         json:"created_at"`
}

// LogJob tracks a single Logpull fetch window.
type LogJob struct {
	ID          uuid.UUID `db:"id"           json:"id"`
	ZoneID      uuid.UUID `db:"zone_id"      json:"zone_id"`
	CustomerID  uuid.UUID `db:"customer_id"  json:"customer_id"`
	PeriodStart time.Time `db:"period_start" json:"period_start"`
	PeriodEnd   time.Time `db:"period_end"   json:"period_end"`
	Status      JobStatus `db:"status"       json:"status"`
	S3Key       string    `db:"s3_key"       json:"s3_key,omitempty"`
	S3Provider  string    `db:"s3_provider"  json:"s3_provider,omitempty"`
	SHA256      string    `db:"sha256"       json:"sha256,omitempty"`
	ChainHash   string    `db:"chain_hash"   json:"chain_hash,omitempty"`
	ByteCount   int64     `db:"byte_count"   json:"byte_count"`
	LogCount    int64     `db:"log_count"    json:"log_count"`
	Attempts   int        `db:"attempts"    json:"attempts"`
	ErrMsg     string     `db:"err_msg"     json:"err_msg,omitempty"`
	VerifiedAt *time.Time `db:"verified_at" json:"verified_at,omitempty"`
	CreatedAt  time.Time  `db:"created_at"  json:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"  json:"updated_at"`
}

// LogObject represents a stored S3 object.
type LogObject struct {
	ID        uuid.UUID `db:"id"         json:"id"`
	JobID     uuid.UUID `db:"job_id"     json:"job_id"`
	S3Key     string    `db:"s3_key"     json:"s3_key"`
	SHA256    string    `db:"sha256"     json:"sha256"`
	ByteCount int64     `db:"byte_count" json:"byte_count"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// LogEntry is a parsed Cloudflare NDJSON log line.
type LogEntry struct {
	RayID     string    `json:"RayID"`
	ClientIP  string    `json:"ClientIP"`
	Timestamp time.Time `json:"EdgeStartTimestamp"`
	Method    string    `json:"ClientRequestMethod"`
	URI       string    `json:"ClientRequestURI"`
	Status    int       `json:"EdgeResponseStatus"`
	UserAgent string    `json:"ClientRequestUserAgent"`
	ZoneName  string    `json:"ZoneName,omitempty"`
}

// IncidentEvent is used for NIS2 incident report export.
type IncidentEvent struct {
	OccurredAt  time.Time `json:"occurred_at"`
	ZoneID      string    `json:"zone_id"`
	ZoneName    string    `json:"zone_name"`
	ClientIP    string    `json:"client_ip"`
	RayID       string    `json:"ray_id"`
	HTTPStatus  int       `json:"http_status"`
	Description string    `json:"description"`
}

// SearchFilter carries parameters for the log search API.
type SearchFilter struct {
	CustomerID uuid.UUID
	ZoneID     *uuid.UUID
	IP         string
	RayID      string
	From       time.Time
	To         time.Time
	StatusCode *int
	Limit      int
	Offset     int
}
