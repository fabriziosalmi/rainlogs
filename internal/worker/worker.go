package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fabriziosalmi/rainlogs/internal/cloudflare"
	"github.com/fabriziosalmi/rainlogs/internal/config"
	"github.com/fabriziosalmi/rainlogs/internal/db"
	"github.com/fabriziosalmi/rainlogs/internal/kms"
	"github.com/fabriziosalmi/rainlogs/internal/models"
	"github.com/fabriziosalmi/rainlogs/internal/queue"
	"github.com/fabriziosalmi/rainlogs/internal/storage"
	"github.com/fabriziosalmi/rainlogs/pkg/worm"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type LogPullProcessor struct {
	db      *db.DB
	kms     *kms.Encryptor
	storage *storage.MultiStore
	queue   *asynq.Client
	cfCfg   config.CloudflareConfig
	log     *zap.Logger
	limiter *rate.Limiter
}

func NewLogPullProcessor(db *db.DB, kms *kms.Encryptor, storage *storage.MultiStore, queue *asynq.Client, cfCfg config.CloudflareConfig, log *zap.Logger) *LogPullProcessor {
	var limiter *rate.Limiter
	if cfCfg.RateLimit > 0 {
		limiter = rate.NewLimiter(rate.Limit(cfCfg.RateLimit), 1)
	}
	return &LogPullProcessor{
		db:      db,
		kms:     kms,
		storage: storage,
		queue:   queue,
		cfCfg:   cfCfg,
		log:     log,
		limiter: limiter,
	}
}

func (p *LogPullProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	payload, err := queue.ParseLogPullPayload(t)
	if err != nil {
		return fmt.Errorf("parse payload: %w", err)
	}

	// 1. Create LogJob
	job := &models.LogJob{
		ID:          uuid.New(),
		ZoneID:      payload.ZoneID,
		CustomerID:  payload.CustomerID,
		PeriodStart: payload.PeriodStart,
		PeriodEnd:   payload.PeriodEnd,
		Status:      models.JobStatusPending,
	}
	if err := p.db.LogJobs.Create(ctx, job); err != nil {
		return fmt.Errorf("create job: %w", err)
	}

	// 2. Get Customer & Zone
	customer, err := p.db.Customers.GetByID(ctx, payload.CustomerID)
	if err != nil {
		return p.failJob(ctx, job, fmt.Errorf("get customer: %w", err))
	}
	zone, err := p.db.Zones.GetByID(ctx, payload.ZoneID)
	if err != nil {
		return p.failJob(ctx, job, fmt.Errorf("get zone: %w", err))
	}

	// 3. Decrypt CF API Key
	cfKey, err := p.kms.Decrypt(customer.CFAPIKeyEnc)
	if err != nil {
		return p.failJob(ctx, job, fmt.Errorf("decrypt cf key: %w", err))
	}

	// 4. Pull Logs from Cloudflare
	if p.limiter != nil {
		if err := p.limiter.Wait(ctx); err != nil {
			return p.failJob(ctx, job, fmt.Errorf("rate limiter: %w", err))
		}
	}

	cfClient := cloudflare.NewClient(p.cfCfg, zone.ZoneID, cfKey)
	logs, err := cfClient.PullLogs(ctx, payload.PeriodStart, payload.PeriodEnd, nil)
	if err != nil {
		return p.failJob(ctx, job, fmt.Errorf("pull logs: %w", err))
	}

	if len(logs) == 0 {
		job.Status = models.JobStatusDone
		job.LogCount = 0
		job.ByteCount = 0
		// No S3 upload for empty windows – skip verify task (nothing to verify).
		return p.db.LogJobs.Update(ctx, job)
	}

	// 5. Hash & WORM
	h := sha256.New()
	h.Write(logs)
	hashStr := hex.EncodeToString(h.Sum(nil))

	prevJob, err := p.db.LogJobs.GetLastJob(ctx, zone.ID)
	prevChainHash := worm.GenesisHash
	if err == nil && prevJob != nil {
		prevChainHash = prevJob.ChainHash
	}

	chainHash := worm.ChainHash(prevChainHash, hashStr, job.ID.String())

	// 6. Upload to S3
	s3Key, s3HashStr, provider, byteCount, logCount, err := p.storage.PutLogs(ctx, customer.ID, zone.ID, payload.PeriodStart, payload.PeriodEnd, logs, "logs")
	if err != nil {
		return p.failJob(ctx, job, fmt.Errorf("s3 upload: %w", err))
	}

	// 7. Update Job
	job.Status = models.JobStatusDone
	job.S3Key = s3Key
	job.S3Provider = provider
	job.SHA256 = s3HashStr
	job.ChainHash = chainHash
	job.ByteCount = byteCount
	job.LogCount = logCount
	if err := p.db.LogJobs.Update(ctx, job); err != nil {
		return fmt.Errorf("update job: %w", err)
	}

	// 8. Enqueue Verify Task. Creating the task structure is always expected
	// to succeed; failure is a programming error and must stop processing.
	// Enqueueing may fail transiently (Redis unavailable) – log at ERROR so
	// operators are alerted; the upload is complete and data is not lost.
	verifyTask, err := queue.NewLogVerifyTask(queue.LogVerifyPayload{JobID: job.ID})
	if err != nil {
		return fmt.Errorf("job %s: create verify task: %w", job.ID, err)
	}
	if _, err := p.queue.EnqueueContext(ctx, verifyTask); err != nil {
		p.log.Error("enqueue verify task failed – WORM integrity check deferred",
			zap.String("job_id", job.ID.String()),
			zap.Error(err),
		)
	}

	return nil
}

