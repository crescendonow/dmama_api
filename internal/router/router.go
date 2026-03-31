package router

import (
	"dmama_api/internal/config"
	"dmama_api/internal/handler"
	"dmama_api/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Setup(app *fiber.App, pool *pgxpool.Pool, cfg *config.Config) {
	// API documentation
	app.Static("/template", "./template")
	app.Static("/docs/static", "./template/static")
	app.Get("/docs", func(c *fiber.Ctx) error {
		return c.SendFile("./template/index.html")
	})

	api := app.Group("/api")

	// Health check (no API key required)
	api.Get("/health", handler.HealthCheck)

	// API key authentication for all routes below
	api.Use(middleware.APIKeyAuth(cfg))

	// Pipe endpoints (1, 2, 11, 12)
	pipeH := handler.NewPipeHandler(pool)
	api.Post("/pipe/assign-logger", pipeH.AssignLogger)
	api.Get("/pipe/geometry", pipeH.GetGeometry)
	api.Get("/pipe/at-leakpoint", pipeH.AtLeakpoint)
	api.Get("/pipe/assets", pipeH.AssetsOnPipe)

	// DMA endpoints (3, 4, 5, 6, 7, 14, 15, 16)
	dmaH := handler.NewDMAHandler(pool)
	api.Get("/dma/boundary", dmaH.GetBoundary)
	api.Get("/dma/usage", dmaH.GetUsage)
	api.Get("/dma/population", dmaH.GetPopulation)
	api.Get("/dma/pipe-length", dmaH.GetPipeLength)
	api.Get("/dma/map", dmaH.GetMapData)
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
}
