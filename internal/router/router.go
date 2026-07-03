package router

import (
	"dmama_api/internal/config"
	"dmama_api/internal/handler"
	"dmama_api/internal/middleware"
	"dmama_api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/mongo"
)

func Setup(app *fiber.App, pool *pgxpool.Pool, gisPool *pgxpool.Pool, featureDB, usersDB *mongo.Database, usageRec *repository.UsageRecorder, featureReady bool, cfg *config.Config) {
	// API documentation
	app.Static("/template", "./template")
	app.Static("/docs/static", "./template/static")
	app.Get("/docs", func(c *fiber.Ctx) error {
		return c.SendFile("./template/index.html")
	})
	app.Get("/docs/api-monitor", func(c *fiber.Ctx) error {
		return c.SendFile("./template/api_monitor.html")
	})

	api := app.Group("/api")

	// Health check (no API key required)
	api.Get("/health", handler.HealthCheck)

	// API key authentication for all routes below
	api.Use(middleware.APIKeyAuth(cfg))

	// API usage monitor is read-only and registered before UsageLogger so dashboard refreshes do
	// not add self-referential records to auth_logs.dmama_use.
	if gisPool != nil {
		monitorH := handler.NewMonitorHandler(gisPool)
		api.Get("/monitor/usage", monitorH.Usage)
	} else {
		api.Get("/monitor/usage", handler.MonitorUnavailable)
	}

	// Usage/credit logging for every authenticated /api request (health above is excluded).
	if usageRec != nil {
		api.Use(middleware.UsageLogger(usageRec))
	}

	// Pipe endpoints (1, 2, 11, 12)
	pipeH := handler.NewPipeHandler(pool)
	api.Post("/pipe/assign-logger", pipeH.AssignLogger)
	api.Get("/pipe/geometry", pipeH.GetGeometry)
	api.Get("/pipe/at-leakpoint", pipeH.AtLeakpoint)
	api.Get("/pipe/assets", pipeH.AssetsOnPipe)

	// DMA endpoints (3, 4, 5, 6, 7, 14, 15, 16, daily_metercount)
	dmaH := handler.NewDMAHandler(pool)
	api.Get("/dma/at-leakpoint", dmaH.AtLeakpoint)
	api.Get("/dma/boundary", dmaH.GetBoundary)
	api.Get("/dma/usage", dmaH.GetUsage)
	api.Get("/dma/population", dmaH.GetPopulation)
	api.Get("/dma/stats", dmaH.GetStats)
	api.Get("/dma/stats-region", dmaH.GetStatsRegion)
	api.Get("/dma/daily_metercount", dmaH.GetDailyMeterCount)
	api.Get("/dma/pipe-length", dmaH.GetPipeLength)
	api.Get("/dma/map", dmaH.GetMapData)
	api.Get("/dma/customers", dmaH.GetCustomers)
	api.Get("/dma/usage-v2", dmaH.GetUsageV2)
	api.Get("/dma/leakpoints-by-size", dmaH.LeakpointsBySize)
	api.Get("/dma/pipe-length-clipped", dmaH.PipeLengthClipped)

	// Office endpoint (8)
	officeH := handler.NewOfficeHandler(pool)
	api.Get("/office", officeH.List)

	// Waterworks endpoint (9)
	wwH := handler.NewWaterworksHandler(pool)
	api.Get("/waterworks", wwH.GetMarkers)

	// Asset endpoints (10, 13)
	assetH := handler.NewAssetHandler(pool)
	api.Get("/asset/nearest", assetH.FindNearest)
	api.Post("/asset/in-area", assetH.InArea)

	// Feature endpoints — CRUD for drawn dma_boundary/flow_meter/step_test.
	// Storage: Vallaris MongoDB; topology validation + mirror: PostgreSQL 16 (dmama_layer).
	// Registered only when both backends are up AND the dmama_layer schema is ready
	// (featureReady, computed in cmd/server/main.go).
	if featureReady {
		featureH := handler.NewFeatureHandler(gisPool, featureDB, usersDB, cfg.SystemUser, cfg.UserDomain)
		api.Post("/features/:shape/:pwaCode/validate", featureH.Validate)
		api.Post("/features/:shape/:pwaCode/sync", featureH.Sync)
		api.Post("/features/:shape/:pwaCode", featureH.Create)
		api.Get("/features/:shape/:pwaCode", featureH.List)
		api.Get("/features/:shape/:pwaCode/:id", featureH.Get)
		api.Put("/features/:shape/:pwaCode/:id", featureH.Update)
		api.Delete("/features/:shape/:pwaCode/:id", featureH.Delete)
	} else {
		// Backends not configured: still register the routes so requests get an explicit 503
		// instead of Fiber's misleading "Cannot POST" 404.
		api.All("/features/*", handler.FeatureUnavailable)
	}
}
