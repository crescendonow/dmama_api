package service

import (
	"context"
	"strings"
	"testing"

	"dmama_api/internal/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestValidateStepTestRejectsOverlapWithExistingMirror(t *testing.T) {
	topology := &existingOverlapTopology{}
	svc := &FeatureService{topology: topology}

	result, err := svc.Validate(
		context.Background(),
		model.ShapeStepTest,
		"5541022",
		&model.FeatureRequest{Geometry: squarePolygon(0, 0, 2, 2)},
		primitive.NilObjectID,
	)
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected overlap with existing step_test to be invalid: %#v", result)
	}
	if got := strings.Join(result.Violations, "; "); !strings.Contains(got, "existing step_test") {
		t.Fatalf("violations = %q, want existing step_test overlap", got)
	}
	if len(topology.shapes) != 1 || topology.shapes[0] != model.ShapeStepTest {
		t.Fatalf("overlap checks = %v, want [%s]", topology.shapes, model.ShapeStepTest)
	}
}

type existingOverlapTopology struct {
	recordingFeatureTopology
	shapes []string
}

func (t *existingOverlapTopology) OverlapsExisting(_ context.Context, shape, _, _, _ string) (bool, error) {
	t.shapes = append(t.shapes, shape)
	return true, nil
}
