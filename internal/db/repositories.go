package db

import (
	"context"
	"fmt"
	"time"

	"github.com/fabriziosalmi/rainlogs/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── CustomerRepository ────────────────────────────────────────────────────────

type CustomerRepository struct{ db *pgxpool.Pool }

func NewCustomerRepository(db *pgxpool.Pool) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) Create(ctx context.Context, c *models.Customer) error {
	const q = `INSERT INTO customers
		(id,name,email,cf_account_id,cf_api_key_enc,retention_days,quota_bytes,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,now(),now())
		RETURNING created_at,updated_at`
	return r.db.QueryRow(ctx, q,
		c.ID, c.Name, c.Email, c.CFAccountID, c.CFAPIKeyEnc, c.RetentionDays, c.QuotaBytes,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Customer, error) {
	const q = `SELECT id,name,email,cf_account_id,cf_api_key_enc,retention_days,quota_bytes,created_at,updated_at
		FROM customers WHERE id=$1 AND deleted_at IS NULL`
	c := &models.Customer{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&c.ID, &c.Name, &c.Email, &c.CFAccountID, &c.CFAPIKeyEnc, &c.RetentionDays, &c.QuotaBytes,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("customer get: %w", err)
	}
	return c, nil
}

func (r *CustomerRepository) List(ctx context.Context) ([]*models.Customer, error) {
	const q = `SELECT id,name,email,cf_account_id,cf_api_key_enc,retention_days,quota_bytes,created_at,updated_at
		FROM customers WHERE deleted_at IS NULL ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Customer
	for rows.Next() {
		c := &models.Customer{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.CFAccountID, &c.CFAPIKeyEnc,
			&c.RetentionDays, &c.QuotaBytes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SoftDelete marks a customer as deleted (GDPR Art. 17 – right to erasure).
func (r *CustomerRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE customers SET deleted_at=now(), updated_at=now() WHERE id=$1 AND deleted_at IS NULL`,
		id,
	)
	return err
}

// ── APIKeyRepository ──────────────────────────────────────────────────────────

type APIKeyRepository struct{ db *pgxpool.Pool }

