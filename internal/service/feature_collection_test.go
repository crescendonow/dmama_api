package service

import (
	"context"
	"strings"
	"testing"

	"dmama_api/internal/model"
)

func TestValidateStepTestCollectionReportsOverlappingMemberIndexes(t *testing.T) {
	topology := &recordingFeatureTopology{
		collectionOverlaps: [][2]int{{0, 1}},
	}
	svc := &FeatureService{topology: topology}

	requests := []model.FeatureRequest{
		{Geometry: squarePolygon(0, 0, 2, 2)},
		{Geometry: squarePolygon(1, 1, 3, 3)},
	}
	result, err := svc.ValidateStepTestCollection(context.Background(), "5541022", requests)
	if err != nil {
		t.Fatalf("ValidateStepTestCollection returned error: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected overlapping collection to be invalid: %#v", result)
	}
	if len(result.Violations) != 1 {
		t.Fatalf("violations = %v, want one overlap violation", result.Violations)
	}
	if violation := result.Violations[0]; !strings.Contains(violation, "index 1") || !strings.Contains(violation, "index 0") {
		t.Fatalf("violation = %q, want both overlapping indexes", violation)
	}
}

type recordingFeatureTopology struct {
	collectionOverlaps [][2]int
}

func (t *recordingFeatureTopology) CheckValidity(context.Context, string) (bool, string, error) {
	return true, "Valid Geometry", nil
}

func (t *recordingFeatureTopology) OverlapsExisting(context.Context, string, string, string, string) (bool, error) {
	return false, nil
}

func (t *recordingFeatureTopology) WithinDmaCoverage(context.Context, string, string) (bool, bool, error) {
	return true, true, nil
}

func (t *recordingFeatureTopology) CollectionOverlaps(context.Context, []string) ([][2]int, error) {
	return t.collectionOverlaps, nil
}

func (t *recordingFeatureTopology) Upsert(context.Context, string, string, string, *int, string, string) error {
	return nil
}

func (t *recordingFeatureTopology) Delete(context.Context, string, string) error {
	return nil
}

func squarePolygon(minX, minY, maxX, maxY float64) map[string]interface{} {
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
