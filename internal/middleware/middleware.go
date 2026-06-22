package middleware

import (
	"crypto/subtle"
	"dmama_api/internal/config"
	"dmama_api/internal/model"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func Setup(app *fiber.App, cfg *config.Config) {
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "${time} | ${status} | ${latency} | ${method} | ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOrigins,
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Content-Type,Authorization,X-API-Key,X-User-Id",
	}))
}

// APIKeyAuth validates the X-API-Key header against the configured DMAMA_KEY.
// If DMAMA_KEY is empty, authentication is skipped (dev mode).
func APIKeyAuth(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg.DmamaKey == "" {
			return c.Next()
		}

		key := c.Get("X-API-Key")
		if subtle.ConstantTimeCompare([]byte(key), []byte(cfg.DmamaKey)) != 1 {
			return c.Status(401).JSON(model.ErrorResponse("invalid or missing API key"))
		}

		return c.Next()
	}
}
