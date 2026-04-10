package handler

import (
	"encoding/json"
	"strconv"

	"dmama_api/internal/model"
	"dmama_api/internal/repository"
	"dmama_api/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AssetHandler struct {
	assetService *service.AssetService
}

func NewAssetHandler(pool *pgxpool.Pool) *AssetHandler {
	return &AssetHandler{
		assetService: service.NewAssetService(pool),
	}
}

// FindNearest finds nearest asset within 20m. (Endpoint 10)
// GET /api/asset/nearest?region=1&lng=100.5&lat=13.7&type=pipe&format=geojson
func (h *AssetHandler) FindNearest(c *fiber.Ctx) error {
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

	assetType := c.Query("type", "pipe")
	if assetType != "pipe" && assetType != "meter" && assetType != "valve" && assetType != "firehydrant" && assetType != "leakpoint" {
		return c.Status(400).JSON(model.ErrorResponse("type must be pipe, meter, valve, firehydrant, or leakpoint"))
	}

	format := model.ParseGeomFormat(c.Query("format"))
	result, err := h.assetService.FindNearest(c.Context(), region, lng, lat, 20, assetType, format)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	if result == nil {
		return c.JSON(model.SuccessResponse(nil))
	}
	return c.JSON(model.SuccessResponse(result))
}

// InArea counts assets within a drawn area. (Endpoint 13)
// POST /api/asset/in-area
func (h *AssetHandler) InArea(c *fiber.Ctx) error {
	type request struct {
		Geometry json.RawMessage `json:"geometry"`
		Buffer   float64         `json:"buffer"`
		Region   *int            `json:"region"`
		GeomType string          `json:"geom_type"` // "Polygon" or "LineString"
	}

	var req request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(model.ErrorResponse("invalid request body"))
	}

	if len(req.Geometry) == 0 {
		return c.Status(400).JSON(model.ErrorResponse("geometry is required"))
	}

	if req.GeomType == "" {
		req.GeomType = "Polygon"
	}

	regions := repository.ResolveRegions(req.Region)
	result, err := h.assetService.CountAssetsInArea(c.Context(), regions, string(req.Geometry), req.Buffer, req.GeomType)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	return c.JSON(model.SuccessResponse(result))
}
