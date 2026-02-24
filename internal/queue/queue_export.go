package queue

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

type LogExportPayload struct {
	ExportID uuid.UUID `json:"export_id"`
}

func NewLogExportTask(exportID uuid.UUID) (*asynq.Task, error) {
	payload, err := json.Marshal(LogExportPayload{
		ExportID: exportID,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeLogExport, payload, asynq.Queue(QueueLow), asynq.MaxRetry(3)), nil
}

func ParseLogExportPayload(t *asynq.Task) (*LogExportPayload, error) {
	var p LogExportPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return nil, fmt.Errorf("queue: unmarshal task: %w", err)
	}
	return &p, nil
}
