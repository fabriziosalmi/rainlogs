package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/fabriziosalmi/rainlogs/internal/auth"
	"github.com/fabriziosalmi/rainlogs/internal/db"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	ContextKeyCustomerID = "customer_id"
)

func APIKeyAuth(db *db.DB) echo.MiddlewareFunc {
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
			prefix, err := auth.PrefixOf(apiKey)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid api key format")
			}

			keys, err := db.APIKeys.GetByPrefix(c.Request().Context(), prefix)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "database error")
			}

			var validKey *uuid.UUID
			var customerID uuid.UUID

			for _, k := range keys {
				if auth.ValidateAPIKey(apiKey, k.KeyHash) {
					validKey = &k.ID
					customerID = k.CustomerID
					break
				}
			}

			if validKey == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid api key")
			}

			// Update last used asynchronously
			go func(id uuid.UUID) {
				_ = db.APIKeys.UpdateLastUsed(context.Background(), id)
			}(*validKey)

			c.Set(ContextKeyCustomerID, customerID)
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
