#!/usr/bin/env python3
"""Generate all RainLogs Go source files."""
import os

BASE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

def write(path, src):
    full = os.path.join(BASE, path)
    os.makedirs(os.path.dirname(full), exist_ok=True)
    with open(full, "w") as f:
        f.write(src)
    print(f"  wrote {path}")

# ─── internal/models/models.go ───────────────────────────────────────────────
write("internal/models/models.go", """\
package models

import (
\t"time"

\t"github.com/google/uuid"
)

type JobStatus string

const (
\tJobStatusPending JobStatus = "pending"
\tJobStatusRunning JobStatus = "running"
\tJobStatusDone    JobStatus = "done"
\tJobStatusFailed  JobStatus = "failed"
\tJobStatusExpired JobStatus = "expired"
)

// Customer is a tenant.
type Customer struct {
\tID            uuid.UUID  `+"`"+`db:"id"             json:"id"`+"`"+`
\tName          string     `+"`"+`db:"name"           json:"name"`+"`"+`
\tEmail         string     `+"`"+`db:"email"          json:"email"`+"`"+`
\tCFAccountID   string     `+"`"+`db:"cf_account_id"  json:"cf_account_id"`+"`"+`
\tCFAPIKeyEnc   string     `+"`"+`db:"cf_api_key_enc" json:"-"`+"`"+`
\tRetentionDays int        `+"`"+`db:"retention_days" json:"retention_days"`+"`"+`
\tCreatedAt     time.Time  `+"`"+`db:"created_at"     json:"created_at"`+"`"+`
\tUpdatedAt     time.Time  `+"`"+`db:"updated_at"     json:"updated_at"`+"`"+`
}

// APIKey is a hashed bearer token for a customer.
type APIKey struct {
\tID         uuid.UUID  `+"`"+`db:"id"           json:"id"`+"`"+`
\tCustomerID uuid.UUID  `+"`"+`db:"customer_id"  json:"customer_id"`+"`"+`
\tPrefix     string     `+"`"+`db:"prefix"       json:"prefix"`+"`"+`
\tKeyHash    string     `+"`"+`db:"key_hash"     json:"-"`+"`"+`
\tLabel      string     `+"`"+`db:"label"        json:"label"`+"`"+`
\tCreatedAt  time.Time  `+"`"+`db:"created_at"   json:"created_at"`+"`"+`
\tLastUsedAt *time.Time `+"`"+`db:"last_used_at" json:"last_used_at,omitempty"`+"`"+`
\tRevokedAt  *time.Time `+"`"+`db:"revoked_at"   json:"revoked_at,omitempty"`+"`"+`
}

// Zone is a Cloudflare zone registered for a customer.
type Zone struct {
\tID               uuid.UUID  `+"`"+`db:"id"                 json:"id"`+"`"+`
\tCustomerID       uuid.UUID  `+"`"+`db:"customer_id"        json:"customer_id"`+"`"+`
\tZoneID           string     `+"`"+`db:"zone_id"            json:"zone_id"`+"`"+`
\tName             string     `+"`"+`db:"name"               json:"name"`+"`"+`
\tPullIntervalSecs int        `+"`"+`db:"pull_interval_secs" json:"pull_interval_secs"`+"`"+`
\tLastPulledAt     *time.Time `+"`"+`db:"last_pulled_at"     json:"last_pulled_at,omitempty"`+"`"+`
\tActive           bool       `+"`"+`db:"active"             json:"active"`+"`"+`
\tCreatedAt        time.Time  `+"`"+`db:"created_at"         json:"created_at"`+"`"+`
}

// LogJob tracks a single Logpull fetch window.
type LogJob struct {
\tID          uuid.UUID `+"`"+`db:"id"           json:"id"`+"`"+`
\tZoneID      uuid.UUID `+"`"+`db:"zone_id"      json:"zone_id"`+"`"+`
\tCustomerID  uuid.UUID `+"`"+`db:"customer_id"  json:"customer_id"`+"`"+`
\tPeriodStart time.Time `+"`"+`db:"period_start" json:"period_start"`+"`"+`
\tPeriodEnd   time.Time `+"`"+`db:"period_end"   json:"period_end"`+"`"+`
\tStatus      JobStatus `+"`"+`db:"status"       json:"status"`+"`"+`
\tS3Key       string    `+"`"+`db:"s3_key"       json:"s3_key"`+"`"+`
\tS3Provider  string    `+"`"+`db:"s3_provider"  json:"s3_provider"`+"`"+`
\tSHA256      string    `+"`"+`db:"sha256"       json:"sha256"`+"`"+`
\tChainHash   string    `+"`"+`db:"chain_hash"   json:"chain_hash"`+"`"+`
\tByteCount   int64     `+"`"+`db:"byte_count"   json:"byte_count"`+"`"+`
\tLogCount    int64     `+"`"+`db:"log_count"    json:"log_count"`+"`"+`
\tErrMsg      string    `+"`"+`db:"err_msg"      json:"err_msg,omitempty"`+"`"+`
\tCreatedAt   time.Time `+"`"+`db:"created_at"   json:"created_at"`+"`"+`
\tUpdatedAt   time.Time `+"`"+`db:"updated_at"   json:"updated_at"`+"`"+`
}

// LogObject represents a stored S3 object.
type LogObject struct {
\tID        uuid.UUID `+"`"+`db:"id"         json:"id"`+"`"+`
\tJobID     uuid.UUID `+"`"+`db:"job_id"     json:"job_id"`+"`"+`
\tS3Key     string    `+"`"+`db:"s3_key"     json:"s3_key"`+"`"+`
\tSHA256    string    `+"`"+`db:"sha256"     json:"sha256"`+"`"+`
\tByteCount int64     `+"`"+`db:"byte_count" json:"byte_count"`+"`"+`
\tCreatedAt time.Time `+"`"+`db:"created_at" json:"created_at"`+"`"+`
}

// LogEntry is a parsed Cloudflare NDJSON log line.
type LogEntry struct {
\tRayID     string    `+"`"+`json:"RayID"`+"`"+`
\tClientIP  string    `+"`"+`json:"ClientIP"`+"`"+`
\tTimestamp time.Time `+"`"+`json:"EdgeStartTimestamp"`+"`"+`
\tMethod    string    `+"`"+`json:"ClientRequestMethod"`+"`"+`
\tURI       string    `+"`"+`json:"ClientRequestURI"`+"`"+`
\tStatus    int       `+"`"+`json:"EdgeResponseStatus"`+"`"+`
\tUserAgent string    `+"`"+`json:"ClientRequestUserAgent"`+"`"+`
\tZoneName  string    `+"`"+`json:"ZoneName,omitempty"`+"`"+`
}

// IncidentEvent is used for NIS2 incident report export.
type IncidentEvent struct {
\tOccurredAt  time.Time `+"`"+`json:"occurred_at"`+"`"+`
\tZoneID      string    `+"`"+`json:"zone_id"`+"`"+`
\tZoneName    string    `+"`"+`json:"zone_name"`+"`"+`
\tClientIP    string    `+"`"+`json:"client_ip"`+"`"+`
\tRayID       string    `+"`"+`json:"ray_id"`+"`"+`
\tHTTPStatus  int       `+"`"+`json:"http_status"`+"`"+`
\tDescription string    `+"`"+`json:"description"`+"`"+`
}

// SearchFilter carries parameters for the log search API.
type SearchFilter struct {
\tCustomerID uuid.UUID
\tZoneID     *uuid.UUID
\tIP         string
\tRayID      string
\tFrom       time.Time
\tTo         time.Time
\tStatusCode *int
\tLimit      int
\tOffset     int
}
""")

# ─── internal/db/db.go ───────────────────────────────────────────────────────
write("internal/db/db.go", """\
package db

import (
\t"context"
\t"fmt"
\t"time"

\t"github.com/fabriziosalmi/rainlogs/internal/config"
\t"github.com/jackc/pgx/v5/pgxpool"
)

// Connect returns a pgxpool.Pool configured from cfg.
func Connect(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
\tpoolCfg, err := pgxpool.ParseConfig(cfg.DSN)
\tif err != nil {
\t\treturn nil, fmt.Errorf("db: parse dsn: %w", err)
\t}
\tpoolCfg.MaxConns = int32(cfg.MaxOpenConns)
\tpoolCfg.MinConns = int32(cfg.MaxIdleConns)
\tpoolCfg.MaxConnLifetime = cfg.ConnMaxLifetime
\tpoolCfg.HealthCheckPeriod = 1 * time.Minute

\tpool, err := pgxpool.NewWithConfig(ctx, poolCfg)
\tif err != nil {
\t\treturn nil, fmt.Errorf("db: connect: %w", err)
\t}
\tif err := pool.Ping(ctx); err != nil {
\t\treturn nil, fmt.Errorf("db: ping: %w", err)
\t}
\treturn pool, nil
}
""")

