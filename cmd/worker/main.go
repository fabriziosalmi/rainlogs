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

	"github.com/fabriziosalmi/rainlogs/internal/config"
	"github.com/fabriziosalmi/rainlogs/internal/db"
	"github.com/fabriziosalmi/rainlogs/internal/kms"
	"github.com/fabriziosalmi/rainlogs/internal/queue"
	"github.com/fabriziosalmi/rainlogs/internal/storage"
	"github.com/fabriziosalmi/rainlogs/internal/worker"
	"github.com/fabriziosalmi/rainlogs/pkg/logger"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	cfg, err := config.Load()
	if err != nil {
		cancel()
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.Must(cfg.App.Env)
	defer log.Sync() //nolint:errcheck // zap.Logger.Sync error is safe to ignore in main

	// 1. Init DB
	database, err := db.Connect(ctx, cfg.Database)
	if err != nil {
		cancel()
		log.Fatal("failed to connect to db", zap.Error(err))
	}
	defer database.Close()

	// 2. Init KMS
	if cfg.KMS.Key == "" {
		cancel()
		log.Fatal("KMS key is required (RAINLOGS_KMS_KEY)")
	}
	kmsService, err := kms.New(cfg.KMS.Key)
	if err != nil {
		cancel()
		log.Fatal("failed to init kms", zap.Error(err))
	}

	// 3. Init Storage
	var backend storage.Backend
	var storeErr error

	switch cfg.Storage.Backend {
	case "s3":
		backend, storeErr = storage.New(ctx, cfg.S3, "s3-default")
	case "fs":
		backend, storeErr = storage.NewFSStore(cfg.Storage.FSRoot)
	default:
		cancel()
		log.Fatal("unknown storage backend", zap.String("backend", cfg.Storage.Backend))
	}

	if storeErr != nil {
		cancel()
		log.Fatal("failed to init storage", zap.String("backend", cfg.Storage.Backend), zap.Error(storeErr))
	}

	s3Client := storage.NewMultiStore(backend)

	// 4. Init Queue
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	queueClient := asynq.NewClient(redisOpt)
	defer queueClient.Close()

	// 5. Init Processors
	pullProcessor := worker.NewLogPullProcessor(database, kmsService, s3Client, queueClient, cfg.Cloudflare, log)
	verifyProcessor := worker.NewLogVerifyProcessor(database, s3Client, log)
	expireProcessor := worker.NewLogExpireProcessor(database, s3Client, log)

	// 6. Start Scheduler
	scheduler := worker.NewZoneScheduler(database, queueClient, log)
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
	mux.HandleFunc(queue.TypeLogVerify, verifyProcessor.ProcessTask)
	mux.HandleFunc(queue.TypeLogExpire, expireProcessor.ProcessTask)

	go func() {
		if err := srv.Run(mux); err != nil {
			log.Fatal("worker server stopped", zap.Error(err))
		}
	}()

	// 8. Minimal HTTP health endpoint for Docker/K8s liveness probes.
	inspector := asynq.NewInspector(redisOpt)
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
		healthSrv := &http.Server{
			Addr:         ":8081",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("worker health server stopped", zap.Error(err))
		}
	}()

	log.Info("worker started", zap.String("env", cfg.App.Env))

	// 9. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down worker...")
	cancel()
	srv.Shutdown()
}
