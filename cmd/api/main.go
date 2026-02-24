package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	apimw "github.com/fabriziosalmi/rainlogs/internal/api/middleware"
	"github.com/fabriziosalmi/rainlogs/internal/api/routes"
	"github.com/fabriziosalmi/rainlogs/internal/config"
	"github.com/fabriziosalmi/rainlogs/internal/db"
	"github.com/fabriziosalmi/rainlogs/internal/kms"
	"github.com/fabriziosalmi/rainlogs/internal/storage"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
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

	// 3. Init Storage (for log download)
	var backend storage.Backend
	switch cfg.Storage.Backend {
	case "s3":
		backend, err = storage.New(ctx, cfg.S3, "s3-default")
	case "fs":
		backend, err = storage.NewFSStore(cfg.Storage.FSRoot)
	default:
		log.Fatalf("unknown storage backend: %s", cfg.Storage.Backend)
	}
	if err != nil {
		log.Fatalf("failed to init storage: %v", err)
	}
	multiStore := storage.NewMultiStore(backend)

	// 4. Init Queue client (for trigger-pull)
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	queueClient := asynq.NewClient(redisOpt)
	defer queueClient.Close()

	// 5. Init Echo
	e := echo.New()
	e.HideBanner = true

	// Global middleware
	e.Use(apimw.RequestID())
	e.Use(apimw.SecurityHeaders())
	e.Use(echomw.Logger())
	e.Use(echomw.Recover())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderAuthorization, echo.HeaderContentType, echo.HeaderAccept},
		MaxAge:       3600,
	}))
	e.Use(echomw.GzipWithConfig(echomw.GzipConfig{Level: 5}))
	// 60 req/s per IP, burst of 120
	e.Use(apimw.RateLimit(60, 120))

	jwtSecret := cfg.JWT.Secret
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	// 6. Register Routes
	routes.Register(e, database, kmsService, jwtSecret, queueClient, multiStore)

	// 7. Enhanced health check
	e.GET("/health", func(c echo.Context) error {
		type depStatus struct {
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		}
		type healthResponse struct {
			Status  string               `json:"status"`
			Version string               `json:"version"`
			Deps    map[string]depStatus `json:"deps"`
		}

		deps := make(map[string]depStatus)
		overall := "ok"

		pingCtx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()
		if err := database.Pool.Ping(pingCtx); err != nil {
			deps["postgres"] = depStatus{Status: "error", Error: err.Error()}
			overall = "degraded"
		} else {
			deps["postgres"] = depStatus{Status: "ok"}
		}

		conn, dialErr := (&net.Dialer{}).DialContext(pingCtx, "tcp", cfg.Redis.Addr)
		if dialErr != nil {
			deps["redis"] = depStatus{Status: "error", Error: dialErr.Error()}
			overall = "degraded"
		} else {
			conn.Close()
			deps["redis"] = depStatus{Status: "ok"}
		}

		status := http.StatusOK
		if overall != "ok" {
			status = http.StatusServiceUnavailable
		}

		return c.JSON(status, healthResponse{
			Status:  overall,
			Version: cfg.App.Version,
			Deps:    deps,
		})
	})

	// 8. Start Server
	go func() {
		port := strconv.Itoa(cfg.App.Port)
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			log.Printf("server stopped: %v", err)
		}
	}()

	// 9. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := e.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
}
