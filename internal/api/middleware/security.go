package middleware

import (
	"github.com/labstack/echo/v4"
)

// SecurityHeaders adds common security headers to every response.
func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-XSS-Protection", "1; mode=block")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Content-Security-Policy", "default-src 'none'")
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			return next(c)
		}
	}
}
