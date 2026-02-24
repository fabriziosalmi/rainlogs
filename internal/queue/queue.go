package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const (
	TypeLogPull      = "log:pull"
	TypeSecurityPoll = "security:poll"
	TypeLogVerify    = "log:verify"
	TypeLogExpire    = "log:expire"

	QueueCritical = "critical"
	QueueDefault  = "default"
	QueueLow      = "low"
)

// LogPullPayload is the task payload for TypeLogPull.
type LogPullPayload struct {
	ZoneID      uuid.UUID `json:"zone_id"`
	CustomerID  uuid.UUID `json:"customer_id"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
}

// SecurityPollPayload is the task payload for TypeSecurityPoll.
type SecurityPollPayload struct {
	ZoneID      uuid.UUID `json:"zone_id"`
	CustomerID  uuid.UUID `json:"customer_id"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
}

// LogVerifyPayload is the task payload for TypeLogVerify.
type LogVerifyPayload struct {
	JobID uuid.UUID `json:"job_id"`
}

// LogExpirePayload is the task payload for TypeLogExpire.
type LogExpirePayload struct {
	CustomerID    uuid.UUID `json:"customer_id"`
	RetentionDays int       `json:"retention_days"`
}

func NewLogPullTask(p LogPullPayload) (*asynq.Task, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal LogPull: %w", err)
	}
	return asynq.NewTask(TypeLogPull, b, asynq.Queue(QueueDefault)), nil
}

func NewSecurityPollTask(p SecurityPollPayload) (*asynq.Task, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal SecurityPoll: %w", err)
	}
	// Use QueueDefault or separate queue? Default is fine.
	return asynq.NewTask(TypeSecurityPoll, b, asynq.Queue(QueueDefault)), nil
}

func NewLogVerifyTask(p LogVerifyPayload) (*asynq.Task, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("queue: marshal LogVerify: %w", err)
	}
	return asynq.NewTask(TypeLogVerify, b, asynq.Queue(QueueLow)), nil
}

func NewLogExpireTask(p LogExpirePayload) (*asynq.Task, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("queue: marshal LogExpire: %w", err)
	}
	return asynq.NewTask(TypeLogExpire, b, asynq.Queue(QueueLow)), nil
}

func ParseLogPullPayload(t *asynq.Task) (LogPullPayload, error) {
	var p LogPullPayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}

func ParseSecurityPollPayload(t *asynq.Task) (SecurityPollPayload, error) {
	var p SecurityPollPayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}

func ParseLogVerifyPayload(t *asynq.Task) (LogVerifyPayload, error) {
	var p LogVerifyPayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}

func ParseLogExpirePayload(t *asynq.Task) (LogExpirePayload, error) {
	var p LogExpirePayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}
