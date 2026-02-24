package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds the full application configuration loaded from env / config file.
type Config struct {
	App        AppConfig        `mapstructure:"app"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Storage    StorageConfig    `mapstructure:"storage"`
	S3         S3Config         `mapstructure:"s3"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Cloudflare CloudflareConfig `mapstructure:"cloudflare"`
	Worker     WorkerConfig     `mapstructure:"worker"`
	KMS        KMSConfig        `mapstructure:"kms"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Env     string `mapstructure:"env"`  // development | production
	Port    int    `mapstructure:"port"` // HTTP API port
	Version string `mapstructure:"version"`
}

type DatabaseConfig struct {
	DSN             string        `mapstructure:"dsn"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type StorageConfig struct {
	Backend string `mapstructure:"backend"` // "s3", "fs", "multi"
	FSRoot  string `mapstructure:"fs_root"` // Root directory for filesystem
}

// S3Config holds credentials for an S3-compatible provider.
// Multiple buckets can be configured via RAINLOGS_S3_PROVIDERS in YAML.
// For simplicity we keep a single default and let storage layer handle routing.
type S3Config struct {
	Endpoint        string `mapstructure:"endpoint"`
	Region          string `mapstructure:"region"`
	Bucket          string `mapstructure:"bucket"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	// ForcePathStyle must be true for Garage / MinIO
	ForcePathStyle bool `mapstructure:"force_path_style"`
	// StorageClass e.g. STANDARD, REDUCED_REDUNDANCY
	StorageClass string `mapstructure:"storage_class"`
}

type JWTConfig struct {
	Secret     string        `mapstructure:"secret"`
	Expiration time.Duration `mapstructure:"expiration"`
}

type CloudflareConfig struct {
	// Base API URL – override for testing
	BaseURL        string        `mapstructure:"base_url"`
	RequestTimeout time.Duration `mapstructure:"request_timeout"`
	// Max log pull window per request (CF limit: 1h)
	MaxWindowSize time.Duration `mapstructure:"max_window_size"`
}

type WorkerConfig struct {
	// How often the scheduler checks for pending pull jobs
	SchedulerInterval time.Duration `mapstructure:"scheduler_interval"`
	// Max concurrent workers processing pull jobs
	Concurrency int `mapstructure:"concurrency"`
	// Retention period for log objects in S3 (e.g. 395 days for NIS2 ~13 months)
	LogRetentionDays int `mapstructure:"log_retention_days"`
}
type KMSConfig struct {
	Key string `mapstructure:"key"`
}

// Load reads configuration from environment variables and optional config file.
// Environment variable prefix: RAINLOGS_
// Example: RAINLOGS_APP_PORT=8080.
func Load() (*Config, error) {
	v := viper.New()

	// ---------- defaults ----------
	v.SetDefault("app.name", "rainlogs")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.version", "0.5.1")
	v.SetDefault("database.dsn", "")

	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "5m")

	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.password", "")

	v.SetDefault("storage.backend", "s3")
	v.SetDefault("storage.fs_root", "./data/logs")

	v.SetDefault("s3.region", "us-east-1")
	v.SetDefault("s3.endpoint", "")
	v.SetDefault("s3.bucket", "")
	v.SetDefault("s3.access_key_id", "")
	v.SetDefault("s3.secret_access_key", "")

	v.SetDefault("s3.force_path_style", true)
	v.SetDefault("s3.storage_class", "STANDARD")

	v.SetDefault("jwt.expiration", "24h")
	v.SetDefault("jwt.secret", "")

	v.SetDefault("cloudflare.base_url", "https://api.cloudflare.com/client/v4")
	v.SetDefault("cloudflare.request_timeout", "30s")
	v.SetDefault("cloudflare.max_window_size", "1h")
	v.SetDefault("kms.key", "")

	v.SetDefault("worker.scheduler_interval", "1m")
	v.SetDefault("worker.concurrency", 10)
	v.SetDefault("worker.log_retention_days", 395) // ~13 months – beyond NIS2 minimum

	// ---------- config file (optional) ----------
	v.SetConfigName("rainlogs")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/rainlogs")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("config: read file: %w", err)
		}
	}

	// ---------- env vars ----------
	v.SetEnvPrefix("RAINLOGS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	return &cfg, nil
}
