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

func TestValidateStepTestFeatureCollectionReportsIndexedOverlap(t *testing.T) {
	validator := &recordingFeatureValidator{
		collectionResult: &model.ValidationResult{
			Valid:      false,
			Violations: []string{"feature at index 1 overlaps feature at index 0 within the collection"},
		},
	}
	h := &FeatureHandler{validator: validator}
	app := fiber.New()
	app.Post("/api/features/:shape/:pwaCode/validate", h.Validate)

	body := `{
		"type":"FeatureCollection",
		"features":[
			{"type":"Feature","id":"first","geometry":{"type":"Polygon","coordinates":[]},"properties":{"name":"first"}},
			{"type":"Feature","id":"second","geometry":{"type":"Polygon","coordinates":[]},"properties":{"name":"second"}}
		]
	}`
	req := httptest.NewRequest("POST", "/api/features/step_test/5541022/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if got, want := validator.names, []string{"first", "second"}; !equalStrings(got, want) {
		t.Fatalf("validated members = %v, want %v", got, want)
	}

	var response struct {
		Success bool                   `json:"success"`
		Data    model.ValidationResult `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.Data.Valid {
		t.Fatalf("success/valid = %t/%t, want true/false", response.Success, response.Data.Valid)
	}
	if got := strings.Join(response.Data.Violations, "; "); !strings.Contains(got, "index 1") || !strings.Contains(got, "index 0") {
		t.Fatalf("violations = %q, want both overlap indexes", got)
	}
}

type recordingFeatureValidator struct {
	names            []string
	collectionResult *model.ValidationResult
}

func (v *recordingFeatureValidator) Validate(context.Context, string, string, *model.FeatureRequest, primitive.ObjectID) (*model.ValidationResult, error) {
	return &model.ValidationResult{Valid: true}, nil
}

func (v *recordingFeatureValidator) ValidateStepTestCollection(_ context.Context, _ string, requests []model.FeatureRequest) (*model.ValidationResult, error) {
	for i := range requests {
		name, _ := requests[i].Properties["name"].(string)
		v.names = append(v.names, name)
	}
	return v.collectionResult, nil
}
