package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fabriziosalmi/rainlogs/internal/db"
	"github.com/fabriziosalmi/rainlogs/internal/kms"
	"github.com/fabriziosalmi/rainlogs/internal/models"
	"github.com/fabriziosalmi/rainlogs/internal/notifications"
	"github.com/fabriziosalmi/rainlogs/internal/queue"
	"github.com/fabriziosalmi/rainlogs/internal/storage"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type LogExportProcessor struct {
	db       *db.DB
	kms      *kms.Encryptor
	storage  *storage.MultiStore // Our internal storage
	log      *zap.Logger
	notifier notifications.NotificationService
}

func NewLogExportProcessor(db *db.DB, kms *kms.Encryptor, storage *storage.MultiStore, log *zap.Logger, notifier notifications.NotificationService) *LogExportProcessor {
	return &LogExportProcessor{
		db:       db,
		kms:      kms,
		storage:  storage,
		log:      log,
		notifier: notifier,
	}
}

func (p *LogExportProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	payload, err := queue.ParseLogExportPayload(t)
	if err != nil {
		return fmt.Errorf("parse payload: %w", err)
	}

	// 1. Get Export Job
	exportJob, err := p.db.LogExports.GetByID(ctx, payload.ExportID)
	if err != nil {
		return fmt.Errorf("get export job: %w", err)
	}

	// Update status to processing
	exportJob.Status = models.ExportStatusProcessing
	if err := p.db.LogExports.Update(ctx, exportJob); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	// 2. Decrypt S3 Config
	jsonConfig, err := p.kms.Decrypt(exportJob.S3ConfigEnc)
	if err != nil {
		return p.failJob(ctx, exportJob, fmt.Errorf("decrypt s3 config: %w", err))
	}

	var s3Cfg models.ExportS3Config
	if err := json.Unmarshal([]byte(jsonConfig), &s3Cfg); err != nil {
		return p.failJob(ctx, exportJob, fmt.Errorf("unmarshal s3 config: %w", err))
	}

	// 3. Init Destination S3 Client
	destClient := s3.New(s3.Options{
		Region:       s3Cfg.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(s3Cfg.AccessKeyID, s3Cfg.SecretAccessKey, ""),
		BaseEndpoint: aws.String(s3Cfg.Endpoint),
		UsePathStyle: true, // Assuming compatible implementations often need this
	})

	// 4. List Log Jobs to Export
	logs, err := p.db.LogJobs.ListForExport(ctx, exportJob.CustomerID, exportJob.FilterStart, exportJob.FilterEnd)
	if err != nil {
		return p.failJob(ctx, exportJob, fmt.Errorf("list logs: %w", err))
	}

	// 5. Export Loop
	var successCount, byteCount int64
	for _, logJob := range logs {
		// a. Read from RainLogs storage
		data, err := p.storage.GetLogs(ctx, logJob.S3Key)
		if err != nil {
			p.log.Error("failed to get log object", zap.String("key", logJob.S3Key), zap.Error(err))
			continue // Skip corrupted/missing logs? Or fail hard? Skipping for now.
		}

		// b. Write to Destination
		destKey := filepath.Join(s3Cfg.PathPrefix, logJob.PeriodStart.Format("2006/01/02"), fmt.Sprintf("%s.log.gz", logJob.ID))
		_, err = destClient.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(s3Cfg.Bucket),
			Key:    aws.String(destKey),
			Body:   bytes.NewReader(data),
		})
		if err != nil {
			p.log.Error("failed to upload to destination", zap.String("dest_key", destKey), zap.Error(err))
			// Retry?
			continue
		}

		successCount++
		byteCount += int64(len(data))
	}

	// 6. Complete
	exportJob.Status = models.ExportStatusCompleted
	exportJob.LogCount = successCount
	exportJob.ByteCount = byteCount
	if err := p.db.LogExports.Update(ctx, exportJob); err != nil {
		return fmt.Errorf("update complete: %w", err)
	}

	p.notifier.SendAlert(ctx, exportJob.CustomerID.String(), "info", fmt.Sprintf("Bulk export completed: %d files uploaded", successCount))
	return nil
}

func (p *LogExportProcessor) failJob(ctx context.Context, job *models.LogExport, err error) error {
	msg := err.Error()
	job.Status = models.ExportStatusFailed
	job.ErrorMsg = &msg
	_ = p.db.LogExports.Update(ctx, job)
	p.notifier.SendAlert(ctx, job.CustomerID.String(), "error", fmt.Sprintf("Bulk export failed: %v", err))
	return err
}
