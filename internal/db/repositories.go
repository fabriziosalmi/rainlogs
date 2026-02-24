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
		(id,name,email,cf_account_id,cf_api_key_enc,retention_days,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,now(),now())
		RETURNING created_at,updated_at`
	return r.db.QueryRow(ctx, q,
		c.ID, c.Name, c.Email, c.CFAccountID, c.CFAPIKeyEnc, c.RetentionDays,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Customer, error) {
	const q = `SELECT id,name,email,cf_account_id,cf_api_key_enc,retention_days,created_at,updated_at
		FROM customers WHERE id=$1`
	c := &models.Customer{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&c.ID, &c.Name, &c.Email, &c.CFAccountID, &c.CFAPIKeyEnc, &c.RetentionDays,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("customer get: %w", err)
	}
	return c, nil
}

func (r *CustomerRepository) List(ctx context.Context) ([]*models.Customer, error) {
	const q = `SELECT id,name,email,cf_account_id,cf_api_key_enc,retention_days,created_at,updated_at
		FROM customers ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Customer
	for rows.Next() {
		c := &models.Customer{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.CFAccountID, &c.CFAPIKeyEnc,
			&c.RetentionDays, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ── APIKeyRepository ──────────────────────────────────────────────────────────

type APIKeyRepository struct{ db *pgxpool.Pool }

func NewAPIKeyRepository(db *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(ctx context.Context, k *models.APIKey) error {
	const q = `INSERT INTO api_keys(id,customer_id,prefix,key_hash,label,created_at)
		VALUES($1,$2,$3,$4,$5,now()) RETURNING created_at`
	return r.db.QueryRow(ctx, q, k.ID, k.CustomerID, k.Prefix, k.KeyHash, k.Label).Scan(&k.CreatedAt)
}

func (r *APIKeyRepository) GetByPrefix(ctx context.Context, prefix string) ([]*models.APIKey, error) {
	const q = `SELECT id,customer_id,prefix,key_hash,label,created_at,last_used_at,revoked_at
		FROM api_keys WHERE prefix=$1 AND revoked_at IS NULL`
	rows, err := r.db.Query(ctx, q, prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.APIKey
	for rows.Next() {
		k := &models.APIKey{}
		if err := rows.Scan(&k.ID, &k.CustomerID, &k.Prefix, &k.KeyHash, &k.Label,
			&k.CreatedAt, &k.LastUsedAt, &k.RevokedAt); err != nil {
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

func (r *APIKeyRepository) GetByCustomerAndID(ctx context.Context, customerID, keyID uuid.UUID) (*models.APIKey, error) {
	const q = `SELECT id,customer_id,prefix,key_hash,label,created_at,last_used_at,revoked_at
		FROM api_keys WHERE customer_id=$1 AND id=$2`
	k := &models.APIKey{}
	err := r.db.QueryRow(ctx, q, customerID, keyID).Scan(
		&k.ID, &k.CustomerID, &k.Prefix, &k.KeyHash, &k.Label,
		&k.CreatedAt, &k.LastUsedAt, &k.RevokedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("api_key get: %w", err)
	}
	return k, nil
}

func (r *APIKeyRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]*models.APIKey, error) {
	const q = `SELECT id,customer_id,prefix,key_hash,label,created_at,last_used_at,revoked_at
		FROM api_keys WHERE customer_id=$1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, q, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.APIKey
	for rows.Next() {
		k := &models.APIKey{}
		if err := rows.Scan(&k.ID, &k.CustomerID, &k.Prefix, &k.KeyHash, &k.Label,
			&k.CreatedAt, &k.LastUsedAt, &k.RevokedAt); err != nil {
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
	const q = `INSERT INTO zones(id,customer_id,zone_id,name,pull_interval_secs,active,created_at)
		VALUES($1,$2,$3,$4,$5,$6,now()) RETURNING created_at`
	return r.db.QueryRow(ctx, q,
		z.ID, z.CustomerID, z.ZoneID, z.Name, z.PullIntervalSecs, z.Active,
	).Scan(&z.CreatedAt)
}

func (r *ZoneRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Zone, error) {
	const q = `SELECT id,customer_id,zone_id,name,pull_interval_secs,last_pulled_at,active,created_at
		FROM zones WHERE id=$1`
	z := &models.Zone{}
	err := r.db.QueryRow(ctx, q, id).Scan(&z.ID, &z.CustomerID, &z.ZoneID, &z.Name,
		&z.PullIntervalSecs, &z.LastPulledAt, &z.Active, &z.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("zone get: %w", err)
	}
	return z, nil
}

func (r *ZoneRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]*models.Zone, error) {
	const q = `SELECT id,customer_id,zone_id,name,pull_interval_secs,last_pulled_at,active,created_at
		FROM zones WHERE customer_id=$1`
	return r.scanZones(ctx, q, customerID)
}

func (r *ZoneRepository) ListDue(ctx context.Context) ([]*models.Zone, error) {
	const q = `SELECT id,customer_id,zone_id,name,pull_interval_secs,last_pulled_at,active,created_at
		FROM zones
		WHERE active=true
		  AND (last_pulled_at IS NULL OR
		       last_pulled_at < now() - (pull_interval_secs || ' seconds')::interval)`
	return r.scanZones(ctx, q)
}

func (r *ZoneRepository) UpdateLastPulled(ctx context.Context, id uuid.UUID, t time.Time) error {
	_, err := r.db.Exec(ctx, `UPDATE zones SET last_pulled_at=$2 WHERE id=$1`, id, t)
	return err
}

func (r *ZoneRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM zones WHERE id=$1`, id)
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
		if err := rows.Scan(&z.ID, &z.CustomerID, &z.ZoneID, &z.Name,
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
func (r *LogJobRepository) MarkVerified(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE log_jobs SET verified_at=now(), updated_at=now() WHERE id=$1`,
		id,
	)
	return err
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