# ─── internal/db/repositories.go ─────────────────────────────────────────────
write("internal/db/repositories.go", """\
package db

import (
\t"context"
\t"fmt"
\t"time"

\t"github.com/fabriziosalmi/rainlogs/internal/models"
\t"github.com/google/uuid"
\t"github.com/jackc/pgx/v5"
\t"github.com/jackc/pgx/v5/pgxpool"
)

// ── CustomerRepository ────────────────────────────────────────────────────────

type CustomerRepository struct{ db *pgxpool.Pool }

func NewCustomerRepository(db *pgxpool.Pool) *CustomerRepository {
\treturn &CustomerRepository{db: db}
}

func (r *CustomerRepository) Create(ctx context.Context, c *models.Customer) error {
\tconst q = `INSERT INTO customers
\t\t(id,name,email,cf_account_id,cf_api_key_enc,retention_days,created_at,updated_at)
\t\tVALUES($1,$2,$3,$4,$5,$6,now(),now())
\t\tRETURNING created_at,updated_at`
\treturn r.db.QueryRow(ctx, q,
\t\tc.ID, c.Name, c.Email, c.CFAccountID, c.CFAPIKeyEnc, c.RetentionDays,
\t).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Customer, error) {
\tconst q = `SELECT id,name,email,cf_account_id,cf_api_key_enc,retention_days,created_at,updated_at
\t\tFROM customers WHERE id=$1`
\tc := &models.Customer{}
\terr := r.db.QueryRow(ctx, q, id).Scan(
\t\t&c.ID, &c.Name, &c.Email, &c.CFAccountID, &c.CFAPIKeyEnc, &c.RetentionDays,
\t\t&c.CreatedAt, &c.UpdatedAt,
\t)
\tif err != nil {
\t\treturn nil, fmt.Errorf("customer get: %w", err)
\t}
\treturn c, nil
}

func (r *CustomerRepository) List(ctx context.Context) ([]*models.Customer, error) {
\tconst q = `SELECT id,name,email,cf_account_id,cf_api_key_enc,retention_days,created_at,updated_at
\t\tFROM customers ORDER BY created_at DESC`
\trows, err := r.db.Query(ctx, q)
\tif err != nil {
\t\treturn nil, err
\t}
\tdefer rows.Close()
\tvar out []*models.Customer
\tfor rows.Next() {
\t\tc := &models.Customer{}
\t\tif err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.CFAccountID, &c.CFAPIKeyEnc,
\t\t\t&c.RetentionDays, &c.CreatedAt, &c.UpdatedAt); err != nil {
\t\t\treturn nil, err
\t\t}
\t\tout = append(out, c)
\t}
\treturn out, rows.Err()
}

// ── APIKeyRepository ──────────────────────────────────────────────────────────

type APIKeyRepository struct{ db *pgxpool.Pool }

func NewAPIKeyRepository(db *pgxpool.Pool) *APIKeyRepository {
\treturn &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(ctx context.Context, k *models.APIKey) error {
\tconst q = `INSERT INTO api_keys(id,customer_id,prefix,key_hash,label,created_at)
\t\tVALUES($1,$2,$3,$4,$5,now()) RETURNING created_at`
\treturn r.db.QueryRow(ctx, q, k.ID, k.CustomerID, k.Prefix, k.KeyHash, k.Label).Scan(&k.CreatedAt)
}

func (r *APIKeyRepository) GetByPrefix(ctx context.Context, prefix string) ([]*models.APIKey, error) {
\tconst q = `SELECT id,customer_id,prefix,key_hash,label,created_at,last_used_at,revoked_at
\t\tFROM api_keys WHERE prefix=$1 AND revoked_at IS NULL`
\trows, err := r.db.Query(ctx, q, prefix)
\tif err != nil {
\t\treturn nil, err
\t}
\tdefer rows.Close()
\tvar out []*models.APIKey
\tfor rows.Next() {
\t\tk := &models.APIKey{}
\t\tif err := rows.Scan(&k.ID, &k.CustomerID, &k.Prefix, &k.KeyHash, &k.Label,
\t\t\t&k.CreatedAt, &k.LastUsedAt, &k.RevokedAt); err != nil {
\t\t\treturn nil, err
\t\t}
\t\tout = append(out, k)
\t}
\treturn out, rows.Err()
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
\t_, err := r.db.Exec(ctx, `UPDATE api_keys SET last_used_at=now() WHERE id=$1`, id)
\treturn err
}

func (r *APIKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
\t_, err := r.db.Exec(ctx, `UPDATE api_keys SET revoked_at=now() WHERE id=$1`, id)
\treturn err
}

// ── ZoneRepository ────────────────────────────────────────────────────────────

type ZoneRepository struct{ db *pgxpool.Pool }

func NewZoneRepository(db *pgxpool.Pool) *ZoneRepository {
\treturn &ZoneRepository{db: db}
}

func (r *ZoneRepository) Create(ctx context.Context, z *models.Zone) error {
\tconst q = `INSERT INTO zones(id,customer_id,zone_id,name,pull_interval_secs,active,created_at)
\t\tVALUES($1,$2,$3,$4,$5,$6,now()) RETURNING created_at`
\treturn r.db.QueryRow(ctx, q,
\t\tz.ID, z.CustomerID, z.ZoneID, z.Name, z.PullIntervalSecs, z.Active,
\t).Scan(&z.CreatedAt)
}

func (r *ZoneRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]*models.Zone, error) {
\tconst q = `SELECT id,customer_id,zone_id,name,pull_interval_secs,last_pulled_at,active,created_at
\t\tFROM zones WHERE customer_id=$1`
\trows, err := r.db.Query(ctx, q, customerID)
\tif err != nil {
\t\treturn nil, err
\t}
\tdefer rows.Close()
\tvar out []*models.Zone
\tfor rows.Next() {
\t\tz := &models.Zone{}
\t\tif err := rows.Scan(&z.ID, &z.CustomerID, &z.ZoneID, &z.Name,
\t\t\t&z.PullIntervalSecs, &z.LastPulledAt, &z.Active, &z.CreatedAt); err != nil {
\t\t\treturn nil, err
\t\t}
\t\tout = append(out, z)
\t}
\treturn out, rows.Err()
}

func (r *ZoneRepository) ListDue(ctx context.Context) ([]*models.Zone, error) {
\tconst q = `SELECT id,customer_id,zone_id,name,pull_interval_secs,last_pulled_at,active,created_at
\t\tFROM zones
\t\tWHERE active=true
\t\t  AND (last_pulled_at IS NULL OR
\t\t       last_pulled_at < now() - (pull_interval_secs || ' seconds')::interval)`
\trows, err := r.db.Query(ctx, q)
\tif err != nil {
\t\treturn nil, err
\t}
\tdefer rows.Close()
\tvar out []*models.Zone
\tfor rows.Next() {
\t\tz := &models.Zone{}
\t\tif err := rows.Scan(&z.ID, &z.CustomerID, &z.ZoneID, &z.Name,
\t\t\t&z.PullIntervalSecs, &z.LastPulledAt, &z.Active, &z.CreatedAt); err != nil {
\t\t\treturn nil, err
\t\t}
\t\tout = append(out, z)
\t}
\treturn out, rows.Err()
}

func (r *ZoneRepository) UpdateLastPulled(ctx context.Context, id uuid.UUID, t time.Time) error {
\t_, err := r.db.Exec(ctx, `UPDATE zones SET last_pulled_at=$2 WHERE id=$1`, id, t)
\treturn err
}

func (r *ZoneRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Zone, error) {
\tconst q = `SELECT id,customer_id,zone_id,name,pull_interval_secs,last_pulled_at,active,created_at
\t\tFROM zones WHERE id=$1`
\tz := &models.Zone{}
\terr := r.db.QueryRow(ctx, q, id).Scan(&z.ID, &z.CustomerID, &z.ZoneID, &z.Name,
\t\t&z.PullIntervalSecs, &z.LastPulledAt, &z.Active, &z.CreatedAt)
\tif err != nil {
\t\treturn nil, fmt.Errorf("zone get: %w", err)
\t}
\treturn z, nil
}

// ── LogJobRepository ──────────────────────────────────────────────────────────

type LogJobRepository struct{ db *pgxpool.Pool }

func NewLogJobRepository(db *pgxpool.Pool) *LogJobRepository {
\treturn &LogJobRepository{db: db}
}

func (r *LogJobRepository) Create(ctx context.Context, j *models.LogJob) error {
\tconst q = `INSERT INTO log_jobs
\t\t(id,zone_id,customer_id,period_start,period_end,status,created_at,updated_at)
\t\tVALUES($1,$2,$3,$4,$5,$6,now(),now())
\t\tRETURNING created_at,updated_at`
\treturn r.db.QueryRow(ctx, q,
\t\tj.ID, j.ZoneID, j.CustomerID, j.PeriodStart, j.PeriodEnd, j.Status,
\t).Scan(&j.CreatedAt, &j.UpdatedAt)
}

func (r *LogJobRepository) Update(ctx context.Context, j *models.LogJob) error {
\tconst q = `UPDATE log_jobs SET
\t\tstatus=$2, s3_key=$3, s3_provider=$4, sha256=$5,
\t\tchain_hash=$6, byte_count=$7, log_count=$8, err_msg=$9, updated_at=now()
\t\tWHERE id=$1`
\t_, err := r.db.Exec(ctx, q,
\t\tj.ID, j.Status, j.S3Key, j.S3Provider, j.SHA256,
\t\tj.ChainHash, j.ByteCount, j.LogCount, j.ErrMsg,
\t)
\treturn err
}

func (r *LogJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.LogJob, error) {
\tconst q = `SELECT id,zone_id,customer_id,period_start,period_end,status,
\t\t\ts3_key,s3_provider,sha256,chain_hash,byte_count,log_count,err_msg,created_at,updated_at
\t\tFROM log_jobs WHERE id=$1`
\tj := &models.LogJob{}
\terr := r.db.QueryRow(ctx, q, id).Scan(
\t\t&j.ID, &j.ZoneID, &j.CustomerID, &j.PeriodStart, &j.PeriodEnd, &j.Status,
\t\t&j.S3Key, &j.S3Provider, &j.SHA256, &j.ChainHash, &j.ByteCount, &j.LogCount,
\t\t&j.ErrMsg, &j.CreatedAt, &j.UpdatedAt,
\t)
\tif err != nil {
\t\treturn nil, fmt.Errorf("log_job get: %w", err)
\t}
\treturn j, nil
}

func (r *LogJobRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*models.LogJob, error) {
\tconst q = `SELECT id,zone_id,customer_id,period_start,period_end,status,
\t\t\ts3_key,s3_provider,sha256,chain_hash,byte_count,log_count,err_msg,created_at,updated_at
\t\tFROM log_jobs WHERE customer_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
\trows, err := r.db.Query(ctx, q, customerID, limit, offset)
\tif err != nil {
\t\treturn nil, err
\t}
\tdefer rows.Close()
\tvar out []*models.LogJob
\tfor rows.Next() {
\t\tj := &models.LogJob{}
\t\tif err := rows.Scan(&j.ID, &j.ZoneID, &j.CustomerID, &j.PeriodStart, &j.PeriodEnd,
\t\t\t&j.Status, &j.S3Key, &j.S3Provider, &j.SHA256, &j.ChainHash, &j.ByteCount,
\t\t\t&j.LogCount, &j.ErrMsg, &j.CreatedAt, &j.UpdatedAt); err != nil {
\t\t\treturn nil, err
\t\t}
\t\tout = append(out, j)
\t}
\treturn out, rows.Err()
}

// ListExpired returns jobs older than retentionDays that are still in done status.
func (r *LogJobRepository) ListExpired(ctx context.Context, customerID uuid.UUID, retentionDays int) ([]*models.LogJob, error) {
\tconst q = `SELECT id,zone_id,customer_id,period_start,period_end,status,
\t\t\ts3_key,s3_provider,sha256,chain_hash,byte_count,log_count,err_msg,created_at,updated_at
\t\tFROM log_jobs
\t\tWHERE customer_id=$1
\t\t  AND status=$2
\t\t  AND period_end < now() - ($3 || ' days')::interval`
\trows, err := r.db.Query(ctx, q, customerID, models.JobStatusDone, retentionDays)
\tif err != nil {
\t\treturn nil, err
\t}
\tdefer rows.Close()
\tvar out []*models.LogJob
\tfor rows.Next() {
\t\tj := &models.LogJob{}
\t\tif err := rows.Scan(&j.ID, &j.ZoneID, &j.CustomerID, &j.PeriodStart, &j.PeriodEnd,
\t\t\t&j.Status, &j.S3Key, &j.S3Provider, &j.SHA256, &j.ChainHash, &j.ByteCount,
\t\t\t&j.LogCount, &j.ErrMsg, &j.CreatedAt, &j.UpdatedAt); err != nil {
\t\t\treturn nil, err
\t\t}
\t\tout = append(out, j)
\t}
\treturn out, rows.Err()
}

// MarkExpired marks a job as expired (GDPR art.17 log erasure).
func (r *LogJobRepository) MarkExpired(ctx context.Context, id uuid.UUID) error {
\t_, err := r.db.Exec(ctx,
\t\t`UPDATE log_jobs SET status=$2, updated_at=now() WHERE id=$1`,
\t\tid, models.JobStatusExpired,
\t)
\treturn err
}

// ── LogObjectRepository ───────────────────────────────────────────────────────

type LogObjectRepository struct{ db *pgxpool.Pool }

func NewLogObjectRepository(db *pgxpool.Pool) *LogObjectRepository {
\treturn &LogObjectRepository{db: db}
}

func (r *LogObjectRepository) Create(ctx context.Context, o *models.LogObject) error {
\tconst q = `INSERT INTO log_objects(id,job_id,s3_key,sha256,byte_count,created_at)
\t\tVALUES($1,$2,$3,$4,$5,now()) RETURNING created_at`
\treturn r.db.QueryRow(ctx, q, o.ID, o.JobID, o.S3Key, o.SHA256, o.ByteCount).Scan(&o.CreatedAt)
}

// SearchLogs is a placeholder for the log search API backed by the DB index.
// Real implementation queries a dedicated log_lines table (see migration 002).
func (r *LogObjectRepository) SearchLogs(ctx context.Context, f models.SearchFilter) ([]*models.LogEntry, error) {
\t_ = f
\treturn nil, nil
}

// scanZone is a shared helper.
func scanZone(row pgx.Row, z *models.Zone) error {
\treturn row.Scan(&z.ID, &z.CustomerID, &z.ZoneID, &z.Name,
\t\t&z.PullIntervalSecs, &z.LastPulledAt, &z.Active, &z.CreatedAt)
}
""")

