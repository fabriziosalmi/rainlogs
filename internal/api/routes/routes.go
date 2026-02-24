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

func Register(e *echo.Echo, database *db.DB, kms *kms.Encryptor, jwtSecret string, queue *asynq.Client, store *storage.MultiStore) {
	h := handlers.NewHandlers(database, kms, queue, store)

	// Public — self-registration only; profile reads require auth (own-record only).
	e.POST("/customers", h.CreateCustomer)

	// ── API-key protected ────────────────────────────────────────────────────
	api := e.Group("/api/v1")
	api.Use(middleware.APIKeyAuth(database))
	api.Use(middleware.CustomerRateLimit(30, 60)) // 30 req/s, burst 60 per customer
	api.Use(middleware.AuditLog(database.AuditEvents))

	// Viewer / Common Routes
	api.GET("/customers/:id", h.GetCustomer) // own record only

	api.GET("/zones", h.ListZones)
	api.GET("/zones/:zone_id/logs", h.GetZoneLogs)
	api.GET("/api-keys", h.ListAPIKeys)
	api.GET("/logs/jobs", h.ListLogJobs)
	api.GET("/logs/jobs/:job_id", h.GetLogJob)
	api.GET("/logs/jobs/:job_id/download", h.DownloadLogs)
	api.GET("/exports/:id", h.Export.Get)
	api.GET("/export", h.ExportCustomerData) // GDPR Art. 20 – data portability
	api.GET("/audit-log", h.ListAuditLog)    // GDPR Art. 30 / NIS2 Art. 21

	// Admin Only Routes
	admin := api.Group("")
	admin.Use(middleware.RequireAdmin())

	admin.DELETE("/customers/:id", h.DeleteCustomer) // GDPR Art. 17 – right to erasure

	admin.POST("/zones", h.CreateZone)
	admin.PATCH("/zones/:zone_id", h.UpdateZone)
	admin.DELETE("/zones/:zone_id", h.DeleteZone)
	admin.POST("/zones/:zone_id/pull", h.TriggerPull)

	admin.POST("/api-keys", h.CreateAPIKey)
	admin.DELETE("/api-keys/:key_id", h.RevokeAPIKey)

	admin.POST("/exports", h.Export.Create)

	// ── JWT protected (dashboard / internal) ────────────────────────────────
	dash := e.Group("/dashboard")
	dash.Use(middleware.JWTAuth(jwtSecret))
	dash.Use(middleware.CustomerRateLimit(30, 60))
	dash.Use(middleware.AuditLog(database.AuditEvents))

	dash.GET("/customers/:id", h.GetCustomer) // own record only
	dash.DELETE("/customers/:id", h.DeleteCustomer)

	dash.POST("/zones", h.CreateZone)
	dash.GET("/zones", h.ListZones)
	dash.PATCH("/zones/:zone_id", h.UpdateZone)
	dash.DELETE("/zones/:zone_id", h.DeleteZone)
	dash.POST("/zones/:zone_id/pull", h.TriggerPull)
	dash.GET("/zones/:zone_id/logs", h.GetZoneLogs)

	dash.POST("/api-keys", h.CreateAPIKey)
	dash.GET("/api-keys", h.ListAPIKeys)
	dash.DELETE("/api-keys/:key_id", h.RevokeAPIKey)

	dash.GET("/logs/jobs", h.ListLogJobs)
	dash.GET("/logs/jobs/:job_id", h.GetLogJob)
	dash.GET("/logs/jobs/:job_id/download", h.DownloadLogs)

	dash.GET("/export", h.ExportCustomerData)
	dash.GET("/audit-log", h.ListAuditLog)
}
