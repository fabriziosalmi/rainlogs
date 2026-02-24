package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/fabriziosalmi/rainlogs/internal/api/middleware"
	"github.com/fabriziosalmi/rainlogs/internal/auth"
	"github.com/fabriziosalmi/rainlogs/internal/db"
	"github.com/fabriziosalmi/rainlogs/internal/kms"
	"github.com/fabriziosalmi/rainlogs/internal/models"
	"github.com/fabriziosalmi/rainlogs/internal/queue"
	"github.com/fabriziosalmi/rainlogs/internal/storage"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
)

type Handlers struct {
	db      *db.DB
	kms     *kms.Encryptor
	queue   *asynq.Client
	storage *storage.MultiStore
}

func NewHandlers(db *db.DB, kms *kms.Encryptor, queue *asynq.Client, store *storage.MultiStore) *Handlers {
	return &Handlers{
		db:      db,
		kms:     kms,
		queue:   queue,
		storage: store,
	}
}

// ── Error helpers ─────────────────────────────────────────────────────────────

type errResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	ReqID   string `json:"request_id,omitempty"`
}

func apiErr(c echo.Context, code int, msg string) error {
	reqID, _ := c.Get(middleware.ContextKeyRequestID).(string)
	return c.JSON(code, errResponse{Code: code, Message: msg, ReqID: reqID})
}

// mustCustomerID extracts the authenticated customer UUID from the Echo context.
// Returns a 500 if middleware failed to populate the value (should never happen
// on a properly guarded route, but avoids a nil-pointer panic if it does).
func mustCustomerID(c echo.Context) (uuid.UUID, error) {
	v, ok := c.Get(middleware.ContextKeyCustomerID).(uuid.UUID)
	if !ok {
		return uuid.UUID{}, apiErr(c, http.StatusInternalServerError, "auth context missing")
	}
	return v, nil
}

// ── Customer Handlers ─────────────────────────────────────────────────────────

type CreateCustomerRequest struct {
	Name          string `json:"name"           validate:"required"`
	Email         string `json:"email"          validate:"required,email"`
	CFAccountID   string `json:"cf_account_id"  validate:"required"`
	CFAPIKey      string `json:"cf_api_key"     validate:"required"`
	RetentionDays int    `json:"retention_days" validate:"required,min=1"`
}

func (h *Handlers) CreateCustomer(c echo.Context) error {
	var req CreateCustomerRequest
	if err := c.Bind(&req); err != nil {
		return apiErr(c, http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" || req.Email == "" || req.CFAccountID == "" || req.CFAPIKey == "" || req.RetentionDays < 1 {
		return apiErr(c, http.StatusBadRequest, "missing required fields")
	}

	encKey, err := h.kms.Encrypt(req.CFAPIKey)
	if err != nil {
		return apiErr(c, http.StatusInternalServerError, "failed to encrypt api key")
	}

	customer := &models.Customer{
		ID:            uuid.New(),
		Name:          req.Name,
		Email:         req.Email,
		CFAccountID:   req.CFAccountID,
		CFAPIKeyEnc:   encKey,
		RetentionDays: req.RetentionDays,
	}

	if err := h.db.Customers.Create(c.Request().Context(), customer); err != nil {
		c.Logger().Errorf("create customer: %v", err)
		return apiErr(c, http.StatusInternalServerError, "failed to create customer")
	}

	return c.JSON(http.StatusCreated, customer)
}

func (h *Handlers) GetCustomer(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiErr(c, http.StatusBadRequest, "invalid id")
	}
	if id != customerID {
		return apiErr(c, http.StatusForbidden, "access denied")
	}

	customer, err := h.db.Customers.GetByID(c.Request().Context(), id)
	if err != nil {
		return apiErr(c, http.StatusNotFound, "customer not found")
	}

	return c.JSON(http.StatusOK, customer)
}

// ── Zone Handlers ─────────────────────────────────────────────────────────────

// zoneResponse adds a computed health field to the Zone model.
type zoneResponse struct {
	models.Zone
	Health string `json:"health"`
}

// zoneHealth returns "ok", "stale", or "never_pulled" based on last pull time.
func zoneHealth(z *models.Zone) string {
	if z.LastPulledAt == nil {
		return "never_pulled"
	}
	if time.Since(*z.LastPulledAt) > time.Duration(z.PullIntervalSecs)*2*time.Second {
		return "stale"
	}
	return "ok"
}

type CreateZoneRequest struct {
	ZoneID           string `json:"zone_id"            validate:"required"`
	Name             string `json:"name"               validate:"required"`
	PullIntervalSecs int    `json:"pull_interval_secs" validate:"required,min=300"`
}

