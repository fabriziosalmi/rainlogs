package models

import (
	"time"

	"github.com/google/uuid"
)

type ExportStatus string

const (
	ExportStatusPending    ExportStatus = "pending"
	ExportStatusProcessing ExportStatus = "processing"
	ExportStatusCompleted  ExportStatus = "completed"
	ExportStatusFailed     ExportStatus = "failed"
)

type LogExport struct {
	ID          uuid.UUID    `db:"id"             json:"id"`
	CustomerID  uuid.UUID    `db:"customer_id"    json:"customer_id"`
	S3ConfigEnc string       `db:"s3_config_enc"  json:"-"`
	FilterStart time.Time    `db:"filter_start"   json:"filter_start"`
	FilterEnd   time.Time    `db:"filter_end"     json:"filter_end"`
	Status      ExportStatus `db:"status"         json:"status"`
	LogCount    int64        `db:"log_count"      json:"log_count"`
	ByteCount   int64        `db:"byte_count"     json:"byte_count"`
	ErrorMsg    *string      `db:"error_msg"      json:"error_msg,omitempty"`
	CreatedAt   time.Time    `db:"created_at"     json:"created_at"`
	UpdatedAt   time.Time    `db:"updated_at"     json:"updated_at"`
}

type ExportS3Config struct {
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	PathPrefix      string `json:"path_prefix"`
}
