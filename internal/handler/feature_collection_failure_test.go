package handler

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"dmama_api/internal/model"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCreateDmaBoundaryFeatureCollectionRejectsPreflightBeforeAnyInsert(t *testing.T) {
	app := fiber.New()
	creator := &recordingFeatureCreator{}
	h := &FeatureHandler{
		creator: creator,
		validator: &recordingFeatureValidator{collectionResult: &model.ValidationResult{
			Valid:      false,
			Violations: []string{"feature at index 1 overlaps feature at index 0 within the collection"},
		}},
	}
	app.Post("/api/features/:shape/:pwaCode", h.Create)

	body := `{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Polygon","coordinates":[]},"properties":{"name":"one"}},{"type":"Feature","geometry":{"type":"Polygon","coordinates":[]},"properties":{"name":"two"}}]}`
	req := httptest.NewRequest("POST", "/api/features/dma_boundary/5521040", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if len(creator.names) != 0 {
		t.Fatalf("create calls = %v, want none before collection preflight passes", creator.names)
	}
}

func TestCreateFlowMeterFeatureCollectionReportsPersistenceErrorIndex(t *testing.T) {
	app := fiber.New()
	creator := &recordingFeatureCreator{responses: []createResponse{
		{feature: &model.Feature{ID: primitive.NewObjectID(), Type: "Feature"}, result: &model.ValidationResult{Valid: true}},
		{err: errors.New("mongo insert failed")},
	}}
	h := &FeatureHandler{
		creator:   creator,
		validator: &recordingFeatureValidator{collectionResult: &model.ValidationResult{Valid: true}},
	}
	app.Post("/api/features/:shape/:pwaCode", h.Create)

	body := `{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Point","coordinates":[99,18]},"properties":{"name":"one"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[99.1,18.1]},"properties":{"name":"two"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[99.2,18.2]},"properties":{"name":"three"}}]}`
	req := httptest.NewRequest("POST", "/api/features/flow_meter/5521040", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	if got, want := creator.names, []string{"one", "two"}; !equalStrings(got, want) {
		t.Fatalf("create calls = %v, want %v", got, want)
	}

	var response model.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !strings.Contains(response.Error, "index 1") || !strings.Contains(response.Error, "mongo insert failed") {
		t.Fatalf("error = %q, want failing index and persistence error", response.Error)
	}
}
