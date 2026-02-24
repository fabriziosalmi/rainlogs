package handlers

import (
	"net/http"

	"github.com/fabriziosalmi/rainlogs/internal/api/middleware"
	"github.com/fabriziosalmi/rainlogs/internal/auth"
	"github.com/fabriziosalmi/rainlogs/internal/db"
	"github.com/fabriziosalmi/rainlogs/internal/kms"
	"github.com/fabriziosalmi/rainlogs/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Handlers struct {
	db  *db.DB
	kms *kms.Encryptor
}

func NewHandlers(db *db.DB, kms *kms.Encryptor) *Handlers {
	return &Handlers{
		db:  db,
		kms: kms,
	}
}

// Customer Handlers

type CreateCustomerRequest struct {
	Name          string `json:"name" validate:"required"`
	Email         string `json:"email" validate:"required,email"`
	CFAccountID   string `json:"cf_account_id" validate:"required"`
	CFAPIKey      string `json:"cf_api_key" validate:"required"`
	RetentionDays int    `json:"retention_days" validate:"required,min=1"`
}

func (h *Handlers) CreateCustomer(c echo.Context) error {
	var req CreateCustomerRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	encKey, err := h.kms.Encrypt(req.CFAPIKey)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to encrypt api key")
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
		c.Logger().Errorf("failed to create customer: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create customer")
	}

	return c.JSON(http.StatusCreated, customer)
}

func (h *Handlers) GetCustomer(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	customer, err := h.db.Customers.GetByID(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "customer not found")
	}

	return c.JSON(http.StatusOK, customer)
}

// Zone Handlers

type CreateZoneRequest struct {
	ZoneID           string `json:"zone_id" validate:"required"`
	Name             string `json:"name" validate:"required"`
	PullIntervalSecs int    `json:"pull_interval_secs" validate:"required,min=60"`
}

func (h *Handlers) CreateZone(c echo.Context) error {
	customerID := c.Get(middleware.ContextKeyCustomerID).(uuid.UUID)

	var req CreateZoneRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
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
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create zone")
	}

	return c.JSON(http.StatusCreated, zone)
}

func (h *Handlers) ListZones(c echo.Context) error {
	customerID := c.Get(middleware.ContextKeyCustomerID).(uuid.UUID)

	zones, err := h.db.Zones.ListByCustomer(c.Request().Context(), customerID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list zones")
	}

	return c.JSON(http.StatusOK, zones)
}

// API Key Handlers

type CreateAPIKeyRequest struct {
	Label string `json:"label" validate:"required"`
}

func (h *Handlers) CreateAPIKey(c echo.Context) error {
	customerID := c.Get(middleware.ContextKeyCustomerID).(uuid.UUID)

	var req CreateAPIKeyRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	plaintext, hash, prefix, err := auth.GenerateAPIKey()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate api key")
	}

	key := &models.APIKey{
		ID:         uuid.New(),
		CustomerID: customerID,
		Prefix:     prefix,
		KeyHash:    hash,
		Label:      req.Label,
	}

	if err := h.db.APIKeys.Create(c.Request().Context(), key); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save api key")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":         key.ID,
		"label":      key.Label,
		"created_at": key.CreatedAt,
		"api_key":    plaintext, // Only shown once
	})
}

// Log Job Handlers

func (h *Handlers) ListLogJobs(c echo.Context) error {
	customerID := c.Get(middleware.ContextKeyCustomerID).(uuid.UUID)

	jobs, err := h.db.LogJobs.ListByCustomer(c.Request().Context(), customerID, 50, 0)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list log jobs")
	}

	return c.JSON(http.StatusOK, jobs)
}
