package routes

import (
	"github.com/fabriziosalmi/rainlogs/internal/api/handlers"
	"github.com/fabriziosalmi/rainlogs/internal/api/middleware"
	"github.com/fabriziosalmi/rainlogs/internal/db"
	"github.com/fabriziosalmi/rainlogs/internal/kms"
	"github.com/fabriziosalmi/rainlogs/internal/storage"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
)

func Register(e *echo.Echo, db *db.DB, kms *kms.Encryptor, jwtSecret string, queue *asynq.Client, store *storage.MultiStore) {
	h := handlers.NewHandlers(db, kms, queue, store)

	// Public
	e.POST("/customers", h.CreateCustomer)
	e.GET("/customers/:id", h.GetCustomer)

	// API-key protected
	api := e.Group("/api/v1")
	api.Use(middleware.APIKeyAuth(db))

	api.POST("/zones", h.CreateZone)
	api.GET("/zones", h.ListZones)
	api.DELETE("/zones/:zone_id", h.DeleteZone)
	api.POST("/zones/:zone_id/pull", h.TriggerPull)

	api.POST("/api-keys", h.CreateAPIKey)
	api.GET("/api-keys", h.ListAPIKeys)
	api.DELETE("/api-keys/:key_id", h.RevokeAPIKey)

	api.GET("/logs/jobs", h.ListLogJobs)
	api.GET("/logs/jobs/:job_id", h.GetLogJob)
	api.GET("/logs/jobs/:job_id/download", h.DownloadLogs)

	// JWT protected (dashboard / internal)
	dash := e.Group("/dashboard")
	dash.Use(middleware.JWTAuth(jwtSecret))

	dash.POST("/zones", h.CreateZone)
	dash.GET("/zones", h.ListZones)
	dash.DELETE("/zones/:zone_id", h.DeleteZone)
	dash.POST("/zones/:zone_id/pull", h.TriggerPull)

	dash.POST("/api-keys", h.CreateAPIKey)
	dash.GET("/api-keys", h.ListAPIKeys)
	dash.DELETE("/api-keys/:key_id", h.RevokeAPIKey)

	dash.GET("/logs/jobs", h.ListLogJobs)
	dash.GET("/logs/jobs/:job_id", h.GetLogJob)
	dash.GET("/logs/jobs/:job_id/download", h.DownloadLogs)
}