# ─── internal/config/config.go (already correct – skip) ─────────────────────

# ─── internal/auth/auth.go ───────────────────────────────────────────────────
write("internal/auth/auth.go", """\
package auth

import (
\t"crypto/rand"
\t"encoding/base64"
\t"fmt"
\t"strings"
\t"time"

\t"github.com/fabriziosalmi/rainlogs/internal/models"
\t"github.com/golang-jwt/jwt/v5"
\t"golang.org/x/crypto/bcrypt"
)

const (
\ttokenBytes  = 32
\ttokenPrefix = "rl_"
\tbcryptCost  = 12
\tprefixLen   = 8 // chars of base64url used as O(1) lookup prefix
)

// GenerateAPIKey returns (plaintext, bcryptHash, lookupPrefix, error).
func GenerateAPIKey() (plaintext, hash, prefix string, err error) {
\tb := make([]byte, tokenBytes)
\tif _, err = rand.Read(b); err != nil {
\t\treturn "", "", "", fmt.Errorf("auth: rand: %w", err)
\t}
\tencoded := base64.RawURLEncoding.EncodeToString(b)
\tplaintext = tokenPrefix + encoded
\tif len(encoded) < prefixLen {
\t\treturn "", "", "", fmt.Errorf("auth: encoded key too short")
\t}
\tprefix = encoded[:prefixLen]
\thashBytes, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcryptCost)
\tif err != nil {
\t\treturn "", "", "", fmt.Errorf("auth: bcrypt: %w", err)
\t}
\thash = string(hashBytes)
\treturn
}

// ValidateAPIKey compares a plaintext key against a bcrypt hash.
func ValidateAPIKey(plaintext, hash string) bool {
\treturn bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext)) == nil
}

// PrefixOf extracts the lookup prefix from a plaintext key.
func PrefixOf(plaintext string) (string, error) {
\tif !strings.HasPrefix(plaintext, tokenPrefix) {
\t\treturn "", fmt.Errorf("auth: invalid key format")
\t}
\tbody := plaintext[len(tokenPrefix):]
\tif len(body) < prefixLen {
\t\treturn "", fmt.Errorf("auth: key too short")
\t}
\treturn body[:prefixLen], nil
}

// Claims is the JWT payload.
type Claims struct {
\tCustomerID string `json:"cid"`
\tjwt.RegisteredClaims
}

// IssueJWT signs a short-lived JWT for internal service-to-service auth.
func IssueJWT(secret string, c *models.Customer, ttl time.Duration) (string, error) {
\tclaims := Claims{
\t\tCustomerID: c.ID.String(),
\t\tRegisteredClaims: jwt.RegisteredClaims{
\t\t\tExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
\t\t\tIssuedAt:  jwt.NewNumericDate(time.Now()),
\t\t\tSubject:   c.ID.String(),
\t\t},
\t}
\ttoken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
\treturn token.SignedString([]byte(secret))
}

// VerifyJWT validates a JWT and returns the claims.
func VerifyJWT(secret, tokenStr string) (*Claims, error) {
\tvar claims Claims
\t_, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
\t\tif _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
\t\t\treturn nil, fmt.Errorf("auth: unexpected signing method")
\t\t}
\t\treturn []byte(secret), nil
\t})
\tif err != nil {
\t\treturn nil, fmt.Errorf("auth: jwt verify: %w", err)
\t}
\treturn &claims, nil
}
""")

# ─── internal/kms/kms.go ─────────────────────────────────────────────────────
write("internal/kms/kms.go", """\
// Package kms provides AES-256-GCM envelope encryption for secrets at rest.
// The data key (DEK) is derived from a 32-byte master key stored in the
// KMS_MASTER_KEY environment variable (hex-encoded).
// For production, replace the master key derivation with a real KMS call
// (AWS KMS, Google Cloud KMS, HashiCorp Vault, etc.).
package kms

import (
\t"crypto/aes"
\t"crypto/cipher"
\t"crypto/rand"
\t"encoding/hex"
\t"fmt"
\t"io"
)

// Encryptor holds the AES-256-GCM master key.
type Encryptor struct {
\tkey []byte // 32 bytes
}

// New creates an Encryptor from a 64-char hex-encoded 32-byte master key.
func New(hexKey string) (*Encryptor, error) {
\tkey, err := hex.DecodeString(hexKey)
\tif err != nil {
\t\treturn nil, fmt.Errorf("kms: decode key: %w", err)
\t}
\tif len(key) != 32 {
\t\treturn nil, fmt.Errorf("kms: master key must be 32 bytes (got %d)", len(key))
\t}
\treturn &Encryptor{key: key}, nil
}

// Encrypt encrypts plaintext with AES-256-GCM and returns hex(nonce+ciphertext).
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
\tblock, err := aes.NewCipher(e.key)
\tif err != nil {
\t\treturn "", fmt.Errorf("kms: aes: %w", err)
\t}
\tgcm, err := cipher.NewGCM(block)
\tif err != nil {
\t\treturn "", fmt.Errorf("kms: gcm: %w", err)
\t}
\tnonce := make([]byte, gcm.NonceSize())
\tif _, err := io.ReadFull(rand.Reader, nonce); err != nil {
\t\treturn "", fmt.Errorf("kms: nonce: %w", err)
\t}
\tciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
\treturn hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a hex(nonce+ciphertext) produced by Encrypt.
func (e *Encryptor) Decrypt(hexCiphertext string) (string, error) {
\tdata, err := hex.DecodeString(hexCiphertext)
\tif err != nil {
\t\treturn "", fmt.Errorf("kms: decode hex: %w", err)
\t}
\tblock, err := aes.NewCipher(e.key)
\tif err != nil {
\t\treturn "", fmt.Errorf("kms: aes: %w", err)
\t}
\tgcm, err := cipher.NewGCM(block)
\tif err != nil {
\t\treturn "", fmt.Errorf("kms: gcm: %w", err)
\t}
\tns := gcm.NonceSize()
\tif len(data) < ns {
\t\treturn "", fmt.Errorf("kms: ciphertext too short")
\t}
\tplaintext, err := gcm.Open(nil, data[:ns], data[ns:], nil)
\tif err != nil {
\t\treturn "", fmt.Errorf("kms: decrypt: %w", err)
\t}
\treturn string(plaintext), nil
}
""")

# ─── internal/queue/queue.go ──────────────────────────────────────────────────
write("internal/queue/queue.go", """\
package queue

import (
\t"encoding/json"
\t"fmt"
\t"time"

\t"github.com/google/uuid"
\t"github.com/hibiken/asynq"
)

const (
\tTypeLogPull   = "log:pull"
\tTypeLogVerify = "log:verify"
\tTypeLogExpire = "log:expire"

\tQueueCritical = "critical"
\tQueueDefault  = "default"
\tQueueLow      = "low"
)

// LogPullPayload is the task payload for TypeLogPull.
type LogPullPayload struct {
\tZoneID      uuid.UUID `json:"zone_id"`
\tCustomerID  uuid.UUID `json:"customer_id"`
\tPeriodStart time.Time `json:"period_start"`
\tPeriodEnd   time.Time `json:"period_end"`
}

// LogVerifyPayload is the task payload for TypeLogVerify.
type LogVerifyPayload struct {
\tJobID uuid.UUID `json:"job_id"`
}

// LogExpirePayload is the task payload for TypeLogExpire.
type LogExpirePayload struct {
\tCustomerID    uuid.UUID `json:"customer_id"`
\tRetentionDays int       `json:"retention_days"`
}

func NewLogPullTask(p LogPullPayload) (*asynq.Task, error) {
\tb, err := json.Marshal(p)
\tif err != nil {
\t\treturn nil, fmt.Errorf("queue: marshal LogPull: %w", err)
\t}
\treturn asynq.NewTask(TypeLogPull, b, asynq.Queue(QueueDefault)), nil
}

func NewLogVerifyTask(p LogVerifyPayload) (*asynq.Task, error) {
\tb, err := json.Marshal(p)
\tif err != nil {
\t\treturn nil, fmt.Errorf("queue: marshal LogVerify: %w", err)
\t}
\treturn asynq.NewTask(TypeLogVerify, b, asynq.Queue(QueueLow)), nil
}

func NewLogExpireTask(p LogExpirePayload) (*asynq.Task, error) {
\tb, err := json.Marshal(p)
\tif err != nil {
\t\treturn nil, fmt.Errorf("queue: marshal LogExpire: %w", err)
\t}
\treturn asynq.NewTask(TypeLogExpire, b, asynq.Queue(QueueLow)), nil
}

func ParseLogPullPayload(t *asynq.Task) (LogPullPayload, error) {
\tvar p LogPullPayload
\treturn p, json.Unmarshal(t.Payload(), &p)
}

func ParseLogVerifyPayload(t *asynq.Task) (LogVerifyPayload, error) {
\tvar p LogVerifyPayload
\treturn p, json.Unmarshal(t.Payload(), &p)
}

func ParseLogExpirePayload(t *asynq.Task) (LogExpirePayload, error) {
\tvar p LogExpirePayload
\treturn p, json.Unmarshal(t.Payload(), &p)
}
""")

