package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/fabriziosalmi/rainlogs/internal/api/routes"
	"github.com/fabriziosalmi/rainlogs/internal/config"
	"github.com/fabriziosalmi/rainlogs/internal/db"
	"github.com/fabriziosalmi/rainlogs/internal/kms"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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

	// 3. Init Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	jwtSecret := cfg.JWT.Secret
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	// 4. Register Routes
	routes.Register(e, database, kmsService, jwtSecret)

	// 5. Start Server
	go func() {
		port := strconv.Itoa(cfg.App.Port)
		if err := e.Start(":" + port); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	// 6. Graceful Shutdown
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
