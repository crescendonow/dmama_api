package tests

import (
	"testing"

	"dmama_api/internal/model"
	"dmama_api/internal/repository"
)

// These tests cover pure logic and need no database.

func TestAlias(t *testing.T) {
	if got := repository.Alias("5521040", model.ShapeDmaBoundary); got != "b5521040_dma_boundary" {
		t.Fatalf("Alias dma_boundary = %q", got)
	}
	if got := repository.Alias("5521040", model.ShapeFlowMeter); got != "b5521040_flow_meter" {
		t.Fatalf("Alias flow_meter = %q", got)
	}
	if got := repository.Alias("5521040", model.ShapeStepTest); got != "b5521040_step_test" {
		t.Fatalf("Alias step_test = %q", got)
	}
}

func TestShapeMetadata(t *testing.T) {
	want := map[string]string{
		model.ShapeDmaBoundary: "Polygon",
		model.ShapeFlowMeter:   "Point",
		model.ShapeStepTest:    "Polygon",
	}
	for shape, geom := range want {
		if got := model.GeometryTypeForShape[shape]; got != geom {
			t.Errorf("GeometryTypeForShape[%s] = %q, want %q", shape, got, geom)
		}
		if !model.AllowedShapes[shape] {
			t.Errorf("shape %s should be allowed", shape)
		}
	}
	if model.AllowedShapes["bogus"] {
		t.Error("bogus shape should not be allowed")
	}
}

func TestAsInt(t *testing.T) {
	cases := []struct {
		in   interface{}
		want int
	}{
		{int(9), 9},
		{int32(52), 52},
		{int64(7), 7},
		{float32(4), 4},
		{float64(3.9), 3},
		{nil, 0},
		{"x", 0},
	}
	for _, c := range cases {
		if got := model.AsInt(c.in); got != c.want {
			t.Errorf("AsInt(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}