func (h *Handlers) CreateZone(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	var req CreateZoneRequest
	if err := c.Bind(&req); err != nil {
		return apiErr(c, http.StatusBadRequest, "invalid request body")
	}
	const maxPullIntervalSecs = 518400 // 6 days – must stay below CF's 7-day log retention
	if req.ZoneID == "" || req.Name == "" || req.PullIntervalSecs < 300 || req.PullIntervalSecs > maxPullIntervalSecs {
		return apiErr(c, http.StatusBadRequest, "missing required fields or pull_interval_secs out of range [300, 518400]")
	}

	zone := &models.Zone{
		ID:               uuid.New(),
		CustomerID:       customerID,
		ZoneID:           req.ZoneID,
		Name:             req.Name,
		PullIntervalSecs: req.PullIntervalSecs,
		Active:           true,
	}

	if err := h.db.Zones.Create(c.Request().Context(), zone); err != nil {
		c.Logger().Errorf("create zone: %v", err)
		return apiErr(c, http.StatusInternalServerError, "failed to create zone")
	}

	return c.JSON(http.StatusCreated, zone)
}

func (h *Handlers) ListZones(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	zones, err := h.db.Zones.ListByCustomer(c.Request().Context(), customerID)
	if err != nil {
		return apiErr(c, http.StatusInternalServerError, "failed to list zones")
	}

	resp := make([]zoneResponse, len(zones))
	for i, z := range zones {
		resp[i] = zoneResponse{Zone: *z, Health: zoneHealth(z)}
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handlers) DeleteZone(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	zoneID, err := uuid.Parse(c.Param("zone_id"))
	if err != nil {
		return apiErr(c, http.StatusBadRequest, "invalid zone_id")
	}

	// Ensure the zone belongs to this customer
	zone, err := h.db.Zones.GetByID(c.Request().Context(), zoneID)
	if err != nil {
		return apiErr(c, http.StatusNotFound, "zone not found")
	}
	if zone.CustomerID != customerID {
		return apiErr(c, http.StatusForbidden, "access denied")
	}

	if err := h.db.Zones.Delete(c.Request().Context(), zoneID); err != nil {
		c.Logger().Errorf("delete zone %s: %v", zoneID, err)
		return apiErr(c, http.StatusInternalServerError, "failed to delete zone")
	}

	return c.NoContent(http.StatusNoContent)
}

// TriggerPull enqueues an immediate log pull for a zone.
func (h *Handlers) TriggerPull(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	zoneID, err := uuid.Parse(c.Param("zone_id"))
	if err != nil {
		return apiErr(c, http.StatusBadRequest, "invalid zone_id")
	}

	zone, err := h.db.Zones.GetByID(c.Request().Context(), zoneID)
	if err != nil {
		return apiErr(c, http.StatusNotFound, "zone not found")
	}
	if zone.CustomerID != customerID {
		return apiErr(c, http.StatusForbidden, "access denied")
	}

	now := time.Now().UTC()
	start := now.Add(-time.Duration(zone.PullIntervalSecs) * time.Second)
	if zone.LastPulledAt != nil {
		start = *zone.LastPulledAt
	}

	task, err := queue.NewLogPullTask(queue.LogPullPayload{
		ZoneID:      zone.ID,
		CustomerID:  customerID,
		PeriodStart: start,
		PeriodEnd:   now,
	})
	if err != nil {
		return apiErr(c, http.StatusInternalServerError, "failed to create pull task")
	}

	info, err := h.queue.EnqueueContext(c.Request().Context(), task)
	if err != nil {
		c.Logger().Errorf("enqueue pull for zone %s: %v", zoneID, err)
		return apiErr(c, http.StatusInternalServerError, "failed to enqueue pull task")
	}

	return c.JSON(http.StatusAccepted, map[string]string{
		"task_id": info.ID,
		"status":  info.State.String(),
	})
}

// GetZoneLogs lists log jobs for a specific zone owned by the authenticated customer.
func (h *Handlers) GetZoneLogs(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	zoneID, err := uuid.Parse(c.Param("zone_id"))
	if err != nil {
		return apiErr(c, http.StatusBadRequest, "invalid zone_id")
	}

	// Verify zone ownership.
	zone, err := h.db.Zones.GetByID(c.Request().Context(), zoneID)
	if err != nil {
		return apiErr(c, http.StatusNotFound, "zone not found")
	}
	if zone.CustomerID != customerID {
		return apiErr(c, http.StatusForbidden, "access denied")
	}

	limit := 50
	offset := 0
	if l := c.QueryParam("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	jobs, err := h.db.LogJobs.ListByZone(c.Request().Context(), customerID, zoneID, limit, offset)
	if err != nil {
		return apiErr(c, http.StatusInternalServerError, "failed to list zone log jobs")
	}

	return c.JSON(http.StatusOK, jobs)
}

// ── API Key Handlers ──────────────────────────────────────────────────────────

type CreateAPIKeyRequest struct {
	Label string `json:"label" validate:"required"`
}

func (h *Handlers) CreateAPIKey(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	var req CreateAPIKeyRequest
	if err := c.Bind(&req); err != nil {
		return apiErr(c, http.StatusBadRequest, "invalid request body")
	}
	if req.Label == "" {
		return apiErr(c, http.StatusBadRequest, "label is required")
	}

	plaintext, hash, prefix, err := auth.GenerateAPIKey()
	if err != nil {
		return apiErr(c, http.StatusInternalServerError, "failed to generate api key")
	}

	key := &models.APIKey{
		ID:         uuid.New(),
		CustomerID: customerID,
		Prefix:     prefix,
		KeyHash:    hash,
		Label:      req.Label,
	}

	if err := h.db.APIKeys.Create(c.Request().Context(), key); err != nil {
		return apiErr(c, http.StatusInternalServerError, "failed to save api key")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":         key.ID,
		"label":      key.Label,
		"prefix":     key.Prefix,
		"created_at": key.CreatedAt,
		"api_key":    plaintext, // Shown only once – store securely
	})
}

func (h *Handlers) ListAPIKeys(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	keys, err := h.db.APIKeys.ListByCustomer(c.Request().Context(), customerID)
	if err != nil {
		return apiErr(c, http.StatusInternalServerError, "failed to list api keys")
	}

	return c.JSON(http.StatusOK, keys)
}

func (h *Handlers) RevokeAPIKey(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	keyID, err := uuid.Parse(c.Param("key_id"))
	if err != nil {
		return apiErr(c, http.StatusBadRequest, "invalid key_id")
	}

	// Verify the key belongs to this customer (single-row lookup).
	if _, err := h.db.APIKeys.GetByCustomerAndID(c.Request().Context(), customerID, keyID); err != nil {
		return apiErr(c, http.StatusNotFound, "api key not found")
	}

	if err := h.db.APIKeys.Revoke(c.Request().Context(), keyID); err != nil {
		return apiErr(c, http.StatusInternalServerError, "failed to revoke api key")
	}

	return c.NoContent(http.StatusNoContent)
}

// ── Log Job Handlers ──────────────────────────────────────────────────────────

func (h *Handlers) ListLogJobs(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	limit := 50
	offset := 0
	if l := c.QueryParam("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	jobs, err := h.db.LogJobs.ListByCustomer(c.Request().Context(), customerID, limit, offset)
	if err != nil {
		return apiErr(c, http.StatusInternalServerError, "failed to list log jobs")
	}

	return c.JSON(http.StatusOK, jobs)
}

func (h *Handlers) GetLogJob(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	jobID, err := uuid.Parse(c.Param("job_id"))
	if err != nil {
		return apiErr(c, http.StatusBadRequest, "invalid job_id")
	}

	job, err := h.db.LogJobs.GetByID(c.Request().Context(), jobID)
	if err != nil {
		return apiErr(c, http.StatusNotFound, "job not found")
	}
	if job.CustomerID != customerID {
		return apiErr(c, http.StatusForbidden, "access denied")
	}

	return c.JSON(http.StatusOK, job)
}

// DownloadLogs streams the raw (decompressed) NDJSON log data for a job.
func (h *Handlers) DownloadLogs(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	jobID, err := uuid.Parse(c.Param("job_id"))
	if err != nil {
		return apiErr(c, http.StatusBadRequest, "invalid job_id")
	}

	job, err := h.db.LogJobs.GetByID(c.Request().Context(), jobID)
	if err != nil {
		return apiErr(c, http.StatusNotFound, "job not found")
	}
	if job.CustomerID != customerID {
		return apiErr(c, http.StatusForbidden, "access denied")
	}
	if job.S3Key == "" {
		return apiErr(c, http.StatusNotFound, "no archive available for this job")
	}

	data, err := h.storage.GetLogs(c.Request().Context(), job.S3Key)
	if err != nil {
		c.Logger().Errorf("download logs for job %s: %v", jobID, err)
		return apiErr(c, http.StatusInternalServerError, "failed to retrieve log archive")
	}

	filename := fmt.Sprintf("rainlogs_%s_%s.ndjson",
		job.PeriodStart.UTC().Format("20060102T150405Z"),
		job.PeriodEnd.UTC().Format("20060102T150405Z"),
	)

	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Response().Header().Set("X-SHA256", job.SHA256)
	c.Response().Header().Set("X-Chain-Hash", job.ChainHash)
	return c.Blob(http.StatusOK, "application/x-ndjson", data)
}
