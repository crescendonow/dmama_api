package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestStepTestGeometryTypeIsPolygon(t *testing.T) {
	if got := GeometryTypeForShape[ShapeStepTest]; got != "Polygon" {
		t.Fatalf("GeometryTypeForShape[%q] = %q, want Polygon", ShapeStepTest, got)
	}
}

func TestStepTestRejectsPointByShapeMapping(t *testing.T) {
	req := FeatureRequest{Geometry: map[string]interface{}{"type": "Point"}}
	if got, _ := req.Geometry["type"].(string); got == GeometryTypeForShape[ShapeStepTest] {
		t.Fatalf("step_test Point unexpectedly matches required geometry type %q", got)
	}
}

func TestNormalizeStepTestFixture(t *testing.T) {
	fixture := loadStepTestFixture(t)
	if got, _ := fixture.Geometry["type"].(string); got != "Polygon" {
		t.Fatalf("fixture geometry type = %q, want Polygon", got)
	}

	props := normalizedStepTestProperties(fixture.Properties, "5521040")
	if got := props["pwaCode"]; got != "5521040" {
		t.Fatalf("normalized pwaCode = %v, want 5521040", got)
	}
	for _, key := range []string{"_id", "_createdAt", "_createdBy", "_updatedAt", "_updatedBy"} {
		if _, ok := props[key]; ok {
			t.Fatalf("server-managed field %s was not stripped", key)
		}
	}
	for _, key := range []string{"STEP_ID", "stepNo", "stepName", "dmaName", "dmaNo", "remark"} {
		if _, ok := props[key]; !ok {
			t.Fatalf("business field %s missing after normalization", key)
		}
	}
}

type stepTestFixture struct {
	Geometry   map[string]interface{} `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

func loadStepTestFixture(t *testing.T) stepTestFixture {
	t.Helper()
	path := filepath.Join("..", "..", "note", "sample_data", "step_test.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read step_test fixture: %v", err)
	}
	jsonish := mongoExportToJSON(string(raw))

	var fixture stepTestFixture
	if err := json.Unmarshal([]byte(jsonish), &fixture); err != nil {
		t.Fatalf("parse normalized step_test fixture: %v", err)
	}
	return fixture
}

func normalizedStepTestProperties(input map[string]interface{}, pwaCode string) map[string]interface{} {
	out := make(map[string]interface{}, len(input))
	for k, v := range input {
		switch k {
		case "_id", "_createdAt", "_createdBy", "_updatedAt", "_updatedBy":
			continue
		default:
			out[k] = v
		}
	}
	out["pwaCode"] = pwaCode
	return out
}

func mongoExportToJSON(s string) string {
	s = regexp.MustCompile(`ObjectId\("([0-9a-fA-F]{24})"\)`).ReplaceAllString(s, `"$1"`)
	s = regexp.MustCompile(`ISODate\("([^\"]+)"\)`).ReplaceAllString(s, `"$1"`)
	s = regexp.MustCompile(`NumberInt\((-?\d+)\)`).ReplaceAllString(s, `$1`)
	return s
}
