package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestGetStatsRejectsMissingParams(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Get("/api/dma/stats", h.GetStats)

	req := httptest.NewRequest("GET", "/api/dma/stats?pwa_code=5531011", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetStatsRejectsInvalidRegion(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Get("/api/dma/stats", h.GetStats)

	req := httptest.NewRequest("GET", "/api/dma/stats?pwa_code=5531011&dma_id=1&region=99", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetStatsRejectsInvalidColumn(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Get("/api/dma/stats", h.GetStats)

	req := httptest.NewRequest("GET", "/api/dma/stats?pwa_code=5531011&dma_id=1&column=prswtusg;drop", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetCustomersRejectsMissingParams(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Get("/api/dma/customers", h.GetCustomers)

	req := httptest.NewRequest("GET", "/api/dma/customers?pwa_code=5531011", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetCustomersRejectsInvalidPWACode(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Get("/api/dma/customers", h.GetCustomers)

	req := httptest.NewRequest("GET", "/api/dma/customers?pwa_code=9999011&dma_id=1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetStatsRegionRejectsMissingRegion(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Get("/api/dma/stats-region", h.GetStatsRegion)

	req := httptest.NewRequest("GET", "/api/dma/stats-region?column=prswtusg", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetStatsRegionRejectsInvalidRegion(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Get("/api/dma/stats-region", h.GetStatsRegion)

	req := httptest.NewRequest("GET", "/api/dma/stats-region?region=99&column=prswtusg", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetStatsRegionRejectsInvalidColumn(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Get("/api/dma/stats-region", h.GetStatsRegion)

	req := httptest.NewRequest("GET", "/api/dma/stats-region?region=9&column=lstwtusg2", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