func (p *LogPullProcessor) failJob(ctx context.Context, job *models.LogJob, err error) error {
	job.Attempts++
	job.Status = models.JobStatusFailed
	job.ErrMsg = err.Error()
	_ = p.db.LogJobs.Update(ctx, job)

	// If the error is 403 Forbidden, it means the feature is not available (likely not Enterprise).
	// We should stop retrying to avoid spamming the logs and the API.
	if strings.Contains(err.Error(), "HTTP 403") {
		p.log.Error("Cloudflare Logpull API not available (requires Enterprise plan). Stopping retry.",
			zap.String("job_id", job.ID.String()),
			zap.Error(err),
		)
		return nil // Return nil to stop retrying
	}

	return err
}

type LogVerifyProcessor struct {
	db      *db.DB
	storage *storage.MultiStore
	log     *zap.Logger
}

func NewLogVerifyProcessor(db *db.DB, storage *storage.MultiStore, log *zap.Logger) *LogVerifyProcessor {
	return &LogVerifyProcessor{
		db:      db,
		storage: storage,
		log:     log,
	}
}

func (p *LogVerifyProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	payload, err := queue.ParseLogVerifyPayload(t)
	if err != nil {
		return fmt.Errorf("parse payload: %w", err)
	}

	job, err := p.db.LogJobs.GetByID(ctx, payload.JobID)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}

	if job.S3Key == "" || job.SHA256 == "" {
		return fmt.Errorf("job missing s3 key or hash")
	}

	data, err := p.storage.GetLogs(ctx, job.S3Key)
	if err != nil {
		return fmt.Errorf("s3 download: %w", err)
	}

	h := sha256.New()
	h.Write(data)

	hashStr := hex.EncodeToString(h.Sum(nil))
	if hashStr != job.SHA256 {
		p.log.Error("WORM integrity violation detected",
			zap.String("job_id", job.ID.String()),
			zap.String("expected_sha256", job.SHA256),
			zap.String("computed_sha256", hashStr),
		)
		return fmt.Errorf("hash mismatch: expected %s, got %s", job.SHA256, hashStr)
	}

	// Stamp verified_at so operators can audit which jobs have been verified.
	if err := p.db.LogJobs.MarkVerified(ctx, job.ID); err != nil {
		p.log.Warn("mark verified failed", zap.String("job_id", job.ID.String()), zap.Error(err))
	}

	return nil
}

type LogExpireProcessor struct {
	db      *db.DB
	storage *storage.MultiStore
	log     *zap.Logger
}

func NewLogExpireProcessor(db *db.DB, storage *storage.MultiStore, log *zap.Logger) *LogExpireProcessor {
	return &LogExpireProcessor{
		db:      db,
		storage: storage,
		log:     log,
	}
}

