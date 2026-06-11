package handler

import (
	"strconv"
	"strings"
	"time"

	"dmama_api/internal/model"
	"dmama_api/internal/repository"
	"dmama_api/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DMAHandler struct {
	dmaRepo    *repository.DMARepo
	dmaService *service.DMAService
	pipeRepo   *repository.PipeRepo
	leakRepo   *repository.LeakpointRepo
}

func NewDMAHandler(pool *pgxpool.Pool) *DMAHandler {
	dmaRepo := repository.NewDMARepo(pool)
	customerRepo := repository.NewCustomerRepo(pool)
	meterRepo := repository.NewMeterRepo(pool)
	pipeRepo := repository.NewPipeRepo(pool)
	leakRepo := repository.NewLeakpointRepo(pool)
	return &DMAHandler{
		dmaRepo:    dmaRepo,
		dmaService: service.NewDMAService(dmaRepo, customerRepo, meterRepo, pipeRepo, leakRepo),
		pipeRepo:   pipeRepo,
		leakRepo:   leakRepo,
	}
}

// resolveRegion resolves region from query params: either region or pwa_code.
func resolveRegion(c *fiber.Ctx) (int, string, error) {
	pwaCode := c.Query("pwa_code")
	regionStr := c.Query("region")

	if regionStr != "" {
		region, err := strconv.Atoi(regionStr)
		if err != nil || !repository.ValidRegion(region) {
			return 0, pwaCode, fiber.NewError(400, "region must be 1-10")
		}
		return region, pwaCode, nil
	}

	if pwaCode != "" {
		region, err := repository.RegionFromPWACode(pwaCode)
		if err != nil {
			return 0, pwaCode, fiber.NewError(400, err.Error())
		}
		return region, pwaCode, nil
	}

	return 0, "", fiber.NewError(400, "region or pwa_code is required")
}

// AtLeakpoint finds the DMA boundary containing a leakpoint coordinate.
// GET /api/dma/at-leakpoint?lng=101.6893&lat=13.9849&pwa_code=5531011
func (h *DMAHandler) AtLeakpoint(c *fiber.Ctx) error {
	lng, err := strconv.ParseFloat(c.Query("lng"), 64)
	if err != nil {
		return c.Status(400).JSON(model.ErrorResponse("lng is required"))
	}

	lat, err := strconv.ParseFloat(c.Query("lat"), 64)
	if err != nil {
		return c.Status(400).JSON(model.ErrorResponse("lat is required"))
	}

	result, err := h.dmaRepo.GetAtLeakpoint(c.Context(), lng, lat, c.Query("pwa_code"))
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	if result == nil {
		return c.JSON(model.SuccessResponse(nil))
	}
	return c.JSON(model.SuccessResponse(result))
}

// GetBoundary fetches a DMA boundary. (Endpoint 3)
// GET /api/dma/boundary?pwa_code=5531011&dma_id=1&format=geojson
func (h *DMAHandler) GetBoundary(c *fiber.Ctx) error {
	pwaCode := c.Query("pwa_code")
	dmaID := c.Query("dma_id")
	if pwaCode == "" || dmaID == "" {
		return c.Status(400).JSON(model.ErrorResponse("pwa_code and dma_id are required"))
	}

	format := model.ParseGeomFormat(c.Query("format"))
	result, err := h.dmaRepo.GetBoundary(c.Context(), pwaCode, dmaID, format)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	if result == nil {
		return c.Status(404).JSON(model.ErrorResponse("DMA boundary not found"))
	}
	return c.JSON(model.SuccessResponse(result))
}

// GetUsage calculates usage within a DMA. (Endpoint 4)
// GET /api/dma/usage?pwa_code=5531011&dma_id=1&column=prswtusg&region=1
func (h *DMAHandler) GetUsage(c *fiber.Ctx) error {
	region, pwaCode, err := resolveRegion(c)
	if err != nil {
		return c.Status(400).JSON(model.ErrorResponse(err.Error()))
	}

	dmaID := c.Query("dma_id")
	column := c.Query("column", "prswtusg")

	if pwaCode == "" || dmaID == "" {
		return c.Status(400).JSON(model.ErrorResponse("pwa_code and dma_id are required"))
	}

	result, err := h.dmaService.GetUsage(c.Context(), pwaCode, dmaID, column, region)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	return c.JSON(model.SuccessResponse(result))
}

// GetPopulation counts population within a DMA. (Endpoint 5)
// GET /api/dma/population?pwa_code=5531011&dma_id=1&column=prswtusg&region=1
func (h *DMAHandler) GetPopulation(c *fiber.Ctx) error {
	region, pwaCode, err := resolveRegion(c)
	if err != nil {
		return c.Status(400).JSON(model.ErrorResponse(err.Error()))
	}

	dmaID := c.Query("dma_id")
	column := c.Query("column", "prswtusg")

	if pwaCode == "" || dmaID == "" {
		return c.Status(400).JSON(model.ErrorResponse("pwa_code and dma_id are required"))
	}

	result, err := h.dmaService.GetPopulation(c.Context(), pwaCode, dmaID, column, region)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	return c.JSON(model.SuccessResponse(result))
}

