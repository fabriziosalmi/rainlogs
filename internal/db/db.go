package db

import (
	"context"
	"fmt"
	"time"

	"github.com/fabriziosalmi/rainlogs/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool        *pgxpool.Pool
	Customers   *CustomerRepository
	APIKeys     *APIKeyRepository
	Zones       *ZoneRepository
	LogJobs     *LogJobRepository
	LogObjects  *LogObjectRepository
	AuditEvents *AuditEventRepository
	LogExports  *LogExportRepository
}

// Connect returns a pgxpool.Pool configured from cfg.
func Connect(ctx context.Context, cfg config.DatabaseConfig) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("db: parse dsn: %w", err)
	}
	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.ConnMaxLifetime
	poolCfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("db: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db: ping: %w", err)
	}

	return &DB{
		Pool:        pool,
		Customers:   NewCustomerRepository(pool),
		APIKeys:     NewAPIKeyRepository(pool),
		Zones:       NewZoneRepository(pool),
		LogJobs:     NewLogJobRepository(pool),
		LogObjects:  NewLogObjectRepository(pool),
		AuditEvents: NewAuditEventRepository(pool),
		LogExports:  NewLogExportRepository(pool),
	}, nil
}

func (db *DB) Close() {
	db.Pool.Close()
}