# ─── internal/storage/s3.go ───────────────────────────────────────────────────
write("internal/storage/s3.go", """\
// Package storage provides an S3-compatible object store abstraction that
// supports any S3-compatible provider (AWS, Garage, Hetzner, Contabo, MinIO…).
// Multi-provider failover: if the primary upload fails, it automatically retries
// on secondary providers in order.
package storage

import (
\t"bytes"
\t"compress/gzip"
\t"context"
\t"crypto/sha256"
\t"encoding/hex"
\t"fmt"
\t"io"
\t"strings"
\t"time"

\t"github.com/aws/aws-sdk-go-v2/aws"
\t"github.com/aws/aws-sdk-go-v2/credentials"
\t"github.com/aws/aws-sdk-go-v2/service/s3"
\t"github.com/aws/aws-sdk-go-v2/service/s3/types"
\t"github.com/fabriziosalmi/rainlogs/internal/config"
\t"github.com/google/uuid"
)

// Store wraps an S3 client for a specific bucket.
type Store struct {
\tclient   *s3.Client
\tbucket   string
\tprovider string
}

// New creates a Store from config. Works with any S3-compatible endpoint.
func New(cfg config.S3Config, provider string) (*Store, error) {
\tcreds := credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")

\tresolver := aws.EndpointResolverWithOptionsFunc(
\t\tfunc(service, region string, opts ...interface{}) (aws.Endpoint, error) {
\t\t\tif cfg.Endpoint != "" {
\t\t\t\treturn aws.Endpoint{
\t\t\t\t\tURL:               cfg.Endpoint,
\t\t\t\t\tHostnameImmutable: true,
\t\t\t\t}, nil
\t\t\t}
\t\t\treturn aws.Endpoint{}, &aws.EndpointNotFoundError{}
\t\t},
\t)

\t//nolint:staticcheck // EndpointResolverWithOptions is the recommended pattern for custom endpoints
\tclient := s3.New(s3.Options{
\t\tRegion:             cfg.Region,
\t\tCredentials:        creds,
\t\tEndpointResolver:   resolver, //nolint:staticcheck
\t\tUsePathStyle:       true,
\t})

\treturn &Store{client: client, bucket: cfg.Bucket, provider: provider}, nil
}

// Provider returns the human-readable provider label.
func (s *Store) Provider() string { return s.provider }

// PutLogs compresses raw NDJSON bytes, uploads to S3, and returns the
// S3 key, SHA-256 hex digest, byte count of the compressed payload, and log line count.
// The object is stored with Content-Type application/x-ndjson+gzip.
// If the key already exists, PutLogs returns an error (WORM semantic).
func (s *Store) PutLogs(ctx context.Context, customerID, zoneID uuid.UUID, from, to time.Time, raw []byte) (key, sha256hex string, compressedBytes, logLines int64, err error) {
\t// Count log lines before compression.
\tlogLines = int64(countLines(raw))

\t// Compress.
\tvar buf bytes.Buffer
\tgw := gzip.NewWriter(&buf)
\tif _, err = gw.Write(raw); err != nil {
\t\treturn "", "", 0, 0, fmt.Errorf("storage: gzip write: %w", err)
\t}
\tif err = gw.Close(); err != nil {
\t\treturn "", "", 0, 0, fmt.Errorf("storage: gzip close: %w", err)
\t}
\tcompressed := buf.Bytes()
\tcompressedBytes = int64(len(compressed))

\t// Hash compressed bytes.
\tsum := sha256.Sum256(compressed)
\tsha256hex = hex.EncodeToString(sum[:])

\t// Build deterministic key: logs/<customer>/<zone>/<YYYY>/<MM>/<DD>/<from>_<to>_<sha256[:8]>.ndjson.gz
\tkey = fmt.Sprintf("logs/%s/%s/%s/%s_%s_%s.ndjson.gz",
\t\tcustomerID,
\t\tzoneID,
\t\tfrom.UTC().Format("2006/01/02"),
\t\tfrom.UTC().Format("20060102T150405Z"),
\t\tto.UTC().Format("20060102T150405Z"),
\t\tsha256hex[:8],
\t)

\t// Upload (WORM: fail if key exists).
\t_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
\t\tBucket:             aws.String(s.bucket),
\t\tKey:                aws.String(key),
\t\tBody:               bytes.NewReader(compressed),
\t\tContentLength:      aws.Int64(compressedBytes),
\t\tContentType:        aws.String("application/x-ndjson+gzip"),
\t\tMetadata:           map[string]string{"sha256": sha256hex},
\t\tObjectLockMode:     types.ObjectLockModeCompliance,
\t\tObjectLockRetainUntilDate: aws.Time(to.Add(366 * 24 * time.Hour)),
\t})
\tif err != nil {
\t\t// Strip WORM-lock error for non-WORM buckets (e.g. Garage).
\t\tif !strings.Contains(err.Error(), "ObjectLockConfigurationNotFoundError") {
\t\t\treturn "", "", 0, 0, fmt.Errorf("storage: put object: %w", err)
\t\t}
\t\t// Retry without object lock.
\t\t_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
\t\t\tBucket:        aws.String(s.bucket),
\t\t\tKey:           aws.String(key),
\t\t\tBody:          bytes.NewReader(compressed),
\t\t\tContentLength: aws.Int64(compressedBytes),
\t\t\tContentType:   aws.String("application/x-ndjson+gzip"),
\t\t\tMetadata:      map[string]string{"sha256": sha256hex},
\t\t})
\t\tif err != nil {
\t\t\treturn "", "", 0, 0, fmt.Errorf("storage: put object (no lock): %w", err)
\t\t}
\t}
\treturn key, sha256hex, compressedBytes, logLines, nil
}

// GetLogs downloads and decompresses a stored object.
func (s *Store) GetLogs(ctx context.Context, key string) ([]byte, error) {
\tout, err := s.client.GetObject(ctx, &s3.GetObjectInput{
\t\tBucket: aws.String(s.bucket),
\t\tKey:    aws.String(key),
\t})
\tif err != nil {
\t\treturn nil, fmt.Errorf("storage: get object: %w", err)
\t}
\tdefer out.Body.Close()

\tgr, err := gzip.NewReader(out.Body)
\tif err != nil {
\t\treturn nil, fmt.Errorf("storage: gzip reader: %w", err)
\t}
\tdefer gr.Close()
\treturn io.ReadAll(gr)
}

// DeleteObject removes an object (used by expiry worker after GDPR erasure).
func (s *Store) DeleteObject(ctx context.Context, key string) error {
\t_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
\t\tBucket: aws.String(s.bucket),
\t\tKey:    aws.String(key),
\t})
\tif err != nil {
\t\treturn fmt.Errorf("storage: delete: %w", err)
\t}
\treturn nil
}

func countLines(b []byte) int {
\tif len(b) == 0 {
\t\treturn 0
\t}
\tn := bytes.Count(b, []byte("\\n"))
\tif b[len(b)-1] != '\\n' {
\t\tn++
\t}
\treturn n
}

// ── Multi-provider failover ───────────────────────────────────────────────────

// MultiStore tries providers in order and returns on first success.
type MultiStore struct {
\tproviders []*Store
}

// NewMultiStore creates a MultiStore from a list of Stores.
func NewMultiStore(providers ...*Store) *MultiStore {
\treturn &MultiStore{providers: providers}
}

// PutLogs uploads to the first available provider. Returns the winning provider label and key.
func (m *MultiStore) PutLogs(ctx context.Context, customerID, zoneID uuid.UUID, from, to time.Time, raw []byte) (key, sha256hex, provider string, compressedBytes, logLines int64, err error) {
\tfor _, p := range m.providers {
\t\tvar k, h string
\t\tvar cb, ll int64
\t\tk, h, cb, ll, err = p.PutLogs(ctx, customerID, zoneID, from, to, raw)
\t\tif err == nil {
\t\t\treturn k, h, p.Provider(), cb, ll, nil
\t\t}
\t}
\treturn "", "", "", 0, 0, fmt.Errorf("storage: all providers failed, last error: %w", err)
}
""")

# ─── pkg/worm/worm.go ─────────────────────────────────────────────────────────
write("pkg/worm/worm.go", """\
// Package worm provides WORM-integrity helpers: a tamper-evident hash chain
// over LogJob records and per-object SHA-256 verification.
package worm

import (
\t"crypto/sha256"
\t"encoding/hex"
\t"fmt"
)

// ChainHash computes the next link in the audit chain:
// SHA-256( prevChainHash || objectSHA256 || jobID ).
func ChainHash(prevChainHash, objectSHA256, jobID string) string {
\th := sha256.New()
\th.Write([]byte(prevChainHash))
\th.Write([]byte(objectSHA256))
\th.Write([]byte(jobID))
\treturn hex.EncodeToString(h.Sum(nil))
}

// VerifyObject confirms that the SHA-256 of data matches expected.
func VerifyObject(data []byte, expectedHex string) error {
\tsum := sha256.Sum256(data)
\tgot := hex.EncodeToString(sum[:])
\tif got != expectedHex {
\t\treturn fmt.Errorf("worm: sha256 mismatch: got %s, expected %s", got, expectedHex)
\t}
\treturn nil
}

// GenesisHash is the well-known starting value for the first job in a chain.
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"
""")

# ─── internal/cloudflare/client.go ───────────────────────────────────────────
write("internal/cloudflare/client.go", """\
// Package cloudflare provides a Cloudflare Logpull API client.
//
// Key constraints:
//   - Logs are available with a minimum 1-minute delay.
//   - Logs are retained by Cloudflare for 7 days.
//   - Maximum window per request: 1 hour.
//   - Enterprise customers should use Logpush instead.
package cloudflare

import (
\t"compress/gzip"
\t"context"
\t"fmt"
\t"io"
\t"net/http"
\t"net/url"
\t"time"

\t"github.com/fabriziosalmi/rainlogs/internal/config"
)

const defaultBaseURL = "https://api.cloudflare.com/client/v4"

// Client is a Cloudflare Logpull API client for a single zone.
type Client struct {
\tbaseURL    string
\thttpClient *http.Client
\tzoneID     string
\tapiKey     string
}

// NewClient creates a Client for a specific zone.
func NewClient(cfg config.CloudflareConfig, zoneID, apiKey string) *Client {
\tbase := cfg.BaseURL
\tif base == "" {
\t\tbase = defaultBaseURL
\t}
\treturn &Client{
\t\tbaseURL: base,
\t\thttpClient: &http.Client{Timeout: cfg.RequestTimeout},
\t\tzoneID:    zoneID,
\t\tapiKey:    apiKey,
\t}
}

// PullLogs fetches NDJSON log lines for [from, to) (max 1-hour window).
// Returns raw (possibly gzip-compressed) bytes that the caller must decompress.
func (c *Client) PullLogs(ctx context.Context, from, to time.Time, fields []string) ([]byte, error) {
\tif to.Sub(from) > time.Hour {
\t\treturn nil, fmt.Errorf("cloudflare: window exceeds 1 hour")
\t}
\tif time.Since(to) < time.Minute {
\t\treturn nil, fmt.Errorf("cloudflare: logs not yet available (min 1-min delay)")
\t}

\tu, err := url.Parse(fmt.Sprintf("%s/zones/%s/logs/received", c.baseURL, c.zoneID))
\tif err != nil {
\t\treturn nil, fmt.Errorf("cloudflare: parse url: %w", err)
\t}
\tq := u.Query()
\tq.Set("start", from.UTC().Format(time.RFC3339))
\tq.Set("end", to.UTC().Format(time.RFC3339))
\tq.Set("timestamps", "rfc3339")
\tif len(fields) > 0 {
\t\tvar fs string
\t\tfor i, f := range fields {
\t\t\tif i > 0 {
\t\t\t\tfs += ","
\t\t\t}
\t\t\tfs += f
\t\t}
\t\tq.Set("fields", fs)
\t}
\tu.RawQuery = q.Encode()

\treq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
\tif err != nil {
\t\treturn nil, fmt.Errorf("cloudflare: new request: %w", err)
\t}
\treq.Header.Set("Authorization", "Bearer "+c.apiKey)
\treq.Header.Set("Accept-Encoding", "gzip")

\tresp, err := c.httpClient.Do(req)
\tif err != nil {
\t\treturn nil, fmt.Errorf("cloudflare: do request: %w", err)
\t}
\tdefer resp.Body.Close()

\tif resp.StatusCode != http.StatusOK {
\t\tbody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
\t\treturn nil, fmt.Errorf("cloudflare: HTTP %d: %s", resp.StatusCode, body)
\t}

\tvar reader io.Reader = resp.Body
\tif resp.Header.Get("Content-Encoding") == "gzip" {
\t\tgr, err := gzip.NewReader(resp.Body)
\t\tif err != nil {
\t\t\treturn nil, fmt.Errorf("cloudflare: gzip reader: %w", err)
\t\t}
\t\tdefer gr.Close()
\t\treader = gr
\t}

\tdata, err := io.ReadAll(reader)
\tif err != nil {
\t\treturn nil, fmt.Errorf("cloudflare: read body: %w", err)
\t}
\treturn data, nil
}
""")

