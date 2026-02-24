package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/fabriziosalmi/rainlogs/internal/config"
	"github.com/fabriziosalmi/rainlogs/internal/db"
	"github.com/fabriziosalmi/rainlogs/internal/kms"
	"github.com/fabriziosalmi/rainlogs/internal/queue"
	"github.com/fabriziosalmi/rainlogs/internal/storage"
	"github.com/fabriziosalmi/rainlogs/internal/worker"
	"github.com/hibiken/asynq"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 1. Init DB
	database, err := db.Connect(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer database.Close()

	// 2. Init KMS
	if cfg.KMS.Key == "" {
		log.Fatal("KMS key is required (RAINLOGS_KMS_KEY)")
	}
	kmsService, err := kms.New(cfg.KMS.Key)
	if err != nil {
		log.Fatalf("failed to init kms: %v", err)
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
		log.Fatalf("unknown storage backend: %s", cfg.Storage.Backend)
	}

	if storeErr != nil {
		log.Fatalf("failed to init storage backend %s: %v", cfg.Storage.Backend, storeErr)
	}

	s3Client := storage.NewMultiStore(backend)

	// 4. Init Queue
	redisOpt, err := asynq.ParseRedisURI(cfg.Redis.Addr)
	if err != nil {
		log.Fatalf("failed to parse redis url: %v", err)
	}

	queueClient := asynq.NewClient(redisOpt)
	defer queueClient.Close()

	// 5. Init Processors
	pullProcessor := worker.NewLogPullProcessor(database, kmsService, s3Client, queueClient)
	verifyProcessor := worker.NewLogVerifyProcessor(database, s3Client)
	expireProcessor := worker.NewLogExpireProcessor(database, s3Client)

	// 6. Start Scheduler
	scheduler := worker.NewZoneScheduler(database, queueClient)
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
			log.Fatalf("could not run server: %v", err)
		}
	}()

	// 8. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down worker...")
	srv.Shutdown()
}
