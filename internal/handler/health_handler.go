package handler

import (
	"dmama_api/internal/model"

	"github.com/gofiber/fiber/v2"
)

func HealthCheck(c *fiber.Ctx) error {
	return c.JSON(model.SuccessResponse(map[string]string{
		"status": "ok",
	}))
}