func NewAPIKeyRepository(db *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(ctx context.Context, k *models.APIKey) error {
	const q = `INSERT INTO api_keys(id,customer_id,prefix,key_hash,label,role,expires_at,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,now()) RETURNING created_at`
	// Fallback for role if not set (though DB default handles it, Go struct zero value is "")
	role := k.Role
	if role == "" {
		role = models.RoleAdmin
	}
	return r.db.QueryRow(ctx, q,
		k.ID, k.CustomerID, k.Prefix, k.KeyHash, k.Label, role, k.ExpiresAt,
	).Scan(&k.CreatedAt)
}

func (r *APIKeyRepository) GetByPrefix(ctx context.Context, prefix string) ([]*models.APIKey, error) {
	const q = `SELECT id,customer_id,prefix,key_hash,label,role,created_at,last_used_at,revoked_at,expires_at
		FROM api_keys WHERE prefix=$1 AND revoked_at IS NULL`
	rows, err := r.db.Query(ctx, q, prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.APIKey
	for rows.Next() {
		k := &models.APIKey{}
		if err := rows.Scan(&k.ID, &k.CustomerID, &k.Prefix, &k.KeyHash, &k.Label, &k.Role,
			&k.CreatedAt, &k.LastUsedAt, &k.RevokedAt, &k.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE api_keys SET last_used_at=now() WHERE id=$1`, id)
	return err
}

func (r *APIKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE api_keys SET revoked_at=now() WHERE id=$1`, id)
	return err
}

// RevokeByCustomer revokes all active API keys for a customer (used in GDPR erasure).
func (r *APIKeyRepository) RevokeByCustomer(ctx context.Context, customerID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE api_keys SET revoked_at=now() WHERE customer_id=$1 AND revoked_at IS NULL`,
		customerID,
	)
	return err
}

func (r *APIKeyRepository) GetByCustomerAndID(ctx context.Context, customerID, keyID uuid.UUID) (*models.APIKey, error) {
	const q = `SELECT id,customer_id,prefix,key_hash,label,role,created_at,last_used_at,revoked_at,expires_at
		FROM api_keys WHERE customer_id=$1 AND id=$2`
	k := &models.APIKey{}
	err := r.db.QueryRow(ctx, q, customerID, keyID).Scan(
		&k.ID, &k.CustomerID, &k.Prefix, &k.KeyHash, &k.Label, &k.Role,
		&k.CreatedAt, &k.LastUsedAt, &k.RevokedAt, &k.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("api_key get: %w", err)
	}
	return k, nil
}

func (r *APIKeyRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]*models.APIKey, error) {
	const q = `SELECT id,customer_id,prefix,key_hash,label,role,created_at,last_used_at,revoked_at,expires_at
		FROM api_keys WHERE customer_id=$1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, q, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.APIKey
	for rows.Next() {
		k := &models.APIKey{}
		if err := rows.Scan(&k.ID, &k.CustomerID, &k.Prefix, &k.KeyHash, &k.Label, &k.Role,
			&k.CreatedAt, &k.LastUsedAt, &k.RevokedAt, &k.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// ── ZoneRepository ────────────────────────────────────────────────────────────

type ZoneRepository struct{ db *pgxpool.Pool }

func NewZoneRepository(db *pgxpool.Pool) *ZoneRepository {
	return &ZoneRepository{db: db}
}

func (r *ZoneRepository) Create(ctx context.Context, z *models.Zone) error {
	const q = `INSERT INTO zones(id,customer_id,zone_id,name,plan,pull_interval_secs,last_pulled_at,active,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,now()) RETURNING created_at`
	if z.Plan == "" {
		z.Plan = models.PlanEnterprise
	}
	return r.db.QueryRow(ctx, q,
		z.ID, z.CustomerID, z.ZoneID, z.Name, z.Plan, z.PullIntervalSecs, z.LastPulledAt, z.Active,
	).Scan(&z.CreatedAt)
}

func (r *ZoneRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Zone, error) {
	const q = `SELECT id,customer_id,zone_id,name,plan,pull_interval_secs,last_pulled_at,active,created_at
		FROM zones WHERE id=$1 AND deleted_at IS NULL`
	z := &models.Zone{}
	err := r.db.QueryRow(ctx, q, id).Scan(&z.ID, &z.CustomerID, &z.ZoneID, &z.Name, &z.Plan,
		&z.PullIntervalSecs, &z.LastPulledAt, &z.Active, &z.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("zone get: %w", err)
	}
	return z, nil
}

func (r *ZoneRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]*models.Zone, error) {
	const q = `SELECT id,customer_id,zone_id,name,plan,pull_interval_secs,last_pulled_at,active,created_at
		FROM zones WHERE customer_id=$1 AND deleted_at IS NULL`
	return r.scanZones(ctx, q, customerID)
}

func (r *ZoneRepository) ListDue(ctx context.Context) ([]*models.Zone, error) {
	const q = `SELECT id,customer_id,zone_id,name,plan,pull_interval_secs,last_pulled_at,active,created_at
		FROM zones
		WHERE active=true
		  AND deleted_at IS NULL
		  AND (last_pulled_at IS NULL OR
		       last_pulled_at < now() - (pull_interval_secs || ' seconds')::interval)`
	return r.scanZones(ctx, q)
}

func (r *ZoneRepository) UpdateLastPulled(ctx context.Context, id uuid.UUID, t time.Time) error {
	_, err := r.db.Exec(ctx, `UPDATE zones SET last_pulled_at=$2 WHERE id=$1`, id, t)
	return err
}

// Delete soft-deletes a zone (GDPR Art. 17 – schema already has deleted_at column).
func (r *ZoneRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE zones SET deleted_at=now(), updated_at=now() WHERE id=$1 AND deleted_at IS NULL`,
		id,
	)
	return err
}

// Update patches mutable zone fields. Only supply the new values for name, plan, intervalSecs, active.
func (r *ZoneRepository) Update(ctx context.Context, zoneID, customerID uuid.UUID, name string, plan models.PlanType, intervalSecs int, active bool) error {
	_, err := r.db.Exec(ctx,
		`UPDATE zones SET name=$3, plan=$4, pull_interval_secs=$5, active=$6, updated_at=now()
		 WHERE id=$1 AND customer_id=$2 AND deleted_at IS NULL`,
		zoneID, customerID, name, plan, intervalSecs, active,
	)
	return err
}

// SoftDeleteByCustomer soft-deletes all non-deleted zones owned by a customer.
func (r *ZoneRepository) SoftDeleteByCustomer(ctx context.Context, customerID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE zones SET deleted_at=now(), updated_at=now()
		 WHERE customer_id=$1 AND deleted_at IS NULL`,
		customerID,
	)
	return err
}

func (r *ZoneRepository) scanZones(ctx context.Context, q string, args ...interface{}) ([]*models.Zone, error) {
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Zone
	for rows.Next() {
		z := &models.Zone{}
		if err := rows.Scan(&z.ID, &z.CustomerID, &z.ZoneID, &z.Name, &z.Plan,
			&z.PullIntervalSecs, &z.LastPulledAt, &z.Active, &z.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, z)
	}
	return out, rows.Err()
}

// ── LogJobRepository ──────────────────────────────────────────────────────────

type LogJobRepository struct{ db *pgxpool.Pool }

func NewLogJobRepository(db *pgxpool.Pool) *LogJobRepository {
	return &LogJobRepository{db: db}
}

func (r *LogJobRepository) Create(ctx context.Context, j *models.LogJob) error {
	const q = `INSERT INTO log_jobs
		(id,zone_id,customer_id,period_start,period_end,status,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,now(),now())
		RETURNING created_at,updated_at`
	return r.db.QueryRow(ctx, q,
		j.ID, j.ZoneID, j.CustomerID, j.PeriodStart, j.PeriodEnd, j.Status,
	).Scan(&j.CreatedAt, &j.UpdatedAt)
}

func (r *LogJobRepository) Update(ctx context.Context, j *models.LogJob) error {
	const q = `UPDATE log_jobs SET
		status=$2, s3_key=$3, s3_provider=$4, sha256=$5,
		chain_hash=$6, byte_count=$7, log_count=$8, err_msg=$9,
		attempts=$10, verified_at=$11, updated_at=now()
		WHERE id=$1`
	_, err := r.db.Exec(ctx, q,
		j.ID, j.Status, j.S3Key, j.S3Provider, j.SHA256,
		j.ChainHash, j.ByteCount, j.LogCount, j.ErrMsg, j.Attempts, j.VerifiedAt,
	)
	return err
}

func (r *LogJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.LogJob, error) {
	const q = `SELECT id,zone_id,customer_id,period_start,period_end,status,
			s3_key,s3_provider,sha256,chain_hash,byte_count,log_count,attempts,err_msg,verified_at,created_at,updated_at
		FROM log_jobs WHERE id=$1`
	j := &models.LogJob{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&j.ID, &j.ZoneID, &j.CustomerID, &j.PeriodStart, &j.PeriodEnd, &j.Status,
		&j.S3Key, &j.S3Provider, &j.SHA256, &j.ChainHash, &j.ByteCount, &j.LogCount,
		&j.Attempts, &j.ErrMsg, &j.VerifiedAt, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("log_job get: %w", err)
	}
	return j, nil
}

func (r *LogJobRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*models.LogJob, error) {
	const q = `SELECT id,zone_id,customer_id,period_start,period_end,status,
			s3_key,s3_provider,sha256,chain_hash,byte_count,log_count,attempts,err_msg,verified_at,created_at,updated_at
		FROM log_jobs WHERE customer_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	return r.scanJobs(ctx, q, customerID, limit, offset)
}

// ListExpired returns done jobs older than retentionDays (GDPR art.17).
func (r *LogJobRepository) ListExpired(ctx context.Context, customerID uuid.UUID, retentionDays int) ([]*models.LogJob, error) {
	const q = `SELECT id,zone_id,customer_id,period_start,period_end,status,
			s3_key,s3_provider,sha256,chain_hash,byte_count,log_count,attempts,err_msg,verified_at,created_at,updated_at
		FROM log_jobs
		WHERE customer_id=$1
		  AND status=$2
		  AND period_end < now() - ($3 || ' days')::interval`
	return r.scanJobs(ctx, q, customerID, models.JobStatusDone, retentionDays)
}

// ListByZone returns jobs for a specific zone owned by customerID.
func (r *LogJobRepository) ListByZone(ctx context.Context, customerID, zoneID uuid.UUID, limit, offset int) ([]*models.LogJob, error) {
	const q = `SELECT id,zone_id,customer_id,period_start,period_end,status,
			s3_key,s3_provider,sha256,chain_hash,byte_count,log_count,attempts,err_msg,verified_at,created_at,updated_at
		FROM log_jobs WHERE customer_id=$1 AND zone_id=$2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`
	return r.scanJobs(ctx, q, customerID, zoneID, limit, offset)
}

// MarkExpired sets a job's status to expired after S3 object deletion.
func (r *LogJobRepository) MarkExpired(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE log_jobs SET status=$2, updated_at=now() WHERE id=$1`,
		id, models.JobStatusExpired,
	)
	return err
}

// MarkVerified stamps verified_at = NOW() on a successfully integrity-checked job.
// GetCurrentUsage returns the total byte count for done jobs in the current month.
func (r *LogJobRepository) GetCurrentUsage(ctx context.Context, customerID uuid.UUID) (int64, error) {
	const q = `
		SELECT COALESCE(SUM(byte_count), 0)
		FROM log_jobs
		WHERE customer_id=$1
		  AND status='done'
		  AND created_at >= date_trunc('month', now())
	`
	var usage int64
	err := r.db.QueryRow(ctx, q, customerID).Scan(&usage)
	return usage, err
}

func (r *LogJobRepository) MarkVerified(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE log_jobs SET verified_at=now(), updated_at=now() WHERE id=$1`,
		id,
	)
	return err
}

func (r *LogJobRepository) ListForExport(ctx context.Context, customerID uuid.UUID, start, end time.Time) ([]*models.LogJob, error) {
	const q = `SELECT id,zone_id,customer_id,period_start,period_end,status,
			s3_key,s3_provider,sha256,chain_hash,byte_count,log_count,attempts,err_msg,verified_at,created_at,updated_at
		FROM log_jobs
		WHERE customer_id=$1
		  AND status='done'
		  AND period_start >= $2 AND period_end <= $3`
	return r.scanJobs(ctx, q, customerID, start, end)
}

func (r *LogJobRepository) scanJobs(ctx context.Context, q string, args ...interface{}) ([]*models.LogJob, error) {
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.LogJob
	for rows.Next() {
		j := &models.LogJob{}
		if err := rows.Scan(&j.ID, &j.ZoneID, &j.CustomerID, &j.PeriodStart, &j.PeriodEnd,
			&j.Status, &j.S3Key, &j.S3Provider, &j.SHA256, &j.ChainHash, &j.ByteCount,
			&j.LogCount, &j.Attempts, &j.ErrMsg, &j.VerifiedAt, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// ── LogObjectRepository ───────────────────────────────────────────────────────

type LogObjectRepository struct{ db *pgxpool.Pool }

func NewLogObjectRepository(db *pgxpool.Pool) *LogObjectRepository {
	return &LogObjectRepository{db: db}
}

func (r *LogObjectRepository) Create(ctx context.Context, o *models.LogObject) error {
	const q = `INSERT INTO log_objects(id,job_id,s3_key,sha256,byte_count,created_at)
		VALUES($1,$2,$3,$4,$5,now()) RETURNING created_at`
	return r.db.QueryRow(ctx, q, o.ID, o.JobID, o.S3Key, o.SHA256, o.ByteCount).Scan(&o.CreatedAt)
}

func (r *LogJobRepository) GetLastJob(ctx context.Context, zoneID uuid.UUID) (*models.LogJob, error) {
	const q = `SELECT id,zone_id,customer_id,period_start,period_end,status,
			s3_key,s3_provider,sha256,chain_hash,byte_count,log_count,attempts,err_msg,verified_at,created_at,updated_at
		FROM log_jobs WHERE zone_id=$1 AND status='done' ORDER BY created_at DESC, id DESC LIMIT 1`
	j := &models.LogJob{}
	err := r.db.QueryRow(ctx, q, zoneID).Scan(
		&j.ID, &j.ZoneID, &j.CustomerID, &j.PeriodStart, &j.PeriodEnd, &j.Status,
		&j.S3Key, &j.S3Provider, &j.SHA256, &j.ChainHash, &j.ByteCount, &j.LogCount,
		&j.Attempts, &j.ErrMsg, &j.VerifiedAt, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return j, nil
}

// ── AuditEventRepository ─────────────────────────────────────────────────────

type AuditEventRepository struct{ db *pgxpool.Pool }

func NewAuditEventRepository(db *pgxpool.Pool) *AuditEventRepository {
	return &AuditEventRepository{db: db}
}

// Create inserts an audit event. Nullable string fields are stored as SQL NULL when empty.
func (r *AuditEventRepository) Create(ctx context.Context, e *models.AuditEvent) error {
	const q = `INSERT INTO audit_events
		(id,customer_id,request_id,action,resource_id,ip_address,status_code,error_detail)
		VALUES($1,$2,$3,$4,NULLIF($5,''),$6,$7,NULLIF($8,''))`
	_, err := r.db.Exec(ctx, q,
		e.ID, e.CustomerID, e.RequestID, e.Action, e.ResourceID,
		e.IPAddress, e.StatusCode, e.ErrorDetail,
	)
	return err
}

// ListByCustomer returns the most recent audit events for a customer.
func (r *AuditEventRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*models.AuditEvent, error) {
	const q = `SELECT id,customer_id,request_id,action,
			COALESCE(resource_id,''),ip_address,status_code,
			COALESCE(error_detail,''),created_at
		FROM audit_events WHERE customer_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, q, customerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.AuditEvent
	for rows.Next() {
		e := &models.AuditEvent{}
		if err := rows.Scan(&e.ID, &e.CustomerID, &e.RequestID, &e.Action,
			&e.ResourceID, &e.IPAddress, &e.StatusCode, &e.ErrorDetail, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListActive returns all active zones.
func (r *ZoneRepository) ListActive(ctx context.Context) ([]*models.Zone, error) {
	const q = `SELECT id,customer_id,zone_id,name,plan,pull_interval_secs,last_pulled_at,active,created_at
		FROM zones
		WHERE active = true AND deleted_at IS NULL`
	return r.scanZones(ctx, q)
}

// ── LogExportRepository ───────────────────────────────────────────────────────

type LogExportRepository struct{ db *pgxpool.Pool }

func NewLogExportRepository(db *pgxpool.Pool) *LogExportRepository {
	return &LogExportRepository{db: db}
}

func (r *LogExportRepository) Create(ctx context.Context, e *models.LogExport) error {
	const q = `INSERT INTO log_exports(id,customer_id,s3_config_enc,filter_start,filter_end,status,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,now(),now())
		RETURNING created_at,updated_at`
	return r.db.QueryRow(ctx, q, e.ID, e.CustomerID, e.S3ConfigEnc, e.FilterStart, e.FilterEnd, e.Status).Scan(&e.CreatedAt, &e.UpdatedAt)
}

func (r *LogExportRepository) Update(ctx context.Context, e *models.LogExport) error {
	const q = `UPDATE log_exports SET status=$2, log_count=$3, byte_count=$4, error_msg=$5, updated_at=now() WHERE id=$1`
	_, err := r.db.Exec(ctx, q, e.ID, e.Status, e.LogCount, e.ByteCount, e.ErrorMsg)
	return err
}

func (r *LogExportRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.LogExport, error) {
	const q = `SELECT id,customer_id,s3_config_enc,filter_start,filter_end,status,log_count,byte_count,error_msg,created_at,updated_at
		FROM log_exports WHERE id=$1`
	e := &models.LogExport{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&e.ID, &e.CustomerID, &e.S3ConfigEnc, &e.FilterStart, &e.FilterEnd,
		&e.Status, &e.LogCount, &e.ByteCount, &e.ErrorMsg, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("log_export get: %w", err)
	}
	return e, nil
}
