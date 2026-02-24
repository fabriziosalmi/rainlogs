package worker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fabriziosalmi/rainlogs/internal/models"
	"github.com/fabriziosalmi/rainlogs/internal/queue"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockLogStorage simulates storage interactions
type MockLogStorage struct {
	mock.Mock
}

func (m *MockLogStorage) DeleteObject(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// MockLogRepository simulates database interactions
type MockLogRepository struct {
	mock.Mock
}

func (m *MockLogRepository) ListExpired(ctx context.Context, customerID uuid.UUID, retentionDays int) ([]*models.LogJob, error) {
	args := m.Called(ctx, customerID, retentionDays)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.LogJob), args.Error(1)
}

func (m *MockLogRepository) MarkExpired(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestLogExpireProcessor_ProcessTask(t *testing.T) {
	// 1. Setup
	mockStorage := new(MockLogStorage)
	mockRepo := new(MockLogRepository)
	logger := zap.NewNop()

	// 2. Define test data
	customerID := uuid.New()
	jobID := uuid.New()
	s3Key := "logs/test-job.gz"
	retentionDays := 30

	expiredJob := &models.LogJob{
		ID:    jobID,
		S3Key: s3Key,
	}

	// 3. Configure Mocks
	// Expect ListExpired to specify retention (e.g. 30 days) and return our job
	mockRepo.On("ListExpired", mock.Anything, customerID, retentionDays).Return([]*models.LogJob{expiredJob}, nil)

	// Expect DeleteObject to be called for the job's S3 key
	mockStorage.On("DeleteObject", mock.Anything, s3Key).Return(nil)

	// Expect MarkExpired to be called for the job ID
	mockRepo.On("MarkExpired", mock.Anything, jobID).Return(nil)

	// 4. Initialize Processor
	p := NewLogExpireProcessor(mockRepo, mockStorage, logger)

	// 5. Create Task Payload
	payload := queue.LogExpirePayload{
		CustomerID:    customerID,
		RetentionDays: retentionDays,
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask(queue.TypeLogExpire, payloadBytes) // TypeLogExpire is defined in queue/queue.go

	// 6. Execute
	err := p.ProcessTask(context.Background(), task)

	// 7. Verify
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}
