package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"dmama_api/internal/database"
	"dmama_api/internal/model"
	"dmama_api/internal/repository"
	"dmama_api/internal/service"
)

func TestStepTestCollectionBoundaryTouchIsAllowedIntegration(t *testing.T) {
	gisURL := os.Getenv("GISDATA_URL")
	if gisURL == "" {
		t.Skip("set GISDATA_URL to run step_test collection boundary-touch integration")
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
		{Geometry: integrationSquare(100.02, 14.0, 100.04, 14.02)},
	}
	result, err := svc.ValidateStepTestCollection(ctx, "0000000", requests)
	if err != nil {
		t.Fatalf("ValidateStepTestCollection returned error: %v", err)
	}
	if !result.Valid {
		t.Fatalf("boundary-touching collection should be valid: violations=%v", result.Violations)
	}
}
