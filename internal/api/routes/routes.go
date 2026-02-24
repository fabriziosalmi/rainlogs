package routes

import (
	"github.com/fabriziosalmi/rainlogs/internal/api/handlers"
	"github.com/fabriziosalmi/rainlogs/internal/api/middleware"
	"github.com/fabriziosalmi/rainlogs/internal/db"
	"github.com/fabriziosalmi/rainlogs/internal/kms"
	"github.com/labstack/echo/v4"
)

func Register(e *echo.Echo, db *db.DB, kms *kms.Encryptor, jwtSecret string) {
	h := handlers.NewHandlers(db, kms)

	// Public routes
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})
	e.POST("/customers", h.CreateCustomer)
	e.GET("/customers/:id", h.GetCustomer)

	// API Key protected routes
	api := e.Group("/api/v1")
	api.Use(middleware.APIKeyAuth(db))

	api.POST("/zones", h.CreateZone)
	api.GET("/zones", h.ListZones)
	api.POST("/api-keys", h.CreateAPIKey)
	api.GET("/logs/jobs", h.ListLogJobs)

	// JWT protected routes (e.g., for a dashboard)
	dash := e.Group("/dashboard")
	dash.Use(middleware.JWTAuth(jwtSecret))

	dash.POST("/zones", h.CreateZone)
	dash.GET("/zones", h.ListZones)
	dash.POST("/api-keys", h.CreateAPIKey)
	dash.GET("/logs/jobs", h.ListLogJobs)
}
