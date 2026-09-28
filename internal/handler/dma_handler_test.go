package handler

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dmama_api/internal/model"

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

	req := httptest.NewRequest("GET", "/api/dma/stats-region?region=9&column=lstwtusg13", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
func TestGetStatsRegionRejectsPWACodeFromAnotherRegion(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Get("/api/dma/stats-region", h.GetStatsRegion)

	req := httptest.NewRequest("GET", "/api/dma/stats-region?region=1&column=prswtusg&pwa_code=5541011", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetStatsRegionForwardsOptionalPWACode(t *testing.T) {
	app := fiber.New()
	recorder := &recordingStatsRegionService{}
	h := &DMAHandler{statsRegion: recorder}
	app.Get("/api/dma/stats-region", h.GetStatsRegion)

	req := httptest.NewRequest("GET", "/api/dma/stats-region?region=1&column=prswtusg&pwa_code=5531011", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if recorder.region != 1 || recorder.column != "prswtusg" || recorder.pwaCode != "5531011" {
		t.Fatalf("forwarded region/column/pwa_code = %d/%q/%q", recorder.region, recorder.column, recorder.pwaCode)
	}
}

// ---- customers-all (GET) ----

func TestGetCustomersAllRejectsInvalidRegion(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Get("/api/dma/customers-all", h.GetCustomersAll)

	req := httptest.NewRequest("GET", "/api/dma/customers-all?region=11", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetCustomersAllRejectsUnknownPWACodePrefix(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Get("/api/dma/customers-all", h.GetCustomersAll)

	req := httptest.NewRequest("GET", "/api/dma/customers-all?pwa_code=9999011", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetCustomersAllRejectsMismatchedRegionAndPWACode(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Get("/api/dma/customers-all", h.GetCustomersAll)

	req := httptest.NewRequest("GET", "/api/dma/customers-all?region=2&pwa_code=5531011", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetCustomersAllRejectsDmaIDWithoutPWACode(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Get("/api/dma/customers-all", h.GetCustomersAll)

	req := httptest.NewRequest("GET", "/api/dma/customers-all?region=1&dma_id=1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetCustomersAllRejectsNonIntegerDmaID(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Get("/api/dma/customers-all", h.GetCustomersAll)

	req := httptest.NewRequest("GET", "/api/dma/customers-all?region=1&pwa_code=5531011&dma_id=a", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetCustomersAllRejectsInvalidUsetype(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Get("/api/dma/customers-all", h.GetCustomersAll)

	req := httptest.NewRequest("GET", "/api/dma/customers-all?usetype=2x", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// ---- customers-all (POST, polygon) ----

func TestPostCustomersAllRejectsMissingMyPolygon(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Post("/api/dma/customers-all", h.PostCustomersAll)

	req := httptest.NewRequest("POST", "/api/dma/customers-all", strings.NewReader(`{"region":1}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPostCustomersAllRejectsInvalidJSONBody(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Post("/api/dma/customers-all", h.PostCustomersAll)

	req := httptest.NewRequest("POST", "/api/dma/customers-all", strings.NewReader(`{not json`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPostCustomersAllRejectsWrongGeometryType(t *testing.T) {
	app := fiber.New()
	h := NewDMAHandler(nil)
	app.Post("/api/dma/customers-all", h.PostCustomersAll)

	req := httptest.NewRequest("POST", "/api/dma/customers-all", strings.NewReader(`{"my_polygon":{"type":"Point","coordinates":[100,13]}}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

type recordingStatsRegionService struct {
	region  int
	column  string
	pwaCode string
}

func (r *recordingStatsRegionService) GetStatsRegion(_ context.Context, region int, column, pwaCode string, _ time.Time) ([]model.DMAStats, error) {
	r.region = region
	r.column = column
	r.pwaCode = pwaCode
	return []model.DMAStats{{PwaCode: pwaCode, Column: column}}, nil
}
