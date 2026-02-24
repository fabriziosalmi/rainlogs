package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/fabriziosalmi/rainlogs/internal/config"
	"github.com/fabriziosalmi/rainlogs/internal/db"
	"github.com/fabriziosalmi/rainlogs/internal/kms"
	"github.com/fabriziosalmi/rainlogs/internal/notifications"
	"github.com/fabriziosalmi/rainlogs/internal/queue"
	"github.com/fabriziosalmi/rainlogs/internal/storage"
	"github.com/fabriziosalmi/rainlogs/internal/worker"
	"github.com/fabriziosalmi/rainlogs/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "application error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	appLog := logger.Must(cfg.App.Env)
	defer appLog.Sync() //nolint:errcheck // zap.Logger.Sync error is safe to ignore

	// 1. Init DB
	database, err := db.Connect(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %w", err)
	}
	defer database.Close()

	// 2. Init KMS
	kmsService, err := kms.NewKeyRing(cfg.KMS.Keys, cfg.KMS.ActiveKey)
	if err != nil {
		return fmt.Errorf("failed to init kms: %w", err)
	}

	// 3. Init Storage
	var backends []storage.Backend

	switch cfg.Storage.Backend {
	case "s3", "multi": // multi is implicit for s3
		// Primary
		primaryID := cfg.S3.Name
		if primaryID == "" {
			primaryID = "s3-primary"
		}
		primary, err := storage.New(ctx, cfg.S3, primaryID)
		if err != nil {
			return fmt.Errorf("failed to init primary storage: %w", err)
		}
		backends = append(backends, primary)

		// Secondary (Failover)
		if cfg.S3Secondary.Bucket != "" {
			secondaryID := cfg.S3Secondary.Name
			if secondaryID == "" {
				secondaryID = "s3-secondary"
			}
			secondary, err := storage.New(ctx, cfg.S3Secondary, secondaryID)
			if err != nil {
				// Don't fail hard if secondary is bad, but warn log
				appLog.Error("failed to init secondary storage", zap.Error(err))
			} else {
				backends = append(backends, secondary)
				appLog.Info("enabled secondary storage failover", zap.String("provider", secondaryID))
			}
		}

	case "fs":
		bst, err := storage.NewFSStore(cfg.Storage.FSRoot)
		if err != nil {
			return fmt.Errorf("failed to init fs storage: %w", err)
		}
		backends = append(backends, bst)

	default:
		return fmt.Errorf("unknown storage backend: %s", cfg.Storage.Backend)
	}

	s3Client := storage.NewMultiStore(backends...)

	// 4. Init Queue
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	queueClient := asynq.NewClient(redisOpt)
	defer queueClient.Close()

	// 5. Init Notifier
	notifier := &notifications.ConsoleNotifier{}

	// 6. Init Processors
	pullProcessor := worker.NewLogPullProcessor(database, kmsService, s3Client, queueClient, cfg.Cloudflare, appLog, notifier)
	securityProcessor := worker.NewSecurityEventsProcessor(database, kmsService, s3Client, queueClient, cfg.Cloudflare, appLog, notifier)
	verifyProcessor := worker.NewLogVerifyProcessor(database, s3Client, appLog)
	expireProcessor := worker.NewLogExpireProcessor(database.LogJobs, s3Client, appLog)
	exportProcessor := worker.NewLogExportProcessor(database, kmsService, s3Client, appLog, notifier)

	// 6b. Init Instant Logs Daemon
	instantLogsManager := worker.NewInstantLogsManager(database, kmsService, s3Client, cfg.Cloudflare, appLog, notifier)
	go instantLogsManager.Start(ctx)

	// 7. Start Scheduler
	scheduler := worker.NewZoneScheduler(database, queueClient, appLog, cfg.Worker.SchedulerInterval)
	go scheduler.Run(ctx)

	// 7. Start Worker Server
	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: cfg.Worker.Concurrency,
			Queues: map[string]int{
				queue.QueueCritical: 6,
				queue.QueueDefault:  3,
				queue.QueueLow:      1,
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(queue.TypeLogPull, pullProcessor.ProcessTask)
	mux.HandleFunc(queue.TypeSecurityPoll, securityProcessor.ProcessTask)
	mux.HandleFunc(queue.TypeLogVerify, verifyProcessor.ProcessTask)
	mux.HandleFunc(queue.TypeLogExport, exportProcessor.ProcessTask)
	mux.HandleFunc(queue.TypeLogExpire, expireProcessor.ProcessTask)

	errChan := make(chan error, 1)

	go func() {
		if err := srv.Run(mux); err != nil {
			errChan <- fmt.Errorf("worker server failed: %w", err)
		}
	}()

	// 8. Minimal HTTP health endpoint for Docker/K8s liveness probes.
	inspector := asynq.NewInspector(redisOpt)
	healthSrv := &http.Server{
		Addr:         ":8081",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		http.HandleFunc("/health/worker", func(w http.ResponseWriter, _ *http.Request) {
			type queueInfo struct {
				Size int `json:"size"`
			}
			type resp struct {
				Status string               `json:"status"`
				Queues map[string]queueInfo `json:"queues"`
			}
			queues := map[string]queueInfo{}
			overall := "ok"
			for _, qName := range []string{queue.QueueCritical, queue.QueueDefault, queue.QueueLow} {
				info, err := inspector.GetQueueInfo(qName)
				if err != nil {
					overall = "degraded"
					queues[qName] = queueInfo{}
					continue
				}
				queues[qName] = queueInfo{Size: info.Size}
			}
			w.Header().Set("Content-Type", "application/json")
			if overall != "ok" {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			_ = json.NewEncoder(w).Encode(resp{Status: overall, Queues: queues})
		})

		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLog.Error("worker health server stopped", zap.Error(err))
		}
	}()

	appLog.Info("worker started", zap.String("env", cfg.App.Env))

	// 9. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		appLog.Info("shutting down worker...")
	case err := <-errChan:
		appLog.Error("worker server crashed", zap.Error(err))
		cancel() // Cancel context to stop other components
		return err
	}

	// Cleanup happens here (defers) including logging sync
	// Stop instant logs and scheduler via context cancel
	cancel()

	// Wait for cleanup if needed? `defer` handles DB/Queue Close.

	// Graceful shutdown of HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		appLog.Error("health server shutdown error", zap.Error(err))
	}

	return nil
}
