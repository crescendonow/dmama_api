package handler

import (
	"dmama_api/internal/model"
	"dmama_api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OfficeHandler struct {
	officeRepo *repository.OfficeRepo
}

func NewOfficeHandler(pool *pgxpool.Pool) *OfficeHandler {
	return &OfficeHandler{
		officeRepo: repository.NewOfficeRepo(pool),
	}
}

// List fetches all PWA offices. (Endpoint 8)
// GET /api/office?format=geojson
func (h *OfficeHandler) List(c *fiber.Ctx) error {
	format := model.ParseGeomFormat(c.Query("format"))
	offices, err := h.officeRepo.GetAll(c.Context(), format)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	count := len(offices)
	return c.JSON(model.SuccessWithCount(offices, count))
}