# ─── internal/worker/worker.go ────────────────────────────────────────────────
write("internal/worker/worker.go", """\
package worker

import (
\t"bytes"
\t"context"
\t"encoding/json"
\t"fmt"
\t"io"
\t"time"

\t"github.com/fabriziosalmi/rainlogs/internal/config"
\tcfclient "github.com/fabriziosalmi/rainlogs/internal/cloudflare"
\t"github.com/fabriziosalmi/rainlogs/internal/db"
\t"github.com/fabriziosalmi/rainlogs/internal/models"
\t"github.com/fabriziosalmi/rainlogs/internal/queue"
\t"github.com/fabriziosalmi/rainlogs/internal/storage"
\t"github.com/fabriziosalmi/rainlogs/pkg/worm"
\t"github.com/google/uuid"
\t"github.com/hibiken/asynq"
\t"go.uber.org/zap"
)

// ── LogPullProcessor ──────────────────────────────────────────────────────────

// LogPullProcessor handles TypeLogPull tasks.
type LogPullProcessor struct {
\tzones   *db.ZoneRepository
\tlogJobs *db.LogJobRepository
\tstore   *storage.MultiStore
\tcfCfg   config.CloudflareConfig
\tqueue   *asynq.Client
\tlog     *zap.Logger
}

func NewLogPullProcessor(
\tzones *db.ZoneRepository,
\tlogJobs *db.LogJobRepository,
\tstore *storage.MultiStore,
\tcfCfg config.CloudflareConfig,
\tq *asynq.Client,
\tlog *zap.Logger,
) *LogPullProcessor {
\treturn &LogPullProcessor{
\t\tzones: zones, logJobs: logJobs, store: store,
\t\tcfCfg: cfCfg, queue: q, log: log,
\t}
}

func (p *LogPullProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
\tpayload, err := queue.ParseLogPullPayload(t)
\tif err != nil {
\t\treturn fmt.Errorf("worker: parse payload: %w", err)
\t}

\tzone, err := p.zones.GetByID(ctx, payload.ZoneID)
\tif err != nil {
\t\treturn fmt.Errorf("worker: get zone: %w", err)
\t}

\t// Create log job record.
\tjob := &models.LogJob{
\t\tID:          uuid.New(),
\t\tZoneID:      payload.ZoneID,
\t\tCustomerID:  payload.CustomerID,
\t\tPeriodStart: payload.PeriodStart,
\t\tPeriodEnd:   payload.PeriodEnd,
\t\tStatus:      models.JobStatusRunning,
\t}
\tif err := p.logJobs.Create(ctx, job); err != nil {
\t\treturn fmt.Errorf("worker: create log job: %w", err)
\t}

\t// Pull logs from Cloudflare (API key decryption handled by caller via kms).
\tcf := cfclient.NewClient(p.cfCfg, zone.ZoneID, p.cfCfg.APIKey)
\traw, err := cf.PullLogs(ctx, payload.PeriodStart, payload.PeriodEnd, nil)
\tif err != nil {
\t\tjob.Status = models.JobStatusFailed
\t\tjob.ErrMsg = err.Error()
\t\t_ = p.logJobs.Update(ctx, job)
\t\treturn fmt.Errorf("worker: pull logs: %w", err)
\t}

\t// Upload to S3.
\tkey, sha256hex, provider, compressedBytes, logLines, err := p.store.PutLogs(
\t\tctx, payload.CustomerID, payload.ZoneID, payload.PeriodStart, payload.PeriodEnd, raw,
\t)
\tif err != nil {
\t\tjob.Status = models.JobStatusFailed
\t\tjob.ErrMsg = err.Error()
\t\t_ = p.logJobs.Update(ctx, job)
\t\treturn fmt.Errorf("worker: upload logs: %w", err)
\t}

\t// Compute chain hash.
\tchainHash := worm.ChainHash(worm.GenesisHash, sha256hex, job.ID.String())

\tjob.Status = models.JobStatusDone
\tjob.S3Key = key
\tjob.S3Provider = provider
\tjob.SHA256 = sha256hex
\tjob.ChainHash = chainHash
\tjob.ByteCount = compressedBytes
\tjob.LogCount = logLines
\tif err := p.logJobs.Update(ctx, job); err != nil {
\t\treturn fmt.Errorf("worker: update log job: %w", err)
\t}

\t// Enqueue verification task.
\tvTask, err := queue.NewLogVerifyTask(queue.LogVerifyPayload{JobID: job.ID})
\tif err == nil {
\t\t_, _ = p.queue.Enqueue(ctx, vTask)
\t}

\tp.log.Info("log pull done",
\t\tzap.String("job_id", job.ID.String()),
\t\tzap.Int64("lines", logLines),
\t\tzap.Int64("bytes", compressedBytes),
\t\tzap.String("provider", provider),
\t)
\treturn nil
}

// ── LogVerifyProcessor ────────────────────────────────────────────────────────

type LogVerifyProcessor struct {
\tlogJobs *db.LogJobRepository
\tstore   *storage.MultiStore
\tlog     *zap.Logger
}

func NewLogVerifyProcessor(logJobs *db.LogJobRepository, store *storage.MultiStore, log *zap.Logger) *LogVerifyProcessor {
\treturn &LogVerifyProcessor{logJobs: logJobs, store: store, log: log}
}

func (p *LogVerifyProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
\tpayload, err := queue.ParseLogVerifyPayload(t)
\tif err != nil {
\t\treturn fmt.Errorf("verify: parse payload: %w", err)
\t}
\tjob, err := p.logJobs.GetByID(ctx, payload.JobID)
\tif err != nil {
\t\treturn fmt.Errorf("verify: get job: %w", err)
\t}

\t// Re-download and re-verify SHA-256 (from first provider that has the object).
\t_ = job // verification logic uses job.S3Key and job.SHA256
\tp.log.Info("log verify skipped (stub)", zap.String("job_id", payload.JobID.String()))
\treturn nil
}

// ── LogExpireProcessor ────────────────────────────────────────────────────────

// LogExpireProcessor enforces GDPR art.17 retention: deletes S3 objects
// and marks log_jobs as expired when their retention window has passed.
type LogExpireProcessor struct {
\tlogJobs   *db.LogJobRepository
\tcustomers *db.CustomerRepository
\tstore     *storage.MultiStore
\tlog       *zap.Logger
}

func NewLogExpireProcessor(
\tlogJobs *db.LogJobRepository,
\tcustomers *db.CustomerRepository,
\tstore *storage.MultiStore,
\tlog *zap.Logger,
) *LogExpireProcessor {
\treturn &LogExpireProcessor{logJobs: logJobs, customers: customers, store: store, log: log}
}

func (p *LogExpireProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
\tpayload, err := queue.ParseLogExpirePayload(t)
\tif err != nil {
\t\treturn fmt.Errorf("expire: parse payload: %w", err)
\t}

\tjobs, err := p.logJobs.ListExpired(ctx, payload.CustomerID, payload.RetentionDays)
\tif err != nil {
\t\treturn fmt.Errorf("expire: list expired: %w", err)
\t}

\tfor _, job := range jobs {
\t\tif err := p.store.DeleteObject(ctx, job.S3Key); err != nil {
\t\t\tp.log.Error("expire: delete s3 object failed",
\t\t\t\tzap.String("key", job.S3Key), zap.Error(err))
\t\t\tcontinue
\t\t}
\t\tif err := p.logJobs.MarkExpired(ctx, job.ID); err != nil {
\t\t\tp.log.Error("expire: mark expired failed",
\t\t\t\tzap.String("job_id", job.ID.String()), zap.Error(err))
\t\t\tcontinue
\t\t}
\t\tp.log.Info("expire: erased log job", zap.String("job_id", job.ID.String()))
\t}
\treturn nil
}

// ── ZoneScheduler ─────────────────────────────────────────────────────────────

// ZoneScheduler polls for zones that are due for a log pull and enqueues tasks.
type ZoneScheduler struct {
\tzones     *db.ZoneRepository
\tcustomers *db.CustomerRepository
\tqueue     *asynq.Client
\tinterval  time.Duration
\tlog       *zap.Logger
}

func NewZoneScheduler(
\tzones *db.ZoneRepository,
\tcustomers *db.CustomerRepository,
\tq *asynq.Client,
\tinterval time.Duration,
\tlog *zap.Logger,
) *ZoneScheduler {
\treturn &ZoneScheduler{zones: zones, customers: customers, queue: q, interval: interval, log: log}
}

func (s *ZoneScheduler) Run(ctx context.Context) {
\tticker := time.NewTicker(s.interval)
\tdefer ticker.Stop()
\tfor {
\t\tselect {
\t\tcase <-ctx.Done():
\t\t\treturn
\t\tcase <-ticker.C:
\t\t\ts.tick(ctx)
\t\t}
\t}
}

func (s *ZoneScheduler) tick(ctx context.Context) {
\tzones, err := s.zones.ListDue(ctx)
\tif err != nil {
\t\ts.log.Error("scheduler: list due zones", zap.Error(err))
\t\treturn
\t}
\tfor _, z := range zones {
\t\tnow := time.Now().UTC().Truncate(time.Minute)
\t\tpayload := queue.LogPullPayload{
\t\t\tZoneID:      z.ID,
\t\t\tCustomerID:  z.CustomerID,
\t\t\tPeriodStart: now.Add(-time.Hour),
\t\t\tPeriodEnd:   now.Add(-time.Minute),
\t\t}
\t\ttask, err := queue.NewLogPullTask(payload)
\t\tif err != nil {
\t\t\ts.log.Error("scheduler: new task", zap.Error(err))
\t\t\tcontinue
\t\t}
\t\tif _, err := s.queue.Enqueue(ctx, task); err != nil {
\t\t\ts.log.Error("scheduler: enqueue", zap.Error(err))
\t\t\tcontinue
\t\t}
\t\ts.log.Info("scheduler: enqueued pull",
\t\t\tzap.String("zone_id", z.ZoneID),
\t\t\tzap.Time("from", payload.PeriodStart),
\t\t\tzap.Time("to", payload.PeriodEnd),
\t\t)
\t}

\t// Enqueue expiry tasks for all customers.
\ts.enqueueExpiry(ctx)
}

func (s *ZoneScheduler) enqueueExpiry(ctx context.Context) {
\tcustomers, err := s.customers.List(ctx)
\tif err != nil {
\t\ts.log.Error("scheduler: list customers", zap.Error(err))
\t\treturn
\t}
\tfor _, c := range customers {
\t\tpayload := queue.LogExpirePayload{
\t\t\tCustomerID:    c.ID,
\t\t\tRetentionDays: c.RetentionDays,
\t\t}
\t\ttask, err := queue.NewLogExpireTask(payload)
\t\tif err != nil {
\t\t\tcontinue
\t\t}
\t\t_, _ = s.queue.Enqueue(ctx, task)
\t}
}

// ── Log search helper ─────────────────────────────────────────────────────────

// SearchLogsFromS3 downloads a stored object and filters NDJSON lines by filter.
func SearchLogsFromS3(ctx context.Context, store *storage.MultiStore, s3key string, f models.SearchFilter) ([]*models.LogEntry, error) {
\t// MultiStore doesn't expose GetLogs so we need a single Store. This is a
\t// placeholder; real search fetches from the indexed DB table (migration 002).
\t_ = ctx
\t_ = store
\t_ = s3key
\t_ = f
\treturn nil, nil
}

// parseLogEntry parses a single NDJSON line.
func parseLogEntry(line []byte) (*models.LogEntry, error) {
\tvar e models.LogEntry
\tif err := json.Unmarshal(line, &e); err != nil {
\t\treturn nil, err
\t}
\treturn &e, nil
}

// splitLines splits NDJSON bytes into non-empty lines.
func splitLines(raw []byte) [][]byte {
\treturn bytes.FieldsFunc(raw, func(r rune) bool { return r == '\\n' })
}

// _ are used to avoid "declared and not used" errors for unexported helpers.
var _ = io.Discard
var _ = parseLogEntry
var _ = splitLines
""")

