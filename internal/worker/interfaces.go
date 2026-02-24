package worker

import (
	"context"

	"github.com/fabriziosalmi/rainlogs/internal/models"
	"github.com/google/uuid"
)

// Interfaces for dependency injection to allow testing.

// LogRepository defines database access for log jobs.
type LogRepository interface {
	ListExpired(ctx context.Context, customerID uuid.UUID, retentionDays int) ([]*models.LogJob, error)
	MarkExpired(ctx context.Context, id uuid.UUID) error
}

// LogStorage defines storage access for log expiration.
type LogStorage interface {
	DeleteObject(ctx context.Context, key string) error
}
