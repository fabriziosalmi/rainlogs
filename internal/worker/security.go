package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

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
)

type SecurityEventsProcessor struct {
	db      *db.DB
	kms     *kms.Encryptor
	storage *storage.MultiStore
	queue   *asynq.Client
	cfCfg   config.CloudflareConfig
	log     *zap.Logger
}

func NewSecurityEventsProcessor(db *db.DB, kms *kms.Encryptor, storage *storage.MultiStore, queue *asynq.Client, cfCfg config.CloudflareConfig, log *zap.Logger) *SecurityEventsProcessor {
	return &SecurityEventsProcessor{
		db:      db,
		kms:     kms,
		storage: storage,
		queue:   queue,
		cfCfg:   cfCfg,
		log:     log,
	}
}

func (p *SecurityEventsProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	payload, err := queue.ParseSecurityPollPayload(t)
	if err != nil {
		return fmt.Errorf("parse payload: %w", err)
	}

	// 1. Create LogJob (reuse LogJob table, maybe add type field later, for now we assume it's just a job)
	// Ideally we should distinguish job type in DB, but for MVP reuse is ok as ID is unique.
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

	// 4. Fetch Security Events
	cfClient := cloudflare.NewGraphQLClient(cfKey) // Or use existing client if extended? No, separate client for now.
	events, err := cfClient.GetSecurityEvents(ctx, zone.ZoneID, payload.PeriodStart, payload.PeriodEnd)
	if err != nil {
		return p.failJob(ctx, job, fmt.Errorf("fetch security events: %w", err))
	}

	if len(events) == 0 {
		job.Status = models.JobStatusDone
		job.LogCount = 0
		job.ByteCount = 0
		return p.db.LogJobs.Update(ctx, job)
	}

	// 5. Convert to NDJSON
	var buffer []byte
	for _, event := range events {
		line, err := json.Marshal(event)
		if err != nil {
			p.log.Error("failed to marshal event", zap.Error(err))
			continue
		}
		buffer = append(buffer, line...)
		buffer = append(buffer, '\n')
	}

	// 6. Hash & WORM (Same logic as LogPull)
	h := sha256.New()
	h.Write(buffer)
	hashStr := hex.EncodeToString(h.Sum(nil))

	prevJob, err := p.db.LogJobs.GetLastJob(ctx, zone.ID)
	prevChainHash := worm.GenesisHash
	if err == nil && prevJob != nil {
		prevChainHash = prevJob.ChainHash
	}
	chainHash := worm.ChainHash(prevChainHash, hashStr, job.ID.String())

	// 7. Upload to S3
	// Note: PutLogs assumes "access logs" folder structure? Or generic?
	// It uses `customerID/zoneID/year/month/day/...`. This is fine.
	// Maybe we should verify prefix in storage/s3.go?
	s3Key, s3HashStr, provider, byteCount, logCount, err := p.storage.PutLogs(ctx, customer.ID, zone.ID, payload.PeriodStart, payload.PeriodEnd, buffer, "security")
	if err != nil {
		return p.failJob(ctx, job, fmt.Errorf("s3 upload: %w", err))
	}

	// 8. Update Job
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

	return nil
}

func (p *SecurityEventsProcessor) failJob(ctx context.Context, job *models.LogJob, err error) error {
	job.Attempts++
	job.Status = models.JobStatusFailed
	job.ErrMsg = err.Error()
	_ = p.db.LogJobs.Update(ctx, job)
	return err
}