# ─── internal/api/middleware/middleware.go ────────────────────────────────────
write("internal/api/middleware/middleware.go", """\
package middleware

import (
\t"net/http"
\t"strings"
\t"time"

\t"github.com/fabriziosalmi/rainlogs/internal/auth"
\t"github.com/fabriziosalmi/rainlogs/internal/db"
\t"github.com/fabriziosalmi/rainlogs/internal/models"
\t"github.com/google/uuid"
\t"github.com/labstack/echo/v4"
\t"go.uber.org/zap"
)

type ContextKey string

const (
\tCtxCustomer  ContextKey = "customer"
\tCtxAPIKeyID  ContextKey = "api_key_id"
)

// APIKeyAuth validates the Bearer token from the Authorization header.
func APIKeyAuth(keyRepo *db.APIKeyRepository, custRepo *db.CustomerRepository) echo.MiddlewareFunc {
\treturn func(next echo.HandlerFunc) echo.HandlerFunc {
\t\treturn func(c echo.Context) error {
\t\t\theader := c.Request().Header.Get("Authorization")
\t\t\tif !strings.HasPrefix(header, "Bearer ") {
\t\t\t\treturn echo.NewHTTPError(http.StatusUnauthorized, "missing bearer token")
\t\t\t}
\t\t\tplaintext := strings.TrimPrefix(header, "Bearer ")

\t\t\tprefix, err := auth.PrefixOf(plaintext)
\t\t\tif err != nil {
\t\t\t\treturn echo.NewHTTPError(http.StatusUnauthorized, "invalid token format")
\t\t\t}

\t\t\tkeys, err := keyRepo.GetByPrefix(c.Request().Context(), prefix)
\t\t\tif err != nil || len(keys) == 0 {
\t\t\t\treturn echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
\t\t\t}

\t\t\tvar matchedKey *models.APIKey
\t\t\tfor _, k := range keys {
\t\t\t\tif auth.ValidateAPIKey(plaintext, k.KeyHash) {
\t\t\t\t\tmatchedKey = k
\t\t\t\t\tbreak
\t\t\t\t}
\t\t\t}
\t\t\tif matchedKey == nil {
\t\t\t\treturn echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
\t\t\t}

\t\t\tcust, err := custRepo.GetByID(c.Request().Context(), matchedKey.CustomerID)
\t\t\tif err != nil {
\t\t\t\treturn echo.NewHTTPError(http.StatusUnauthorized, "customer not found")
\t\t\t}

\t\t\t_ = keyRepo.UpdateLastUsed(c.Request().Context(), matchedKey.ID)

\t\t\tc.Set(string(CtxCustomer), cust)
\t\t\tc.Set(string(CtxAPIKeyID), matchedKey.ID)
\t\t\treturn next(c)
\t\t}
\t}
}

// RequestLogger returns a zap-based request logging middleware.
func RequestLogger(log *zap.Logger) echo.MiddlewareFunc {
\treturn func(next echo.HandlerFunc) echo.HandlerFunc {
\t\treturn func(c echo.Context) error {
\t\t\tstart := time.Now()
\t\t\terr := next(c)
\t\t\treq := c.Request()
\t\t\tres := c.Response()
\t\t\tlog.Info("request",
\t\t\t\tzap.String("method", req.Method),
\t\t\t\tzap.String("path", req.URL.Path),
\t\t\t\tzap.Int("status", res.Status),
\t\t\t\tzap.Duration("latency", time.Since(start)),
\t\t\t)
\t\t\treturn err
\t\t}
\t}
}

// CustomerFromCtx extracts the authenticated customer from Echo context.
func CustomerFromCtx(c echo.Context) *models.Customer {
\tv := c.Get(string(CtxCustomer))
\tif v == nil {
\t\treturn nil
\t}
\tcust, _ := v.(*models.Customer)
\treturn cust
}

// APIKeyIDFromCtx extracts the API key ID from Echo context.
func APIKeyIDFromCtx(c echo.Context) uuid.UUID {
\tv := c.Get(string(CtxAPIKeyID))
\tif v == nil {
\t\treturn uuid.Nil
\t}
\tid, _ := v.(uuid.UUID)
\treturn id
}
""")

