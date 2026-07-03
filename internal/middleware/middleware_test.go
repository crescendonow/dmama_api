package middleware

import (
	"net/http/httptest"
	"testing"

	"dmama_api/internal/config"

	"github.com/gofiber/fiber/v2"
)

func TestAPIKeyAuthRejectsMissingKeyBeforeDownstreamMiddleware(t *testing.T) {
	app := fiber.New()
	downstreamCalled := false

	app.Use(APIKeyAuth(&config.Config{DmamaKey: "secret"}))
	app.Use(func(c *fiber.Ctx) error {
		downstreamCalled = true
		return c.Next()
	})
	app.Get("/api/dma/stats", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	req := httptest.NewRequest("GET", "/api/dma/stats?pwa_code=5521027&dma_id=11&column=prswtusg", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected missing API key to return 401, got %d", resp.StatusCode)
	}
	if downstreamCalled {
		t.Fatal("expected missing API key to stop before downstream middleware")
	}
}

func TestAPIKeyAuthAllowsValidKeyThroughDownstreamMiddleware(t *testing.T) {
	app := fiber.New()
	downstreamCalled := false

	app.Use(APIKeyAuth(&config.Config{DmamaKey: "secret"}))
	app.Use(func(c *fiber.Ctx) error {
		downstreamCalled = true
		return c.Next()
	})
	app.Get("/api/dma/stats", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	req := httptest.NewRequest("GET", "/api/dma/stats?pwa_code=5521027&dma_id=11&column=prswtusg", nil)
	req.Header.Set("X-API-Key", "secret")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected valid API key to continue to route, got %d", resp.StatusCode)
	}
	if !downstreamCalled {
		t.Fatal("expected valid API key to continue through downstream middleware")
	}
}
