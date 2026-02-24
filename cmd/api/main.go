package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/fabriziosalmi/rainlogs/pkg/logger"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.Must(cfg.App.Env)
	defer log.Sync() //nolint:errcheck

	// 1. Init DB
	database, err := db.Connect(ctx, cfg.Database)
	if err != nil {
		log.Fatal("failed to connect to db", zap.Error(err))
	}
	defer database.Close()

	// 2. Init KMS
	if cfg.KMS.Key == "" {
		log.Fatal("KMS key is required (RAINLOGS_KMS_KEY)")
	}
	kmsService, err := kms.New(cfg.KMS.Key)
	if err != nil {
		log.Fatal("failed to init kms", zap.Error(err))
	}

	// 3. Init Storage (for log download)
	var backend storage.Backend
	switch cfg.Storage.Backend {
	case "s3":
		backend, err = storage.New(ctx, cfg.S3, "s3-default")
	case "fs":
		backend, err = storage.NewFSStore(cfg.Storage.FSRoot)
	default:
		log.Fatal("unknown storage backend", zap.String("backend", cfg.Storage.Backend))
	}
	if err != nil {
		log.Fatal("failed to init storage", zap.String("backend", cfg.Storage.Backend), zap.Error(err))
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
		AllowMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPatch,
			http.MethodDelete, http.MethodOptions,
		},
		AllowHeaders: []string{echo.HeaderAuthorization, echo.HeaderContentType, echo.HeaderAccept},
		MaxAge:       3600,
	}))
	e.Use(echomw.GzipWithConfig(echomw.GzipConfig{Level: 5}))
	// 60 req/s per IP, burst of 120 (global guard before auth)
	e.Use(apimw.RateLimit(60, 120))

	// C1: Prometheus metrics (unauthenticated – standard SRE convention)
	e.Use(echoprometheus.NewMiddleware("rainlogs"))
	e.GET("/metrics", echoprometheus.NewHandler())

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

		pingCtx, pingCancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer pingCancel()
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

		b, _ := json.Marshal(healthResponse{
			Status:  overall,
			Version: cfg.App.Version,
			Deps:    deps,
		})
		return c.JSONBlob(status, b)
	})

	// 8. Start Server
	go func() {
		port := strconv.Itoa(cfg.App.Port)
		log.Info("API server starting", zap.String("port", port), zap.String("env", cfg.App.Env))
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			log.Error("server stopped", zap.Error(err))
		}
	}()

	// 9. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down API server...")
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := e.Shutdown(ctxShutdown); err != nil {
		log.Fatal("server forced to shutdown", zap.Error(err))
	}
}
