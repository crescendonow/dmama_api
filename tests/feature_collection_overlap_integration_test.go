package tests

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"dmama_api/internal/database"
	"dmama_api/internal/model"
	"dmama_api/internal/repository"
	"dmama_api/internal/service"
)

func TestStepTestCollectionOverlapIntegration(t *testing.T) {
	gisURL := os.Getenv("GISDATA_URL")
	if gisURL == "" {
		t.Skip("set GISDATA_URL to run step_test collection overlap integration")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := database.NewPool(ctx, gisURL)
	if err != nil {
		t.Fatalf("connect GISDATA_URL: %v", err)
	}
	defer pool.Close()

	svc := service.NewFeatureService(nil, repository.NewTopologyRepo(pool), nil, "", "")
	requests := []model.FeatureRequest{
		{Geometry: integrationSquare(100.0, 14.0, 100.02, 14.02)},
		{Geometry: integrationSquare(100.01, 14.01, 100.03, 14.03)},
	}
	result, err := svc.ValidateStepTestCollection(ctx, "0000000", requests)
	if err != nil {
		t.Fatalf("ValidateStepTestCollection returned error: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected overlapping collection to be invalid: %#v", result)
	}
	if got := strings.Join(result.Violations, "; "); !strings.Contains(got, "index 1 overlaps feature at index 0") {
		t.Fatalf("violations = %q, want collection overlap indexes", got)
	}
}

func integrationSquare(minX, minY, maxX, maxY float64) map[string]interface{} {
	return map[string]interface{}{
		"type": "Polygon",
		"coordinates": [][][]float64{{
			{minX, minY},
			{maxX, minY},
			{maxX, maxY},
			{minX, maxY},
			{minX, minY},
		}},
	}
}
