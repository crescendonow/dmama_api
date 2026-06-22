package tests

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"dmama_api/internal/database"
	"dmama_api/internal/model"
	"dmama_api/internal/repository"
	"dmama_api/internal/service"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestStepTestPolygonValidationIntegration(t *testing.T) {
	gisURL := os.Getenv("GISDATA_URL")
	mongoURI := os.Getenv("MONGO_URI")
	if gisURL == "" || mongoURI == "" {
		t.Skip("set GISDATA_URL and MONGO_URI to run step_test integration validation")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, gisURL)
	if err != nil {
		t.Fatalf("connect GISDATA_URL: %v", err)
	}
	defer pool.Close()

	svc := service.NewFeatureService(nil, repository.NewTopologyRepo(pool), nil, os.Getenv("SYSTEM_USER_ID"), "pwa.co.th")

	t.Run("fixture polygon validates within coverage when mirror exists", func(t *testing.T) {
		result, err := svc.Validate(ctx, model.ShapeStepTest, "5521040", loadStepTestRequest(t), primitive.NilObjectID)
		if err != nil {
			t.Fatalf("validate fixture: %v", err)
		}
		if noCoverageWarning(result) {
			t.Skip("branch 5521040 has no dma_boundary mirror; coverage rule produced warning as designed")
		}
		if !result.Valid {
			t.Fatalf("fixture should validate: violations=%v warnings=%v", result.Violations, result.Warnings)
		}
	})

	t.Run("self intersect polygon fails", func(t *testing.T) {
		result, err := svc.Validate(ctx, model.ShapeStepTest, "5521040", &model.FeatureRequest{Geometry: selfIntersectPolygon()}, primitive.NilObjectID)
		if err != nil {
			t.Fatalf("validate self-intersect polygon: %v", err)
		}
		if result.Valid {
			t.Fatalf("self-intersect polygon should fail validation: warnings=%v", result.Warnings)
		}
	})

	t.Run("outside coverage fails when mirror exists", func(t *testing.T) {
		result, err := svc.Validate(ctx, model.ShapeStepTest, "5521040", &model.FeatureRequest{Geometry: farAwayPolygon()}, primitive.NilObjectID)
		if err != nil {
			t.Fatalf("validate outside polygon: %v", err)
		}
		if noCoverageWarning(result) {
			t.Skip("branch 5521040 has no dma_boundary mirror; coverage rule produced warning as designed")
		}
		if result.Valid {
			t.Fatalf("outside polygon should fail when coverage exists: warnings=%v", result.Warnings)
		}
	})

	t.Run("no boundary mirror warns without blocking", func(t *testing.T) {
		result, err := svc.Validate(ctx, model.ShapeStepTest, "0000000", &model.FeatureRequest{Geometry: farAwayPolygon()}, primitive.NilObjectID)
		if err != nil {
			t.Fatalf("validate no-coverage branch: %v", err)
		}
		if !noCoverageWarning(result) {
			t.Fatalf("expected no-coverage warning, got violations=%v warnings=%v", result.Violations, result.Warnings)
		}
		if !result.Valid {
			t.Fatalf("no coverage should warn, not block: violations=%v", result.Violations)
		}
	})
}

func loadStepTestRequest(t *testing.T) *model.FeatureRequest {
	t.Helper()
	path := filepath.Join("..", "note", "sample_data", "step_test.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read step_test fixture: %v", err)
	}
	jsonish := mongoExportToJSON(string(raw))

	var fixture struct {
		Geometry   map[string]interface{} `json:"geometry"`
		Properties map[string]interface{} `json:"properties"`
	}
	if err := json.Unmarshal([]byte(jsonish), &fixture); err != nil {
		t.Fatalf("parse step_test fixture: %v", err)
	}

	props := make(map[string]interface{}, len(fixture.Properties))
	for k, v := range fixture.Properties {
		switch k {
		case "_id", "_createdAt", "_createdBy", "_updatedAt", "_updatedBy":
			continue
		default:
			props[k] = v
		}
	}
	props["pwaCode"] = "5521040"
	return &model.FeatureRequest{Geometry: fixture.Geometry, Properties: props}
}

func mongoExportToJSON(s string) string {
	s = regexp.MustCompile(`ObjectId\("([0-9a-fA-F]{24})"\)`).ReplaceAllString(s, `"$1"`)
	s = regexp.MustCompile(`ISODate\("([^\"]+)"\)`).ReplaceAllString(s, `"$1"`)
	s = regexp.MustCompile(`NumberInt\((-?\d+)\)`).ReplaceAllString(s, `$1`)
	return s
}

func selfIntersectPolygon() map[string]interface{} {
	return map[string]interface{}{
		"type": "Polygon",
		"coordinates": [][][]float64{{
			{0, 0}, {1, 1}, {1, 0}, {0, 1}, {0, 0},
		}},
	}
}

func farAwayPolygon() map[string]interface{} {
	return map[string]interface{}{
		"type": "Polygon",
		"coordinates": [][][]float64{{
			{-120, 35}, {-119.99, 35}, {-119.99, 35.01}, {-120, 35.01}, {-120, 35},
		}},
	}
}

func noCoverageWarning(result *model.ValidationResult) bool {
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "no dma_boundary mirrored") {
			return true
		}
	}
	return false
}
