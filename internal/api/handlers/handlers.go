package handlers

import (
	"errors"
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
	"github.com/jackc/pgx/v5/pgconn"
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
	Code      int    `json:"code"`
	Message   string `json:"message"`
	ErrorCode string `json:"error_code,omitempty"`
	ReqID     string `json:"request_id,omitempty"`
}

func apiErr(c echo.Context, code int, msg string, errCode ...string) error {
	reqID, _ := c.Get(middleware.ContextKeyRequestID).(string)
	resp := errResponse{Code: code, Message: msg, ReqID: reqID}
	if len(errCode) > 0 {
		resp.ErrorCode = errCode[0]
	}
	return c.JSON(code, resp)
}

// isUniqueViolation returns true if err is a PostgreSQL unique-constraint violation (23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
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
		return apiErr(c, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
	}
	if req.Name == "" || req.Email == "" || req.CFAccountID == "" || req.CFAPIKey == "" || req.RetentionDays < 1 {
		return apiErr(c, http.StatusBadRequest, "missing required fields", "INVALID_REQUEST")
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
		if isUniqueViolation(err) {
			return apiErr(c, http.StatusConflict, "email already registered", "CUSTOMER_EMAIL_EXISTS")
		}
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
		return apiErr(c, http.StatusBadRequest, "invalid id", "INVALID_REQUEST")
	}
	if id != customerID {
		return apiErr(c, http.StatusForbidden, "access denied", "ACCESS_DENIED")
	}

	customer, err := h.db.Customers.GetByID(c.Request().Context(), id)
	if err != nil {
		return apiErr(c, http.StatusNotFound, "customer not found")
	}

	return c.JSON(http.StatusOK, customer)
}

// DeleteCustomer permanently erases all customer data (GDPR Art. 17 – right to erasure).
func (h *Handlers) DeleteCustomer(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiErr(c, http.StatusBadRequest, "invalid id", "INVALID_REQUEST")
	}
	if id != customerID {
		return apiErr(c, http.StatusForbidden, "access denied", "ACCESS_DENIED")
	}

	ctx := c.Request().Context()

	// 1. Delete all stored log objects from object storage (best-effort).
	jobs, err := h.db.LogJobs.ListByCustomer(ctx, customerID, 9999, 0)
	if err != nil {
		c.Logger().Errorf("list jobs for erasure %s: %v", customerID, err)
	} else {
		for _, job := range jobs {
			if job.S3Key != "" {
				if delErr := h.storage.DeleteObject(ctx, job.S3Key); delErr != nil {
					c.Logger().Warnf("erasure: delete object %s: %v", job.S3Key, delErr)
				}
				_ = h.db.LogJobs.MarkExpired(ctx, job.ID)
			}
		}
	}

	// 2. Soft-delete all zones.
	if err := h.db.Zones.SoftDeleteByCustomer(ctx, customerID); err != nil {
		c.Logger().Errorf("soft-delete zones for %s: %v", customerID, err)
	}

	// 3. Revoke all API keys.
	if err := h.db.APIKeys.RevokeByCustomer(ctx, customerID); err != nil {
		c.Logger().Errorf("revoke keys for %s: %v", customerID, err)
	}

	// 4. Soft-delete the customer record.
	if err := h.db.Customers.SoftDelete(ctx, customerID); err != nil {
		c.Logger().Errorf("soft-delete customer %s: %v", customerID, err)
		return apiErr(c, http.StatusInternalServerError, "failed to erase customer")
	}

	return c.NoContent(http.StatusNoContent)
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
		return apiErr(c, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
	}
	const maxPullIntervalSecs = 518400 // 6 days – must stay below CF's 7-day log retention
	if req.ZoneID == "" || req.Name == "" || req.PullIntervalSecs < 300 || req.PullIntervalSecs > maxPullIntervalSecs {
		return apiErr(c, http.StatusBadRequest, "missing required fields or pull_interval_secs out of range [300, 518400]", "INVALID_REQUEST")
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
		return apiErr(c, http.StatusBadRequest, "invalid zone_id", "INVALID_REQUEST")
	}

	// Ensure the zone belongs to this customer
	zone, err := h.db.Zones.GetByID(c.Request().Context(), zoneID)
	if err != nil {
		return apiErr(c, http.StatusNotFound, "zone not found", "ZONE_NOT_FOUND")
	}
	if zone.CustomerID != customerID {
		return apiErr(c, http.StatusForbidden, "access denied", "ACCESS_DENIED")
	}

	if err := h.db.Zones.Delete(c.Request().Context(), zoneID); err != nil {
		c.Logger().Errorf("delete zone %s: %v", zoneID, err)
		return apiErr(c, http.StatusInternalServerError, "failed to delete zone")
	}

	return c.NoContent(http.StatusNoContent)
}

