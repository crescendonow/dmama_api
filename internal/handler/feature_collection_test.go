package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"dmama_api/internal/model"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCreateStepTestFeatureCollectionCreatesEveryFeatureInOrder(t *testing.T) {
	app := fiber.New()
	creator := &recordingFeatureCreator{}
	h := &FeatureHandler{creator: creator}
	app.Post("/api/features/:shape/:pwaCode", h.Create)

	body := `{
		"type":"FeatureCollection",
		"features":[
			{"type":"Feature","id":"client-one","geometry":{"type":"Polygon","coordinates":[]},"properties":{"name":"one","_id":"client-one"}},
			{"type":"Feature","id":"client-two","geometry":{"type":"Polygon","coordinates":[]},"properties":{"name":"two","_id":"client-two"}}
		]
	}`
	req := httptest.NewRequest("POST", "/api/features/step_test/5521040", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	if got, want := creator.names, []string{"one", "two"}; !equalStrings(got, want) {
		t.Fatalf("create order = %v, want %v", got, want)
	}

	var response struct {
		Success bool                    `json:"success"`
		Count   int                     `json:"count"`
		Data    model.FeatureCollection `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.Count != 2 {
		t.Fatalf("response success/count = %t/%d, want true/2", response.Success, response.Count)
	}
	if response.Data.Type != "FeatureCollection" || len(response.Data.Features) != 2 {
		t.Fatalf("response data = %#v, want two-feature collection", response.Data)
	}
	for i, feature := range response.Data.Features {
		if feature.ID.IsZero() {
			t.Fatalf("response feature %d did not include server-created id", i)
		}
	}
}

func TestCreateStepTestFeatureCollectionRejectsInvalidEnvelopeBeforeCreate(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "wrong collection type",
			body: `{"type":"Feature","features":[{"type":"Feature","geometry":{"type":"Polygon","coordinates":[]},"properties":{"name":"one"}}]}`,
		},
		{
			name: "empty features",
			body: `{"type":"FeatureCollection","features":[]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			creator := &recordingFeatureCreator{}
			h := &FeatureHandler{creator: creator}
			app.Post("/api/features/:shape/:pwaCode", h.Create)

			req := httptest.NewRequest("POST", "/api/features/step_test/5521040", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test returned error: %v", err)
			}
			if resp.StatusCode != fiber.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
			if len(creator.names) != 0 {
				t.Fatalf("create calls = %v, want none", creator.names)
			}
		})
	}
}

func TestCreateStepTestFeatureCollectionStopsAtTopologyFailureAndReportsIndex(t *testing.T) {
	app := fiber.New()
	creator := &recordingFeatureCreator{responses: []createResponse{
		{
			feature: &model.Feature{ID: primitive.NewObjectID(), Type: "Feature"},
			result:  &model.ValidationResult{Valid: true},
		},
		{
			result: &model.ValidationResult{
				Valid:      false,
				Violations: []string{"invalid geometry: self-intersection"},
			},
		},
	}}
	h := &FeatureHandler{creator: creator}
	app.Post("/api/features/:shape/:pwaCode", h.Create)

	body := `{
		"type":"FeatureCollection",
		"features":[
			{"type":"Feature","geometry":{"type":"Polygon","coordinates":[]},"properties":{"name":"first"}},
			{"type":"Feature","geometry":{"type":"Polygon","coordinates":[]},"properties":{"name":"second"}},
			{"type":"Feature","geometry":{"type":"Polygon","coordinates":[]},"properties":{"name":"third"}}
		]
	}`
	req := httptest.NewRequest("POST", "/api/features/step_test/5521040", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if got, want := creator.names, []string{"first", "second"}; !equalStrings(got, want) {
		t.Fatalf("create calls = %v, want %v", got, want)
	}

	var response model.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Success {
		t.Fatal("expected success=false")
	}
	if !strings.Contains(response.Error, "index 1") {
		t.Fatalf("error = %q, want failing index 1", response.Error)
	}
}

func TestCreateStepTestFeatureCollectionParsesMembersInOrderAndReportsMalformedIndex(t *testing.T) {
	app := fiber.New()
	creator := &recordingFeatureCreator{}
	h := &FeatureHandler{creator: creator}
	app.Post("/api/features/:shape/:pwaCode", h.Create)

	body := `{
		"type":"FeatureCollection",
		"features":[
			{"type":"Feature","geometry":{"type":"Polygon","coordinates":[]},"properties":{"name":"first"}},
			{"type":"Feature","geometry":{"type":"Polygon","coordinates":[]},"properties":"not-an-object"},
			{"type":"Feature","geometry":{"type":"Polygon","coordinates":[]},"properties":{"name":"third"}}
		]
	}`
	req := httptest.NewRequest("POST", "/api/features/step_test/5521040", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if got, want := creator.names, []string{"first"}; !equalStrings(got, want) {
		t.Fatalf("create calls = %v, want %v", got, want)
	}

	var response model.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !strings.Contains(response.Error, "index 1") {
		t.Fatalf("error = %q, want malformed index 1", response.Error)
	}
}

func TestCreateNonStepTestStillAcceptsSingleFeatureRequest(t *testing.T) {
	app := fiber.New()
	creator := &recordingFeatureCreator{}
	h := &FeatureHandler{creator: creator}
	app.Post("/api/features/:shape/:pwaCode", h.Create)

	body := `{"geometry":{"type":"Polygon","coordinates":[]},"properties":{"name":"legacy boundary"}}`
	req := httptest.NewRequest("POST", "/api/features/dma_boundary/5521040", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	if got, want := creator.names, []string{"legacy boundary"}; !equalStrings(got, want) {
		t.Fatalf("create calls = %v, want %v", got, want)
	}
}

type recordingFeatureCreator struct {
	names     []string
	responses []createResponse
}

type createResponse struct {
	feature *model.Feature
	result  *model.ValidationResult
	err     error
}

func (c *recordingFeatureCreator) Create(_ context.Context, shape, pwaCode string, req *model.FeatureRequest, _ string) (*model.Feature, *model.ValidationResult, error) {
	name, _ := req.Properties["name"].(string)
	c.names = append(c.names, name)
	if len(c.responses) > 0 {
		response := c.responses[0]
		c.responses = c.responses[1:]
		return response.feature, response.result, response.err
	}
	return &model.Feature{ID: primitive.NewObjectID(), Type: "Feature", Geometry: req.Geometry, Properties: req.Properties}, &model.ValidationResult{Valid: true}, nil
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