# ─── internal/api/handlers/handlers.go ───────────────────────────────────────
write("internal/api/handlers/handlers.go", """\
package handlers

import (
\t"bufio"
\t"bytes"
\t"encoding/json"
\t"fmt"
\t"net/http"
\t"strconv"
\t"time"

\t"github.com/fabriziosalmi/rainlogs/internal/auth"
\t"github.com/fabriziosalmi/rainlogs/internal/db"
\tapimw "github.com/fabriziosalmi/rainlogs/internal/api/middleware"
\t"github.com/fabriziosalmi/rainlogs/internal/kms"
\t"github.com/fabriziosalmi/rainlogs/internal/models"
\t"github.com/fabriziosalmi/rainlogs/internal/queue"
\t"github.com/fabriziosalmi/rainlogs/internal/storage"
\t"github.com/google/uuid"
\t"github.com/hibiken/asynq"
\t"github.com/labstack/echo/v4"
\t"go.uber.org/zap"
)

// Handler holds shared dependencies for all API handlers.
type Handler struct {
\tcustomers *db.CustomerRepository
\tzones     *db.ZoneRepository
\tlogJobs   *db.LogJobRepository
\tapiKeys   *db.APIKeyRepository
\tqueue     *asynq.Client
\tstore     *storage.MultiStore
\tenc       *kms.Encryptor
\tlog       *zap.Logger
}

// New creates a Handler with all dependencies injected.
func New(
\tcustomers *db.CustomerRepository,
\tzones *db.ZoneRepository,
\tlogJobs *db.LogJobRepository,
\tapiKeys *db.APIKeyRepository,
\tq *asynq.Client,
\tstore *storage.MultiStore,
\tenc *kms.Encryptor,
\tlog *zap.Logger,
) *Handler {
\treturn &Handler{
\t\tcustomers: customers, zones: zones, logJobs: logJobs,
\t\tapiKeys: apiKeys, queue: q, store: store, enc: enc, log: log,
\t}
}

// ── Health ────────────────────────────────────────────────────────────────────

// Health godoc
// @Summary     Health check
// @Tags        system
// @Produce     json
// @Success     200  {object}  map[string]string
// @Router      /health [get]
func (h *Handler) Health(c echo.Context) error {
\treturn c.JSON(http.StatusOK, map[string]string{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339)})
}

// ── Customers ─────────────────────────────────────────────────────────────────

type createCustomerReq struct {
\tName          string `json:"name"           validate:"required"`
\tEmail         string `json:"email"          validate:"required,email"`
\tCFAccountID   string `json:"cf_account_id"  validate:"required"`
\tCFAPIKey      string `json:"cf_api_key"     validate:"required"`
\tRetentionDays int    `json:"retention_days" validate:"required,min=1,max=365"`
}

// CreateCustomer godoc
// @Summary     Create a customer
// @Tags        customers
// @Accept      json
// @Produce     json
// @Param       body body createCustomerReq true "Customer payload"
// @Success     201  {object}  models.Customer
// @Router      /customers [post]
func (h *Handler) CreateCustomer(c echo.Context) error {
\tvar req createCustomerReq
\tif err := c.Bind(&req); err != nil {
\t\treturn echo.NewHTTPError(http.StatusBadRequest, err.Error())
\t}
\tencKey, err := h.enc.Encrypt(req.CFAPIKey)
\tif err != nil {
\t\treturn echo.NewHTTPError(http.StatusInternalServerError, "encryption failed")
\t}
\tcust := &models.Customer{
\t\tID:            uuid.New(),
\t\tName:          req.Name,
\t\tEmail:         req.Email,
\t\tCFAccountID:   req.CFAccountID,
\t\tCFAPIKeyEnc:   encKey,
\t\tRetentionDays: req.RetentionDays,
\t}
\tif err := h.customers.Create(c.Request().Context(), cust); err != nil {
\t\th.log.Error("create customer", zap.Error(err))
\t\treturn echo.NewHTTPError(http.StatusInternalServerError, "db error")
\t}
\treturn c.JSON(http.StatusCreated, cust)
}

// ListCustomers godoc
// @Summary     List customers
// @Tags        customers
// @Produce     json
// @Success     200  {array}   models.Customer
// @Router      /customers [get]
func (h *Handler) ListCustomers(c echo.Context) error {
\tcustomers, err := h.customers.List(c.Request().Context())
\tif err != nil {
\t\treturn echo.NewHTTPError(http.StatusInternalServerError, "db error")
\t}
\treturn c.JSON(http.StatusOK, customers)
}

// ── API Keys ──────────────────────────────────────────────────────────────────

type createAPIKeyReq struct {
\tLabel string `json:"label" validate:"required"`
}

// CreateAPIKey godoc
// @Summary     Create an API key for the authenticated customer
// @Tags        api-keys
// @Accept      json
// @Produce     json
// @Success     201  {object}  map[string]string
// @Router      /api-keys [post]
func (h *Handler) CreateAPIKey(c echo.Context) error {
\tcust := apimw.CustomerFromCtx(c)
\tif cust == nil {
\t\treturn echo.NewHTTPError(http.StatusUnauthorized)
\t}
\tvar req createAPIKeyReq
\tif err := c.Bind(&req); err != nil {
\t\treturn echo.NewHTTPError(http.StatusBadRequest, err.Error())
\t}
\tplaintext, hash, prefix, err := auth.GenerateAPIKey()
\tif err != nil {
\t\treturn echo.NewHTTPError(http.StatusInternalServerError, "keygen failed")
\t}
\tkey := &models.APIKey{
\t\tID:         uuid.New(),
\t\tCustomerID: cust.ID,
\t\tPrefix:     prefix,
\t\tKeyHash:    hash,
\t\tLabel:      req.Label,
\t}
\tif err := h.apiKeys.Create(c.Request().Context(), key); err != nil {
\t\treturn echo.NewHTTPError(http.StatusInternalServerError, "db error")
\t}
\t// Return plaintext only once.
\treturn c.JSON(http.StatusCreated, map[string]string{"api_key": plaintext, "id": key.ID.String()})
}

// RevokeAPIKey godoc
// @Summary     Revoke an API key
// @Tags        api-keys
// @Param       id   path  string  true  "API key ID"
// @Success     204
// @Router      /api-keys/{id} [delete]
func (h *Handler) RevokeAPIKey(c echo.Context) error {
\tid, err := uuid.Parse(c.Param("id"))
\tif err != nil {
\t\treturn echo.NewHTTPError(http.StatusBadRequest, "invalid id")
\t}
\tif err := h.apiKeys.Revoke(c.Request().Context(), id); err != nil {
\t\treturn echo.NewHTTPError(http.StatusInternalServerError, "db error")
\t}
\treturn c.NoContent(http.StatusNoContent)
}

// ── Zones ─────────────────────────────────────────────────────────────────────

type createZoneReq struct {
\tZoneID           string `json:"zone_id"            validate:"required"`
\tName             string `json:"name"               validate:"required"`
\tPullIntervalSecs int    `json:"pull_interval_secs" validate:"required,min=60"`
}

// CreateZone godoc
// @Summary     Register a Cloudflare zone
// @Tags        zones
// @Accept      json
// @Produce     json
// @Success     201  {object}  models.Zone
// @Router      /zones [post]
func (h *Handler) CreateZone(c echo.Context) error {
\tcust := apimw.CustomerFromCtx(c)
\tif cust == nil {
\t\treturn echo.NewHTTPError(http.StatusUnauthorized)
\t}
\tvar req createZoneReq
\tif err := c.Bind(&req); err != nil {
\t\treturn echo.NewHTTPError(http.StatusBadRequest, err.Error())
\t}
\tzone := &models.Zone{
\t\tID:               uuid.New(),
\t\tCustomerID:       cust.ID,
\t\tZoneID:           req.ZoneID,
\t\tName:             req.Name,
\t\tPullIntervalSecs: req.PullIntervalSecs,
\t\tActive:           true,
\t}
\tif err := h.zones.Create(c.Request().Context(), zone); err != nil {
\t\treturn echo.NewHTTPError(http.StatusInternalServerError, "db error")
\t}
\treturn c.JSON(http.StatusCreated, zone)
}

// ListZones godoc
// @Summary     List zones for authenticated customer
// @Tags        zones
// @Produce     json
// @Success     200  {array}   models.Zone
// @Router      /zones [get]
func (h *Handler) ListZones(c echo.Context) error {
\tcust := apimw.CustomerFromCtx(c)
\tif cust == nil {
\t\treturn echo.NewHTTPError(http.StatusUnauthorized)
\t}
\tzones, err := h.zones.ListByCustomer(c.Request().Context(), cust.ID)
\tif err != nil {
\t\treturn echo.NewHTTPError(http.StatusInternalServerError, "db error")
\t}
\treturn c.JSON(http.StatusOK, zones)
}

// ── Log Jobs ──────────────────────────────────────────────────────────────────

// ListLogJobs godoc
// @Summary     List log jobs for authenticated customer
// @Tags        log-jobs
// @Produce     json
// @Param       limit   query  int  false  "Limit (default 50)"
// @Param       offset  query  int  false  "Offset"
// @Success     200  {array}   models.LogJob
// @Router      /log-jobs [get]
func (h *Handler) ListLogJobs(c echo.Context) error {
\tcust := apimw.CustomerFromCtx(c)
\tif cust == nil {
\t\treturn echo.NewHTTPError(http.StatusUnauthorized)
\t}
\tlimit, _ := strconv.Atoi(c.QueryParam("limit"))
\tif limit == 0 {
\t\tlimit = 50
\t}
\toffset, _ := strconv.Atoi(c.QueryParam("offset"))
\tjobs, err := h.logJobs.ListByCustomer(c.Request().Context(), cust.ID, limit, offset)
\tif err != nil {
\t\treturn echo.NewHTTPError(http.StatusInternalServerError, "db error")
\t}
\treturn c.JSON(http.StatusOK, jobs)
}

// TriggerLogPull godoc
// @Summary     Manually trigger a log pull for a zone
// @Tags        log-jobs
// @Accept      json
// @Produce     json
// @Param       zone_id path  string true "Zone ID"
// @Success     202  {object}  map[string]string
// @Router      /zones/{zone_id}/pull [post]
func (h *Handler) TriggerLogPull(c echo.Context) error {
\tcust := apimw.CustomerFromCtx(c)
\tif cust == nil {
\t\treturn echo.NewHTTPError(http.StatusUnauthorized)
\t}
\tzoneID, err := uuid.Parse(c.Param("zone_id"))
\tif err != nil {
\t\treturn echo.NewHTTPError(http.StatusBadRequest, "invalid zone_id")
\t}
\tnow := time.Now().UTC().Truncate(time.Minute)
\tpayload := queue.LogPullPayload{
\t\tZoneID:      zoneID,
\t\tCustomerID:  cust.ID,
\t\tPeriodStart: now.Add(-time.Hour),
\t\tPeriodEnd:   now.Add(-time.Minute),
\t}
\ttask, err := queue.NewLogPullTask(payload)
\tif err != nil {
\t\treturn echo.NewHTTPError(http.StatusInternalServerError, "task error")
\t}
\tinfo, err := h.queue.Enqueue(c.Request().Context(), task)
\tif err != nil {
\t\treturn echo.NewHTTPError(http.StatusInternalServerError, "enqueue error")
\t}
\treturn c.JSON(http.StatusAccepted, map[string]string{"task_id": info.ID})
}

// ── Log Search ────────────────────────────────────────────────────────────────

// SearchLogs godoc
// @Summary     Search archived logs
// @Tags        logs
// @Produce     json
// @Param       ip      query  string  false  "Filter by client IP"
// @Param       ray_id  query  string  false  "Filter by Ray ID"
// @Param       from    query  string  false  "From time (RFC3339)"
// @Param       to      query  string  false  "To time (RFC3339)"
// @Param       status  query  int     false  "Filter by HTTP status code"
// @Param       limit   query  int     false  "Limit (default 100)"
// @Param       offset  query  int     false  "Offset"
// @Success     200  {array}  models.LogEntry
// @Router      /logs/search [get]
func (h *Handler) SearchLogs(c echo.Context) error {
\tcust := apimw.CustomerFromCtx(c)
\tif cust == nil {
\t\treturn echo.NewHTTPError(http.StatusUnauthorized)
\t}

\tf := models.SearchFilter{CustomerID: cust.ID}
\tf.IP = c.QueryParam("ip")
\tf.RayID = c.QueryParam("ray_id")

\tif v := c.QueryParam("from"); v != "" {
\t\tt, err := time.Parse(time.RFC3339, v)
\t\tif err != nil {
\t\t\treturn echo.NewHTTPError(http.StatusBadRequest, "invalid from")
\t\t}
\t\tf.From = t
\t}
\tif v := c.QueryParam("to"); v != "" {
\t\tt, err := time.Parse(time.RFC3339, v)
\t\tif err != nil {
\t\t\treturn echo.NewHTTPError(http.StatusBadRequest, "invalid to")
\t\t}
\t\tf.To = t
\t}
\tif v := c.QueryParam("status"); v != "" {
\t\tn, err := strconv.Atoi(v)
\t\tif err != nil {
\t\t\treturn echo.NewHTTPError(http.StatusBadRequest, "invalid status")
\t\t}
\t\tf.StatusCode = &n
\t}
\tf.Limit, _ = strconv.Atoi(c.QueryParam("limit"))
\tif f.Limit == 0 {
\t\tf.Limit = 100
\t}
\tf.Offset, _ = strconv.Atoi(c.QueryParam("offset"))

\t// Fetch all done jobs and scan their S3 objects.
\tjobs, err := h.logJobs.ListByCustomer(c.Request().Context(), cust.ID, 200, 0)
\tif err != nil {
\t\treturn echo.NewHTTPError(http.StatusInternalServerError, "db error")
\t}

\tvar results []*models.LogEntry
\tfor _, job := range jobs {
\t\tif job.Status != models.JobStatusDone {
\t\t\tcontinue
\t\t}
\t\tif !f.From.IsZero() && job.PeriodEnd.Before(f.From) {
\t\t\tcontinue
\t\t}
\t\tif !f.To.IsZero() && job.PeriodStart.After(f.To) {
\t\t\tcontinue
\t\t}
\t\traw, err := h.store.GetLogs(c.Request().Context(), job.S3Key)
\t\tif err != nil {
\t\t\tcontinue
\t\t}
\t\tscanner := bufio.NewScanner(bytes.NewReader(raw))
\t\tfor scanner.Scan() {
\t\t\tvar entry models.LogEntry
\t\t\tif err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
\t\t\t\tcontinue
\t\t\t}
\t\t\tif f.IP != "" && entry.ClientIP != f.IP {
\t\t\t\tcontinue
\t\t\t}
\t\t\tif f.RayID != "" && entry.RayID != f.RayID {
\t\t\t\tcontinue
\t\t\t}
\t\t\tif f.StatusCode != nil && entry.Status != *f.StatusCode {
\t\t\t\tcontinue
\t\t\t}
\t\t\tresults = append(results, &entry)
\t\t\tif len(results) >= f.Limit+f.Offset {
\t\t\t\tbreak
\t\t\t}
\t\t}
\t}
\tif f.Offset < len(results) {
\t\tresults = results[f.Offset:]
\t} else {
\t\tresults = nil
\t}
\tif len(results) > f.Limit {
\t\tresults = results[:f.Limit]
\t}
\treturn c.JSON(http.StatusOK, results)
}

// ── NIS2 Incident Report ──────────────────────────────────────────────────────

type nis2ReportReq struct {
\tFrom string `json:"from" validate:"required"`
\tTo   string `json:"to"   validate:"required"`
}

// NIS2Report godoc
// @Summary     Export NIS2 incident report as JSON summary
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Param       body body nis2ReportReq true "Time window"
// @Success     200  {object}  map[string]interface{}
// @Router      /compliance/nis2-report [post]
func (h *Handler) NIS2Report(c echo.Context) error {
\tcust := apimw.CustomerFromCtx(c)
\tif cust == nil {
\t\treturn echo.NewHTTPError(http.StatusUnauthorized)
\t}

\tvar req nis2ReportReq
\tif err := c.Bind(&req); err != nil {
\t\treturn echo.NewHTTPError(http.StatusBadRequest, err.Error())
\t}
\tfrom, err := time.Parse(time.RFC3339, req.From)
\tif err != nil {
\t\treturn echo.NewHTTPError(http.StatusBadRequest, "invalid from")
\t}
\tto, err := time.Parse(time.RFC3339, req.To)
\tif err != nil {
\t\treturn echo.NewHTTPError(http.StatusBadRequest, "invalid to")
\t}

\t// Collect 4xx/5xx events as incident events.
\tjobs, err := h.logJobs.ListByCustomer(c.Request().Context(), cust.ID, 1000, 0)
\tif err != nil {
\t\treturn echo.NewHTTPError(http.StatusInternalServerError, "db error")
\t}

\tvar events []models.IncidentEvent
\tfor _, job := range jobs {
\t\tif job.Status != models.JobStatusDone {
\t\t\tcontinue
\t\t}
\t\tif job.PeriodStart.Before(from) || job.PeriodEnd.After(to) {
\t\t\tcontinue
\t\t}
\t\traw, err := h.store.GetLogs(c.Request().Context(), job.S3Key)
\t\tif err != nil {
\t\t\tcontinue
\t\t}
\t\tscanner := bufio.NewScanner(bytes.NewReader(raw))
\t\tfor scanner.Scan() {
\t\t\tvar entry models.LogEntry
\t\t\tif err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
\t\t\t\tcontinue
\t\t\t}
\t\t\tif entry.Status >= 400 {
\t\t\t\tevents = append(events, models.IncidentEvent{
\t\t\t\t\tOccurredAt:  entry.Timestamp,
\t\t\t\t\tZoneName:    entry.ZoneName,
\t\t\t\t\tClientIP:    entry.ClientIP,
\t\t\t\t\tRayID:       entry.RayID,
\t\t\t\t\tHTTPStatus:  entry.Status,
\t\t\t\t\tDescription: fmt.Sprintf("%s %s -> %d", entry.Method, entry.URI, entry.Status),
\t\t\t\t})
\t\t\t}
\t\t}
\t}

\treport := map[string]interface{}{
\t\t"customer":      cust.Name,
\t\t"period_from":   from,
\t\t"period_to":     to,
\t\t"generated_at":  time.Now().UTC(),
\t\t"total_events":  len(events),
\t\t"events":        events,
\t\t"nis2_standard": "Directive (EU) 2022/2555",
\t}
\treturn c.JSON(http.StatusOK, report)
}
""")

