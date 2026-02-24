package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"

	"github.com/fabriziosalmi/rainlogs/internal/db"
	"github.com/fabriziosalmi/rainlogs/internal/kms"
	"github.com/fabriziosalmi/rainlogs/internal/models"
	"github.com/fabriziosalmi/rainlogs/internal/queue"
)

type ExportHandler struct {
	db    *db.DB
	queue *asynq.Client
	kms   *kms.Encryptor
}

func NewExportHandler(db *db.DB, queue *asynq.Client, kms *kms.Encryptor) *ExportHandler {
	return &ExportHandler{db: db, queue: queue, kms: kms}
}

type CreateExportRequest struct {
	S3Config models.ExportS3Config `json:"s3_config"`
	Start    time.Time             `json:"start"`
	End      time.Time             `json:"end"`
}

func (h *ExportHandler) Create(c echo.Context) error {
	var req CreateExportRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	customerID := c.Get("customer_id").(uuid.UUID)

	// Encrypt S3 Config
	configBytes, _ := json.Marshal(req.S3Config)
	configEnc, err := h.kms.Encrypt(string(configBytes))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to encrypt config"})
	}

	export := &models.LogExport{
		ID:          uuid.New(),
		CustomerID:  customerID,
		S3ConfigEnc: configEnc,
		FilterStart: req.Start,
		FilterEnd:   req.End,
		Status:      models.ExportStatusPending,
	}

	if err := h.db.LogExports.Create(c.Request().Context(), export); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create export job"})
	}

	// Enqueue Task
	task, _ := queue.NewLogExportTask(export.ID)
	if _, err := h.queue.Enqueue(task); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to enqueue task"})
	}

	return c.JSON(http.StatusCreated, export)
}

func (h *ExportHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	export, err := h.db.LogExports.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "export not found"})
	}

	// Ensure customer owns export
	customerID := c.Get("customer_id").(uuid.UUID)
	if export.CustomerID != customerID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	return c.JSON(http.StatusOK, export)
}
