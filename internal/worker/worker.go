package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
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
)

type LogPullProcessor struct {
	db      *db.DB
	kms     *kms.Encryptor
	storage *storage.MultiStore
	queue   *asynq.Client
	cfCfg   config.CloudflareConfig
}

func NewLogPullProcessor(db *db.DB, kms *kms.Encryptor, storage *storage.MultiStore, queue *asynq.Client, cfCfg config.CloudflareConfig) *LogPullProcessor {
	return &LogPullProcessor{
		db:      db,
		kms:     kms,
		storage: storage,
		queue:   queue,
		cfCfg:   cfCfg,
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
	cfClient := cloudflare.NewClient(p.cfCfg, zone.ZoneID, string(cfKey))
	logs, err := cfClient.PullLogs(ctx, payload.PeriodStart, payload.PeriodEnd, nil)
	if err != nil {
		return p.failJob(ctx, job, fmt.Errorf("pull logs: %w", err))
	}

	if len(logs) == 0 {
		job.Status = models.JobStatusDone
		job.LogCount = 0
		job.ByteCount = 0
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
	s3Key, s3HashStr, provider, byteCount, logCount, err := p.storage.PutLogs(ctx, customer.ID, zone.ID, payload.PeriodStart, payload.PeriodEnd, logs)
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

	// 8. Enqueue Verify Task
	verifyTask, err := queue.NewLogVerifyTask(queue.LogVerifyPayload{JobID: job.ID})
	if err != nil {
		log.Printf("failed to enqueue verify task: %v", err)
	} else {
		if _, err := p.queue.EnqueueContext(ctx, verifyTask); err != nil {
			log.Printf("failed to enqueue verify task: %v", err)
		}
	}

	return nil
}

func (p *LogPullProcessor) failJob(ctx context.Context, job *models.LogJob, err error) error {
	job.Status = models.JobStatusFailed
	job.ErrMsg = err.Error()
	_ = p.db.LogJobs.Update(ctx, job)
	return err
}

type LogVerifyProcessor struct {
	db      *db.DB
	storage *storage.MultiStore
}

func NewLogVerifyProcessor(db *db.DB, storage *storage.MultiStore) *LogVerifyProcessor {
	return &LogVerifyProcessor{
		db:      db,
		storage: storage,
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
		return fmt.Errorf("hash mismatch: expected %s, got %s", job.SHA256, hashStr)
	}

	return nil
}

type LogExpireProcessor struct {
	db      *db.DB
	storage *storage.MultiStore
}

func NewLogExpireProcessor(db *db.DB, storage *storage.MultiStore) *LogExpireProcessor {
	return &LogExpireProcessor{
		db:      db,
		storage: storage,
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
				log.Printf("failed to delete s3 object %s: %v", job.S3Key, err)
				continue
			}
		}
		if err := p.db.LogJobs.MarkExpired(ctx, job.ID); err != nil {
			log.Printf("failed to mark job %s expired: %v", job.ID, err)
		}
	}

	return nil
}

type ZoneScheduler struct {
	db    *db.DB
	queue *asynq.Client
}

func NewZoneScheduler(db *db.DB, queue *asynq.Client) *ZoneScheduler {
	return &ZoneScheduler{
		db:    db,
		queue: queue,
	}
}

func (s *ZoneScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
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
		log.Printf("scheduler: list due zones: %v", err)
		return
	}

	for _, zone := range zones {
		now := time.Now().UTC()
		start := now.Add(-time.Duration(zone.PullIntervalSecs) * time.Second)
		if zone.LastPulledAt != nil {
			start = *zone.LastPulledAt
		}

		task, err := queue.NewLogPullTask(queue.LogPullPayload{
			ZoneID:      zone.ID,
			CustomerID:  zone.CustomerID,
			PeriodStart: start,
			PeriodEnd:   now,
		})
		if err != nil {
			log.Printf("scheduler: create task for zone %s: %v", zone.ID, err)
			continue
		}

		if _, err := s.queue.EnqueueContext(ctx, task); err != nil {
			log.Printf("scheduler: enqueue task for zone %s: %v", zone.ID, err)
			continue
		}

		if err := s.db.Zones.UpdateLastPulled(ctx, zone.ID, now); err != nil {
			log.Printf("scheduler: update last pulled for zone %s: %v", zone.ID, err)
		}
	}
}
