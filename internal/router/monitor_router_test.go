package router

import (
	"net/http/httptest"
	"testing"

	"dmama_api/internal/config"

	"github.com/gofiber/fiber/v2"
)

func TestSetupRegistersMonitorUsageUnavailableWhenGISPoolMissing(t *testing.T) {
	app := fiber.New()
	Setup(app, nil, nil, nil, nil, nil, false, &config.Config{})

	req := httptest.NewRequest("GET", "/api/monitor/usage", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("expected monitor route to return 503 when GIS pool is missing, got %d", resp.StatusCode)
	}
}