// GetStats returns merged usage and population statistics for a DMA. (Endpoint stats)
// GET /api/dma/stats?pwa_code=5531011&dma_id=1&column=prswtusg
func (h *DMAHandler) GetStats(c *fiber.Ctx) error {
	region, pwaCode, err := resolveRegion(c)
	if err != nil {
		return c.Status(400).JSON(model.ErrorResponse(err.Error()))
	}

	dmaID := c.Query("dma_id")
	if pwaCode == "" || dmaID == "" {
		return c.Status(400).JSON(model.ErrorResponse("pwa_code and dma_id are required"))
	}

	column, err := service.ResolveStatsColumn(c.Query("year"), c.Query("month"), c.Query("column"), time.Now())
	if err != nil {
		return c.Status(400).JSON(model.ErrorResponse(err.Error()))
	}

	result, err := h.dmaService.GetStats(c.Context(), pwaCode, dmaID, column, region)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	if result == nil {
		return c.Status(404).JSON(model.ErrorResponse("DMA not found for pwa_code=" + pwaCode + ", dma_id=" + dmaID))
	}
	return c.JSON(model.SuccessResponse(result))
}

// GetDailyMeterCount counts active meters within a DMA. The optional column query param is ignored.
// GET /api/dma/daily_metercount?pwa_code=5531011&dma_id=1&column=prswtusg
func (h *DMAHandler) GetDailyMeterCount(c *fiber.Ctx) error {
	region, pwaCode, err := resolveRegion(c)
	if err != nil {
		return c.Status(400).JSON(model.ErrorResponse(err.Error()))
	}

	dmaID := c.Query("dma_id")
	if pwaCode == "" || dmaID == "" {
		return c.Status(400).JSON(model.ErrorResponse("pwa_code and dma_id are required"))
	}

	result, err := h.dmaService.GetDailyMeterCount(c.Context(), pwaCode, dmaID, region)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	return c.JSON(model.SuccessResponse(result))
}

// GetPipeLength calculates pipe length within a DMA. (Endpoint 6)
// GET /api/dma/pipe-length?pwa_code=5531011&dma_id=1&region=1
func (h *DMAHandler) GetPipeLength(c *fiber.Ctx) error {
	region, pwaCode, err := resolveRegion(c)
	if err != nil {
		return c.Status(400).JSON(model.ErrorResponse(err.Error()))
	}

	dmaID := c.Query("dma_id")
	if pwaCode == "" || dmaID == "" {
		return c.Status(400).JSON(model.ErrorResponse("pwa_code and dma_id are required"))
	}

	result, err := h.dmaService.GetPipeLength(c.Context(), pwaCode, dmaID, region)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	return c.JSON(model.SuccessResponse(result))
}

// GetMapData fetches DMA boundaries for map display. (Endpoint 7)
// GET /api/dma/map?pwa_code=5531011&ids=5531011-1,5531011-2&format=geojson
func (h *DMAHandler) GetMapData(c *fiber.Ctx) error {
	format := model.ParseGeomFormat(c.Query("format"))

	var pwaCode *string
	if p := c.Query("pwa_code"); p != "" {
		pwaCode = &p
	}

	var ids []string
	if idsStr := c.Query("ids"); idsStr != "" {
		ids = strings.Split(idsStr, ",")
	}

	items, err := h.dmaRepo.GetMapData(c.Context(), format, pwaCode, ids)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	count := len(items)
	return c.JSON(model.SuccessWithCount(items, count))
}

// GetUsageV2 fetches DMA usage via direct spatial join. (Endpoint 14)
// GET /api/dma/usage-v2?pwa_code=5531011&dma_id=1&region=1
func (h *DMAHandler) GetUsageV2(c *fiber.Ctx) error {
	region, pwaCode, err := resolveRegion(c)
	if err != nil {
		return c.Status(400).JSON(model.ErrorResponse(err.Error()))
	}

	dmaID := c.Query("dma_id")
	if pwaCode == "" || dmaID == "" {
		return c.Status(400).JSON(model.ErrorResponse("pwa_code and dma_id are required"))
	}

	result, err := h.dmaRepo.GetUsageV2(c.Context(), region, pwaCode, dmaID)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	if result == nil {
		return c.Status(404).JSON(model.ErrorResponse("no data found"))
	}
	return c.JSON(model.SuccessResponse(result))
}

// LeakpointsBySize counts leakpoints by pipe size in DMA. (Endpoint 15)
// GET /api/dma/leakpoints-by-size?pwa_code=5531011&dma_id=1&region=1
func (h *DMAHandler) LeakpointsBySize(c *fiber.Ctx) error {
	region, pwaCode, err := resolveRegion(c)
	if err != nil {
		return c.Status(400).JSON(model.ErrorResponse(err.Error()))
	}

	dmaID := c.Query("dma_id")
	if pwaCode == "" || dmaID == "" {
		return c.Status(400).JSON(model.ErrorResponse("pwa_code and dma_id are required"))
	}

	result, err := h.leakRepo.GetLeakpointsBySizeInDMA(c.Context(), region, pwaCode, dmaID)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	return c.JSON(model.SuccessResponse(result))
}

// PipeLengthClipped calculates clipped pipe length in DMA. (Endpoint 16)
// GET /api/dma/pipe-length-clipped?pwa_code=5531011&dma_id=1&region=1
func (h *DMAHandler) PipeLengthClipped(c *fiber.Ctx) error {
	region, pwaCode, err := resolveRegion(c)
	if err != nil {
		return c.Status(400).JSON(model.ErrorResponse(err.Error()))
	}

	dmaID := c.Query("dma_id")
	if pwaCode == "" || dmaID == "" {
		return c.Status(400).JSON(model.ErrorResponse("pwa_code and dma_id are required"))
	}

	result, err := h.pipeRepo.ClippedPipeLengthInDMA(c.Context(), region, pwaCode, dmaID)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	return c.JSON(model.SuccessResponse(result))
}
