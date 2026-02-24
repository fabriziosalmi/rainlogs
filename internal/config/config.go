package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds the full application configuration loaded from env / config file.
type Config struct {
	App           AppConfig          `mapstructure:"app"`
	Database      DatabaseConfig     `mapstructure:"database"`
	Redis         RedisConfig        `mapstructure:"redis"`
	Storage       StorageConfig      `mapstructure:"storage"`
	S3            S3Config           `mapstructure:"s3"`
	S3Secondary   S3Config           `mapstructure:"s3_secondary"`
	JWT           JWTConfig          `mapstructure:"jwt"`
	Cloudflare    CloudflareConfig   `mapstructure:"cloudflare"`
	Worker        WorkerConfig       `mapstructure:"worker"`
	KMS           KMSConfig          `mapstructure:"kms"`
	Notifications NotificationConfig `mapstructure:"notifications"`
	RateLimits    RateLimitConfig    `mapstructure:"rate_limits"`
}

type RateLimitConfig struct {
	// Reqs/5min defaults
	Enterprise int `mapstructure:"enterprise"` // e.g. 1200
	Business   int `mapstructure:"business"`   // e.g. 600
	Pro        int `mapstructure:"pro"`        // e.g. 150
	Free       int `mapstructure:"free"`       // e.g. 30
}

type NotificationConfig struct {
	SlackWebhookURL string `mapstructure:"slack_webhook_url"`
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
// Multiple buckets can be configured.
type S3Config struct {
	Name            string `mapstructure:"name"` // Provider label (e.g. "garage", "hetzner")
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
	// Rate limit for Cloudflare API requests
	RateLimit float64 `mapstructure:"rate_limit"`
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
	Key       string            `mapstructure:"key"`        // Legacy single key (mapped to "v1")
	Keys      map[string]string `mapstructure:"keys"`       // Map of keyID -> hexKey
	ActiveKey string            `mapstructure:"active_key"` // ID of the key to use for encryption
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

	v.SetDefault("notifications.slack_webhook_url", "")

	v.SetDefault("cloudflare.base_url", "https://api.cloudflare.com/client/v4")
	v.SetDefault("cloudflare.request_timeout", "30s")
	v.SetDefault("cloudflare.max_window_size", "1h")
	v.SetDefault("kms.key", "")

	v.SetDefault("worker.scheduler_interval", "1m")
	v.SetDefault("worker.concurrency", 10)
	v.SetDefault("worker.log_retention_days", 395) // ~13 months – beyond NIS2 minimum

	v.SetDefault("rate_limits.enterprise", 1200) // 1200 reqs/5min (standard Ent)
	v.SetDefault("rate_limits.business", 600)    // Safe guess
	v.SetDefault("rate_limits.pro", 300)         // Safe guess
	v.SetDefault("rate_limits.free", 150)        // Safe guess

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
	// 4. Validate / Normalize KMS configuration
	if cfg.KMS.Keys == nil {
		cfg.KMS.Keys = make(map[string]string)
	}
	if cfg.KMS.Key != "" {
		// If legacy key is present, ensure it's in the map as "v1"
		if _, ok := cfg.KMS.Keys["v1"]; !ok {
			cfg.KMS.Keys["v1"] = cfg.KMS.Key
		}
	}
	if cfg.KMS.ActiveKey == "" {
		// Default to v1 if not specified
		cfg.KMS.ActiveKey = "v1"
	}
	// Ensure active key exists
	if _, ok := cfg.KMS.Keys[cfg.KMS.ActiveKey]; !ok {
		return nil, fmt.Errorf("active key %s not defined in kms.keys", cfg.KMS.ActiveKey)
	}
	return &cfg, nil
}
