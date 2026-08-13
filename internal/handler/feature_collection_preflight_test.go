package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"dmama_api/internal/model"

	"github.com/gofiber/fiber/v2"
)

func TestCreateStepTestFeatureCollectionRejectsOverlapBeforeAnyInsert(t *testing.T) {
	creator := &recordingFeatureCreator{}
	validator := &recordingFeatureValidator{
		collectionResult: &model.ValidationResult{
			Valid:      false,
			Violations: []string{"feature at index 1 overlaps feature at index 0 within the collection"},
		},
	}
	h := &FeatureHandler{creator: creator, validator: validator}
	app := fiber.New()
	app.Post("/api/features/:shape/:pwaCode", h.Create)

	body := `{
		"type":"FeatureCollection",
		"features":[
			{"type":"Feature","geometry":{"type":"Polygon","coordinates":[]},"properties":{"name":"first"}},
			{"type":"Feature","geometry":{"type":"Polygon","coordinates":[]},"properties":{"name":"second"}}
		]
	}`
	req := httptest.NewRequest("POST", "/api/features/step_test/5541022", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if len(creator.names) != 0 {
		t.Fatalf("create calls = %v, want none before collection preflight passes", creator.names)
	}
}