// UpdateZoneRequest carries mutable zone fields (all optional – only provided fields are applied).
type UpdateZoneRequest struct {
	Name             *string `json:"name"`
	PullIntervalSecs *int    `json:"pull_interval_secs"`
	Active           *bool   `json:"active"`
}

// UpdateZone patches a zone (pause/resume/rename) without deleting it.
func (h *Handlers) UpdateZone(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	zoneID, err := uuid.Parse(c.Param("zone_id"))
	if err != nil {
		return apiErr(c, http.StatusBadRequest, "invalid zone_id", "INVALID_REQUEST")
	}

	var req UpdateZoneRequest
	if err := c.Bind(&req); err != nil {
		return apiErr(c, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
	}

	ctx := c.Request().Context()
	zone, err := h.db.Zones.GetByID(ctx, zoneID)
	if err != nil {
		return apiErr(c, http.StatusNotFound, "zone not found", "ZONE_NOT_FOUND")
	}
	if zone.CustomerID != customerID {
		return apiErr(c, http.StatusForbidden, "access denied", "ACCESS_DENIED")
	}

	// Apply only the provided fields.
	name := zone.Name
	intervalSecs := zone.PullIntervalSecs
	active := zone.Active

	if req.Name != nil {
		name = *req.Name
	}
	if req.PullIntervalSecs != nil {
		const maxPullIntervalSecs = 518400
		if *req.PullIntervalSecs < 300 || *req.PullIntervalSecs > maxPullIntervalSecs {
			return apiErr(c, http.StatusBadRequest, "pull_interval_secs out of range [300, 518400]", "INVALID_REQUEST")
		}
		intervalSecs = *req.PullIntervalSecs
	}
	if req.Active != nil {
		active = *req.Active
	}

	if err := h.db.Zones.Update(ctx, zoneID, customerID, name, intervalSecs, active); err != nil {
		c.Logger().Errorf("update zone %s: %v", zoneID, err)
		return apiErr(c, http.StatusInternalServerError, "failed to update zone")
	}

	// Return the updated zone.
	updated, err := h.db.Zones.GetByID(ctx, zoneID)
	if err != nil {
		return apiErr(c, http.StatusInternalServerError, "failed to retrieve updated zone")
	}
	return c.JSON(http.StatusOK, zoneResponse{Zone: *updated, Health: zoneHealth(updated)})
}

// TriggerPull enqueues an immediate log pull for a zone.
func (h *Handlers) TriggerPull(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	zoneID, err := uuid.Parse(c.Param("zone_id"))
	if err != nil {
		return apiErr(c, http.StatusBadRequest, "invalid zone_id", "INVALID_REQUEST")
	}

	zone, err := h.db.Zones.GetByID(c.Request().Context(), zoneID)
	if err != nil {
		return apiErr(c, http.StatusNotFound, "zone not found", "ZONE_NOT_FOUND")
	}
	if zone.CustomerID != customerID {
		return apiErr(c, http.StatusForbidden, "access denied", "ACCESS_DENIED")
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
		return apiErr(c, http.StatusBadRequest, "invalid zone_id", "INVALID_REQUEST")
	}

	// Verify zone ownership.
	zone, err := h.db.Zones.GetByID(c.Request().Context(), zoneID)
	if err != nil {
		return apiErr(c, http.StatusNotFound, "zone not found", "ZONE_NOT_FOUND")
	}
	if zone.CustomerID != customerID {
		return apiErr(c, http.StatusForbidden, "access denied", "ACCESS_DENIED")
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
	Label         string `json:"label"           validate:"required"`
	ExpiresInDays int    `json:"expires_in_days"` // 0 = never expires
}

func (h *Handlers) CreateAPIKey(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	var req CreateAPIKeyRequest
	if err := c.Bind(&req); err != nil {
		return apiErr(c, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
	}
	if req.Label == "" {
		return apiErr(c, http.StatusBadRequest, "label is required", "INVALID_REQUEST")
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
	if req.ExpiresInDays > 0 {
		t := time.Now().UTC().AddDate(0, 0, req.ExpiresInDays)
		key.ExpiresAt = &t
	}

	if err := h.db.APIKeys.Create(c.Request().Context(), key); err != nil {
		return apiErr(c, http.StatusInternalServerError, "failed to save api key")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":         key.ID,
		"label":      key.Label,
		"prefix":     key.Prefix,
		"created_at": key.CreatedAt,
		"expires_at": key.ExpiresAt,
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
		return apiErr(c, http.StatusBadRequest, "invalid key_id", "INVALID_REQUEST")
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
		return apiErr(c, http.StatusBadRequest, "invalid job_id", "INVALID_REQUEST")
	}

	job, err := h.db.LogJobs.GetByID(c.Request().Context(), jobID)
	if err != nil {
		return apiErr(c, http.StatusNotFound, "job not found", "JOB_NOT_FOUND")
	}
	if job.CustomerID != customerID {
		return apiErr(c, http.StatusForbidden, "access denied", "ACCESS_DENIED")
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
		return apiErr(c, http.StatusBadRequest, "invalid job_id", "INVALID_REQUEST")
	}

	job, err := h.db.LogJobs.GetByID(c.Request().Context(), jobID)
	if err != nil {
		return apiErr(c, http.StatusNotFound, "job not found", "JOB_NOT_FOUND")
	}
	if job.CustomerID != customerID {
		return apiErr(c, http.StatusForbidden, "access denied", "ACCESS_DENIED")
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

// ── GDPR / Compliance Handlers ────────────────────────────────────────────────

// ExportCustomerData returns a structured JSON export of all customer data.
// GDPR Art. 20 – right to data portability.
func (h *Handlers) ExportCustomerData(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()

	customer, err := h.db.Customers.GetByID(ctx, customerID)
	if err != nil {
		return apiErr(c, http.StatusNotFound, "customer not found")
	}

	zones, err := h.db.Zones.ListByCustomer(ctx, customerID)
	if err != nil {
		return apiErr(c, http.StatusInternalServerError, "failed to fetch zones")
	}

	keys, err := h.db.APIKeys.ListByCustomer(ctx, customerID)
	if err != nil {
		return apiErr(c, http.StatusInternalServerError, "failed to fetch api keys")
	}

	jobs, err := h.db.LogJobs.ListByCustomer(ctx, customerID, 500, 0)
	if err != nil {
		return apiErr(c, http.StatusInternalServerError, "failed to fetch log jobs")
	}

	type customerExport struct {
		ID            uuid.UUID `json:"id"`
		Name          string    `json:"name"`
		Email         string    `json:"email"`
		CFAccountID   string    `json:"cf_account_id"`
		RetentionDays int       `json:"retention_days"`
		CreatedAt     time.Time `json:"created_at"`
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"exported_at":    time.Now().UTC(),
		"customer":       customerExport{customer.ID, customer.Name, customer.Email, customer.CFAccountID, customer.RetentionDays, customer.CreatedAt},
		"zones":          zones,
		"api_keys":       keys,
		"log_jobs_total": len(jobs),
		"log_jobs":       jobs,
	})
}

// ListAuditLog returns the most recent audit events for the authenticated customer.
// GDPR Art. 30 / NIS2 Art. 21.
func (h *Handlers) ListAuditLog(c echo.Context) error {
	customerID, err := mustCustomerID(c)
	if err != nil {
		return err
	}

	limit := 100
	offset := 0
	if l := c.QueryParam("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 1000 {
			limit = v
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	events, err := h.db.AuditEvents.ListByCustomer(c.Request().Context(), customerID, limit, offset)
	if err != nil {
		return apiErr(c, http.StatusInternalServerError, "failed to list audit events")
	}

	return c.JSON(http.StatusOK, events)
}