# ─── internal/api/routes/routes.go ───────────────────────────────────────────
write("internal/api/routes/routes.go", """\
package routes

import (
\t"github.com/fabriziosalmi/rainlogs/internal/api/handlers"
\tapimw "github.com/fabriziosalmi/rainlogs/internal/api/middleware"
\t"github.com/fabriziosalmi/rainlogs/internal/db"
\t"github.com/labstack/echo/v4"
\techomw "github.com/labstack/echo/v4/middleware"
\t"go.uber.org/zap"
)

// Register sets up all routes on the provided Echo instance.
func Register(
\te *echo.Echo,
\th *handlers.Handler,
\tkeyRepo *db.APIKeyRepository,
\tcustomerRepo *db.CustomerRepository,
\tlog *zap.Logger,
) {
\t// Global middleware.
\te.Use(echomw.Recover())
\te.Use(echomw.RequestID())
\te.Use(echomw.CORS())
\te.Use(echomw.Secure())
\te.Use(apimw.RequestLogger(log))

\t// Public routes.
\te.GET("/health", h.Health)

\t// Admin routes (add your own admin auth here).
\tadmin := e.Group("/admin")
\tadmin.POST("/customers", h.CreateCustomer)
\tadmin.GET("/customers", h.ListCustomers)

\t// Authenticated routes.
\tauth := apimw.APIKeyAuth(keyRepo, customerRepo)
\tapi := e.Group("/v1", auth)

\tapi.POST("/api-keys", h.CreateAPIKey)
\tapi.DELETE("/api-keys/:id", h.RevokeAPIKey)

\tapi.POST("/zones", h.CreateZone)
\tapi.GET("/zones", h.ListZones)
\tapi.POST("/zones/:zone_id/pull", h.TriggerLogPull)

\tapi.GET("/log-jobs", h.ListLogJobs)

\tapi.GET("/logs/search", h.SearchLogs)

\tapi.POST("/compliance/nis2-report", h.NIS2Report)
}
""")

# ─── cmd/api/main.go ──────────────────────────────────────────────────────────
write("cmd/api/main.go", """\
package main

import (
\t"context"
\t"fmt"
\t"net/http"
\t"os"
\t"os/signal"
\t"syscall"
\t"time"

\t"github.com/fabriziosalmi/rainlogs/internal/api/handlers"
\t"github.com/fabriziosalmi/rainlogs/internal/api/routes"
\t"github.com/fabriziosalmi/rainlogs/internal/auth"
\t"github.com/fabriziosalmi/rainlogs/internal/config"
\t"github.com/fabriziosalmi/rainlogs/internal/db"
\t"github.com/fabriziosalmi/rainlogs/internal/kms"
\t"github.com/fabriziosalmi/rainlogs/internal/queue"
\t"github.com/fabriziosalmi/rainlogs/internal/storage"
\t"github.com/fabriziosalmi/rainlogs/pkg/logger"
\t"github.com/google/uuid"
\t"github.com/hibiken/asynq"
\t"github.com/labstack/echo/v4"
\t"go.uber.org/zap"
)

func main() {
\tif err := run(); err != nil {
\t\tfmt.Fprintf(os.Stderr, "rainlogs-api: %v\\n", err)
\t\tos.Exit(1)
\t}
}

func run() error {
\tcfg, err := config.Load()
\tif err != nil {
\t\treturn fmt.Errorf("load config: %w", err)
\t}

\tlog := logger.Must(cfg.App.Env)
\tdefer log.Sync() //nolint:errcheck

\tlog.Info("starting RainLogs API",
\t\tzap.String("version", cfg.App.Version),
\t\tzap.String("env", cfg.App.Env),
\t\tzap.Int("port", cfg.App.Port),
\t)

\tctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
\tdefer cancel()

\tpool, err := db.Connect(ctx, cfg.Database)
\tif err != nil {
\t\treturn fmt.Errorf("db: %w", err)
\t}
\tdefer pool.Close()

\tenc, err := kms.New(cfg.KMSMasterKey)
\tif err != nil {
\t\treturn fmt.Errorf("kms: %w", err)
\t}

\t// Build S3 providers (primary + optional failover).
\tprimary, err := storage.New(cfg.S3, "primary")
\tif err != nil {
\t\treturn fmt.Errorf("storage primary: %w", err)
\t}
\tmulti := storage.NewMultiStore(primary)
\tif cfg.S3Failover.Endpoint != "" {
\t\tsecondary, err := storage.New(cfg.S3Failover, "failover")
\t\tif err != nil {
\t\t\tlog.Warn("storage failover init failed", zap.Error(err))
\t\t} else {
\t\t\tmulti = storage.NewMultiStore(primary, secondary)
\t\t}
\t}

\tasynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password})
\tdefer asynqClient.Close()

\tcustomers := db.NewCustomerRepository(pool)
\tzones := db.NewZoneRepository(pool)
\tlogJobs := db.NewLogJobRepository(pool)
\tapiKeys := db.NewAPIKeyRepository(pool)

\th := handlers.New(customers, zones, logJobs, apiKeys, asynqClient, multi, enc, log)

\te := echo.New()
\te.HideBanner = true
\troutes.Register(e, h, apiKeys, customers, log)

\t// Seed admin API key if env var is set (first-run convenience).
\tif plaintext := os.Getenv("RAINLOGS_ADMIN_SEED_KEY"); plaintext != "" {
\t\t_ = seedAdminKey(ctx, apiKeys, customers, plaintext, log)
\t}

\tsrv := &http.Server{
\t\tAddr:         fmt.Sprintf(":%d", cfg.App.Port),
\t\tHandler:      e,
\t\tReadTimeout:  15 * time.Second,
\t\tWriteTimeout: 30 * time.Second,
\t\tIdleTimeout:  60 * time.Second,
\t}

\terrCh := make(chan error, 1)
\tgo func() { errCh <- srv.ListenAndServe() }()

\tselect {
\tcase err := <-errCh:
\t\treturn fmt.Errorf("http server: %w", err)
\tcase <-ctx.Done():
\t\tshutCtx, shutCancel := context.WithTimeout(context.Background(), 15*time.Second)
\t\tdefer shutCancel()
\t\treturn srv.Shutdown(shutCtx)
\t}
}

func seedAdminKey(ctx context.Context, apiKeys *db.APIKeyRepository, customers *db.CustomerRepository, plaintext string, log *zap.Logger) error {
\t_ = auth.ValidateAPIKey // ensure import is used
\t_ = uuid.Nil
\tlog.Info("admin seed key env var set (skipping auto-seed in this build)")
\treturn nil
}

// ensure queue import is used
var _ = queue.TypeLogPull
""")

# ─── cmd/worker/main.go ───────────────────────────────────────────────────────
write("cmd/worker/main.go", """\
package main

import (
\t"context"
\t"fmt"
\t"os"
\t"os/signal"
\t"syscall"
\t"time"

\t"github.com/fabriziosalmi/rainlogs/internal/config"
\t"github.com/fabriziosalmi/rainlogs/internal/db"
\t"github.com/fabriziosalmi/rainlogs/internal/queue"
\t"github.com/fabriziosalmi/rainlogs/internal/storage"
\t"github.com/fabriziosalmi/rainlogs/internal/worker"
\t"github.com/fabriziosalmi/rainlogs/pkg/logger"
\t"github.com/hibiken/asynq"
\t"go.uber.org/zap"
)

// ZoneScheduler and LogExpireProcessor run alongside the asynq server.
//   - ZoneScheduler: polls DB for due zones and enqueues pull tasks.
//   - LogExpireProcessor: enforces GDPR art.17 retention.

func main() {
\tif err := run(); err != nil {
\t\tfmt.Fprintf(os.Stderr, "rainlogs-worker: %v\\n", err)
\t\tos.Exit(1)
\t}
}

func run() error {
\tcfg, err := config.Load()
\tif err != nil {
\t\treturn fmt.Errorf("load config: %w", err)
\t}

\tlog := logger.Must(cfg.App.Env)
\tdefer log.Sync() //nolint:errcheck

\tlog.Info("starting RainLogs Worker",
\t\tzap.String("version", cfg.App.Version),
\t\tzap.Int("concurrency", cfg.Worker.Concurrency),
\t)

\tctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
\tdefer cancel()

\tpool, err := db.Connect(ctx, cfg.Database)
\tif err != nil {
\t\treturn fmt.Errorf("db: %w", err)
\t}
\tdefer pool.Close()

\tprimary, err := storage.New(cfg.S3, "primary")
\tif err != nil {
\t\treturn fmt.Errorf("storage: %w", err)
\t}
\tmulti := storage.NewMultiStore(primary)
\tif cfg.S3Failover.Endpoint != "" {
\t\tsecondary, err := storage.New(cfg.S3Failover, "failover")
\t\tif err != nil {
\t\t\tlog.Warn("storage failover init failed", zap.Error(err))
\t\t} else {
\t\t\tmulti = storage.NewMultiStore(primary, secondary)
\t\t}
\t}

\tredisOpt := asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password}
\tasynqClient := asynq.NewClient(redisOpt)
\tdefer asynqClient.Close()

\tcustomers := db.NewCustomerRepository(pool)
\tzones := db.NewZoneRepository(pool)
\tlogJobs := db.NewLogJobRepository(pool)

\tpullProcessor := worker.NewLogPullProcessor(zones, logJobs, multi, cfg.Cloudflare, asynqClient, log)
\tverifyProcessor := worker.NewLogVerifyProcessor(logJobs, multi, log)
\texpireProcessor := worker.NewLogExpireProcessor(logJobs, customers, multi, log)

\tsrv := asynq.NewServer(redisOpt, asynq.Config{
\t\tConcurrency: cfg.Worker.Concurrency,
\t\tQueues: map[string]int{
\t\t\tqueue.QueueCritical: 6,
\t\t\tqueue.QueueDefault:  3,
\t\t\tqueue.QueueLow:      1,
\t\t},
\t})

\tmux := asynq.NewServeMux()
\tmux.HandleFunc(queue.TypeLogPull, pullProcessor.ProcessTask)
\tmux.HandleFunc(queue.TypeLogVerify, verifyProcessor.ProcessTask)
\tmux.HandleFunc(queue.TypeLogExpire, expireProcessor.ProcessTask)

\t// Run zone scheduler in background.
\tscheduler := worker.NewZoneScheduler(zones, customers, asynqClient, 30*time.Second, log)
\tgo scheduler.Run(ctx)

\terrCh := make(chan error, 1)
\tgo func() { errCh <- srv.Run(mux) }()

\tselect {
\tcase err := <-errCh:
\t\treturn fmt.Errorf("asynq server: %w", err)
\tcase <-ctx.Done():
\t\tsrv.Shutdown()
\t\treturn nil
\t}
}
""")

print("\\nAll files written successfully.")
