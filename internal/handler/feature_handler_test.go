package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// These paths reject before any service/DB access, so a nil service is safe.

func TestCreateRejectsInvalidBody(t *testing.T) {
	app := fiber.New()
	h := &FeatureHandler{svc: nil}
	app.Post("/api/features/:shape/:pwaCode", h.Create)

	req := httptest.NewRequest("POST", "/api/features/dma_boundary/5521040", strings.NewReader("{not json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateRejectsInvalidShape(t *testing.T) {
	app := fiber.New()
	h := &FeatureHandler{svc: nil}
	app.Post("/api/features/:shape/:pwaCode", h.Create)

	req := httptest.NewRequest("POST", "/api/features/not_a_shape/5521040", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetRejectsInvalidID(t *testing.T) {
	app := fiber.New()
	h := &FeatureHandler{svc: nil}
	app.Get("/api/features/:shape/:pwaCode/:id", h.Get)

	req := httptest.NewRequest("GET", "/api/features/dma_boundary/5521040/not-an-objectid", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestDeleteRejectsInvalidID(t *testing.T) {
	app := fiber.New()
	h := &FeatureHandler{svc: nil}
	app.Delete("/api/features/:shape/:pwaCode/:id", h.Delete)

	req := httptest.NewRequest("DELETE", "/api/features/flow_meter/5521040/xyz", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestListRejectsInvalidBBox(t *testing.T) {
	app := fiber.New()
	h := &FeatureHandler{svc: nil}
	app.Get("/api/features/:shape/:pwaCode", h.List)

	req := httptest.NewRequest("GET", "/api/features/dma_boundary/5521040?bbox=1,2,3", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestParseBBox(t *testing.T) {
	bbox, err := parseBBox("100.5, 13.7, 100.6, 13.8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []float64{100.5, 13.7, 100.6, 13.8}
	for i := range want {
		if bbox[i] != want[i] {
			t.Fatalf("bbox[%d] = %v, want %v", i, bbox[i], want[i])
		}
	}

	if _, err := parseBBox("1,2,3"); err == nil {
		t.Fatal("expected error for 3-value bbox")
	}
	if _, err := parseBBox("a,b,c,d"); err == nil {
		t.Fatal("expected error for non-numeric bbox")
	}
}
