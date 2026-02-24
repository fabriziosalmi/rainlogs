package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Backend defines the interface for log storage systems.
type Backend interface {
	// PutLogs stores compressed logs and returns metadata.
	// logType distinguishes the bucket path prefix (e.g. "logs" vs "security").
	PutLogs(ctx context.Context, customerID, zoneID uuid.UUID, from, to time.Time, raw []byte, logType string) (key, sha256hex string, compressedBytes, logLines int64, err error)

	// GetLogs retrieves the raw compressed content of a log object.
	GetLogs(ctx context.Context, key string) ([]byte, error)

	// DeleteObject removes a log object (used for retention/expiry).
	DeleteObject(ctx context.Context, key string) error

	// Provider returns the name of the storage provider (e.g., "s3", "fs").
	Provider() string
}