func (p *LogExpireProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	payload, err := queue.ParseLogExpirePayload(t)
	if err != nil {
		return fmt.Errorf("parse payload: %w", err)
	}

	jobs, err := p.db.LogJobs.ListExpired(ctx, payload.CustomerID, payload.RetentionDays)
	if err != nil {
		return fmt.Errorf("list expired jobs: %w", err)
	}

	for _, job := range jobs {
		if job.S3Key != "" {
			if err := p.storage.DeleteObject(ctx, job.S3Key); err != nil {
				p.log.Error("failed to delete s3 object", zap.String("s3_key", job.S3Key), zap.Error(err))
				continue
			}
		}
		if err := p.db.LogJobs.MarkExpired(ctx, job.ID); err != nil {
			p.log.Error("failed to mark job expired", zap.String("job_id", job.ID.String()), zap.Error(err))
		}
	}

	return nil
}

type ZoneScheduler struct {
	db       *db.DB
	queue    *asynq.Client
	log      *zap.Logger
	interval time.Duration
}

func NewZoneScheduler(db *db.DB, queue *asynq.Client, log *zap.Logger, interval time.Duration) *ZoneScheduler {
	return &ZoneScheduler{
		db:       db,
		queue:    queue,
		log:      log,
		interval: interval,
	}
}

func (s *ZoneScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.schedule(ctx)
		}
	}
}

func (s *ZoneScheduler) schedule(ctx context.Context) {
	zones, err := s.db.Zones.ListDue(ctx)
	if err != nil {
		s.log.Error("scheduler: list due zones", zap.Error(err))
		return
	}

	for _, zone := range zones {
		now := time.Now().UTC()
		start := now.Add(-time.Duration(zone.PullIntervalSecs) * time.Second)
		if zone.LastPulledAt != nil {
			start = *zone.LastPulledAt
		}

		var task *asynq.Task
		var taskID string

		// Dispatch based on plan type
		switch zone.Plan {
		case models.PlanEnterprise:
			// Default LogPull behavior
			t, err := queue.NewLogPullTask(queue.LogPullPayload{
				ZoneID:      zone.ID,
				CustomerID:  zone.CustomerID,
				PeriodStart: start,
				PeriodEnd:   now,
			})
			if err != nil {
				s.log.Error("scheduler: create log pull task", zap.String("zone_id", zone.ID.String()), zap.Error(err))
				continue
			}
			task = t
			taskID = fmt.Sprintf("pull-%s-%d", zone.ID, start.Unix())

		case models.PlanFreePro:
			// Security Events Poller
			t, err := queue.NewSecurityPollTask(queue.SecurityPollPayload{
				ZoneID:      zone.ID,
				CustomerID:  zone.CustomerID,
				PeriodStart: start,
				PeriodEnd:   now,
			})
			if err != nil {
				s.log.Error("scheduler: create security poll task", zap.String("zone_id", zone.ID.String()), zap.Error(err))
				continue
			}
			task = t
			taskID = fmt.Sprintf("sec-%s-%d", zone.ID, start.Unix())

		case models.PlanBusiness:
			// Instant Logs - Handled by daemon, skip scheduler
			// TODO: Implement health check for Instant Logs Daemon
			continue

		default:
			// Default to Enterprise logic if not specified (backward compatibility)
			t, err := queue.NewLogPullTask(queue.LogPullPayload{
				ZoneID:      zone.ID,
				CustomerID:  zone.CustomerID,
				PeriodStart: start,
				PeriodEnd:   now,
			})
			if err != nil {
				s.log.Error("scheduler: create fallback task", zap.String("zone_id", zone.ID.String()), zap.Error(err))
				continue
			}
			task = t
			taskID = fmt.Sprintf("pull-%s-%d", zone.ID, start.Unix())
		}

		_, err = s.queue.EnqueueContext(ctx, task, asynq.TaskID(taskID))
		if err != nil {
			if errors.Is(err, asynq.ErrTaskIDConflict) || errors.Is(err, asynq.ErrDuplicateTask) {
				// Task already queued for this window – safe to skip.
				continue
			}
			s.log.Error("scheduler: enqueue task", zap.String("zone_id", zone.ID.String()), zap.Error(err))
			continue
		}

		if err := s.db.Zones.UpdateLastPulled(ctx, zone.ID, now); err != nil {
			s.log.Error("scheduler: update last pulled", zap.String("zone_id", zone.ID.String()), zap.Error(err))
		}
	}
}
