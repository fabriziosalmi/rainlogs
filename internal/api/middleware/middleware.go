package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/fabriziosalmi/rainlogs/internal/auth"
	"github.com/fabriziosalmi/rainlogs/internal/db"
	"github.com/fabriziosalmi/rainlogs/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	ContextKeyCustomerID = "customer_id"
)

func APIKeyAuth(database *db.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization format")
			}

			apiKey := parts[1]
			// Reject oversized keys before any DB interaction (prevents DoS via huge strings).
			if len(apiKey) > 256 {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid api key format")
			}
			prefix, err := auth.PrefixOf(apiKey)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid api key format")
			}

			keys, err := database.APIKeys.GetByPrefix(c.Request().Context(), prefix)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "database error")
			}

			var validKey *models.APIKey

			for _, k := range keys {
				if auth.ValidateAPIKey(apiKey, k.KeyHash) {
					validKey = k
					break
				}
			}

			if validKey == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid api key")
			}

			// A1: enforce key expiration (ISO 27001 A.9.4).
			if validKey.ExpiresAt != nil && time.Now().After(*validKey.ExpiresAt) {
				return echo.NewHTTPError(http.StatusUnauthorized, "api key expired")
			}

			// Update last used asynchronously
			go func(id uuid.UUID) {
				_ = database.APIKeys.UpdateLastUsed(context.Background(), id)
			}(validKey.ID)

			c.Set(ContextKeyCustomerID, validKey.CustomerID)
			return next(c)
		}
	}
}

func JWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization format")
			}

			tokenStr := parts[1]
			claims, err := auth.VerifyJWT(secret, tokenStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}

			customerID, err := uuid.Parse(claims.CustomerID)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid customer id in token")
			}

			c.Set(ContextKeyCustomerID, customerID)
			return next(c)
		}
	}
}

// AuditLog writes a persistent audit record for every mutating request (POST, PATCH, DELETE).
// Records are written asynchronously (fire-and-forget) to avoid blocking the response path.
// GDPR Art. 30 / NIS2 Art. 21.
func AuditLog(auditRepo *db.AuditEventRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			method := c.Request().Method
			if method != http.MethodPost && method != http.MethodPatch && method != http.MethodDelete {
				return next(c)
			}

			err := next(c)

			statusCode := c.Response().Status
			if err != nil {
				if he, ok := err.(*echo.HTTPError); ok {
					statusCode = he.Code
				} else {
					statusCode = http.StatusInternalServerError
				}
			}

			custID, _ := c.Get(ContextKeyCustomerID).(uuid.UUID)
			reqID, _ := c.Get(ContextKeyRequestID).(string)
			action := auditAction(method, c.Path())
			resourceID := auditResourceID(c)

			var customerPtr *uuid.UUID
			if custID != uuid.Nil {
				id := custID
				customerPtr = &id
			}

			event := &models.AuditEvent{
				ID:         uuid.New(),
				CustomerID: customerPtr,
				RequestID:  reqID,
				Action:     action,
				ResourceID: resourceID,
				IPAddress:  c.RealIP(),
				StatusCode: statusCode,
			}

			go func() {
				_ = auditRepo.Create(context.Background(), event)
			}()

			return err
		}
	}
}

// auditAction maps HTTP method + registered route path to a semantic action string.
func auditAction(method, path string) string {
	p := path
	for _, pfx := range []string{"/api/v1", "/dashboard"} {
		if strings.HasPrefix(p, pfx) {
			p = p[len(pfx):]
			break
		}
	}
	switch method + " " + p {
	case "POST /zones":
		return "ZONE_CREATE"
	case "DELETE /zones/:zone_id":
		return "ZONE_DELETE"
	case "PATCH /zones/:zone_id":
		return "ZONE_UPDATE"
	case "POST /zones/:zone_id/pull":
		return "ZONE_PULL"
	case "DELETE /customers/:id":
		return "CUSTOMER_ERASE"
	case "POST /api-keys":
		return "APIKEY_CREATE"
	case "DELETE /api-keys/:key_id":
		return "APIKEY_REVOKE"
	default:
		return method + " " + p
	}
}

// auditResourceID extracts the primary resource identifier from route parameters.
func auditResourceID(c echo.Context) string {
	for _, param := range []string{"zone_id", "id", "key_id", "job_id"} {
		if v := c.Param(param); v != "" {
			return v
		}
	}
	return ""
}
