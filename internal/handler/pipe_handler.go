package handler

import (
	"strconv"
	"strings"

	"dmama_api/internal/model"
	"dmama_api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PipeHandler struct {
	pipeRepo *repository.PipeRepo
}

func NewPipeHandler(pool *pgxpool.Pool) *PipeHandler {
	return &PipeHandler{
		pipeRepo: repository.NewPipeRepo(pool),
	}
}

// AssignLogger finds nearest pipe within 20m of GPS. (Endpoint 1)
// POST /api/pipe/assign-logger
func (h *PipeHandler) AssignLogger(c *fiber.Ctx) error {
	type request struct {
		Lat    float64 `json:"lat"`
		Lng    float64 `json:"lng"`
		Region int     `json:"region"`
	}
	var req request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(model.ErrorResponse("invalid request body"))
	}
	if !repository.ValidRegion(req.Region) {
		return c.Status(400).JSON(model.ErrorResponse("region must be 1-10"))
	}

	result, err := h.pipeRepo.FindNearestPipe(c.Context(), req.Region, req.Lng, req.Lat, 20)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	if result == nil {
		return c.JSON(model.SuccessResponse(nil))
	}
	return c.JSON(model.SuccessResponse(result))
}

// GetGeometry fetches pipe geometries by pipe_id and/or pwa_code. (Endpoint 2)
// GET /api/pipe/geometry?region=1&pipe_id=PIPE001,PIPE002&pwa_code=5531&format=geojson
func (h *PipeHandler) GetGeometry(c *fiber.Ctx) error {
	region, err := strconv.Atoi(c.Query("region"))
	if err != nil || !repository.ValidRegion(region) {
		return c.Status(400).JSON(model.ErrorResponse("region must be 1-10"))
	}

	pipeIDStr := c.Query("pipe_id")
	pwaCode := c.Query("pwa_code")
	if pipeIDStr == "" && pwaCode == "" {
		return c.Status(400).JSON(model.ErrorResponse("pipe_id or pwa_code is required"))
	}

	var pipeIDs []string
	if pipeIDStr != "" {
		for _, p := range strings.Split(pipeIDStr, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				pipeIDs = append(pipeIDs, p)
			}
		}
	}

	format := model.ParseGeomFormat(c.Query("format"))
	pipes, err := h.pipeRepo.GetGeometry(c.Context(), region, pipeIDs, pwaCode, format)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	return c.JSON(model.SuccessResponse(pipes))
}

// AtLeakpoint finds pipe at a leakpoint's geometry. (Endpoint 11)
// GET /api/pipe/at-leakpoint?region=1&lng=101.6893&lat=13.9849
func (h *PipeHandler) AtLeakpoint(c *fiber.Ctx) error {
	region, err := strconv.Atoi(c.Query("region"))
	if err != nil || !repository.ValidRegion(region) {
		return c.Status(400).JSON(model.ErrorResponse("region must be 1-10"))
	}

	lng, err := strconv.ParseFloat(c.Query("lng"), 64)
	if err != nil {
		return c.Status(400).JSON(model.ErrorResponse("lng is required"))
	}
	lat, err := strconv.ParseFloat(c.Query("lat"), 64)
	if err != nil {
		return c.Status(400).JSON(model.ErrorResponse("lat is required"))
	}

	result, err := h.pipeRepo.GetPipeAtLeakpoint(c.Context(), region, lng, lat)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	if result == nil {
		return c.JSON(model.SuccessResponse(nil))
	}
	return c.JSON(model.SuccessResponse(result))
}

// AssetsOnPipe finds assets on a pipe's geometry. (Endpoint 12)
// GET /api/pipe/assets?region=1&wkb_geometry=...&asset_type=meter&format=geojson
func (h *PipeHandler) AssetsOnPipe(c *fiber.Ctx) error {
	region, err := strconv.Atoi(c.Query("region"))
	if err != nil || !repository.ValidRegion(region) {
		return c.Status(400).JSON(model.ErrorResponse("region must be 1-10"))
	}

	wkbGeometry := c.Query("wkb_geometry")
	if wkbGeometry == "" {
		return c.Status(400).JSON(model.ErrorResponse("wkb_geometry is required"))
	}

	assetType := c.Query("asset_type")
	if assetType != "meter" && assetType != "valve" && assetType != "leakpoint" {
		return c.Status(400).JSON(model.ErrorResponse("asset_type must be meter, valve, or leakpoint"))
	}

	format := model.ParseGeomFormat(c.Query("format"))
	results, err := h.pipeRepo.GetAssetsOnPipe(c.Context(), region, wkbGeometry, assetType, format)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	return c.JSON(model.SuccessResponse(results))
}
