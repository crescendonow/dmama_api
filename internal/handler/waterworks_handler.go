package handler

import (
	"strconv"

	"dmama_api/internal/model"
	"dmama_api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WaterworksHandler struct {
	wwRepo *repository.WaterworksRepo
}

func NewWaterworksHandler(pool *pgxpool.Pool) *WaterworksHandler {
	return &WaterworksHandler{
		wwRepo: repository.NewWaterworksRepo(pool),
	}
}

// GetMarkers fetches waterworks markers by type. (Endpoint 9)
// GET /api/waterworks?type=3&format=geojson
func (h *WaterworksHandler) GetMarkers(c *fiber.Ctx) error {
	var stationType *int
	if typeStr := c.Query("type"); typeStr != "" {
		parsed, err := strconv.Atoi(typeStr)
		if err != nil {
			return c.Status(400).JSON(model.ErrorResponse("type must be an integer"))
		}
		stationType = &parsed
	}

	format := model.ParseGeomFormat(c.Query("format"))
	results, err := h.wwRepo.GetByType(c.Context(), stationType, format)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	count := len(results)
	return c.JSON(model.SuccessWithCount(results, count))
}
