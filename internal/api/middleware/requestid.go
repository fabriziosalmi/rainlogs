package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const ContextKeyRequestID = "request_id"

// RequestID injects a unique request ID into every request context and response header.
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Request().Header.Get("X-Request-ID")
			if id == "" {
				id = uuid.New().String()
			}
			c.Set(ContextKeyRequestID, id)
			c.Response().Header().Set("X-Request-ID", id)
			return next(c)
		}
	}
}
