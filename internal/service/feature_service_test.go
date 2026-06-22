package service

import (
	"testing"

	"dmama_api/internal/model"
)

func TestStepTestUsesDmaCoverageRule(t *testing.T) {
	if !usesDmaCoverageRule(model.ShapeStepTest) {
		t.Fatal("step_test should use dma coverage validation")
	}
	if got := outsideDmaCoverageMessage(model.ShapeStepTest); got != "polygon is outside the branch dma_boundary coverage" {
		t.Fatalf("step_test coverage message = %q", got)
	}
}

func TestFlowMeterStillUsesPointCoverageMessage(t *testing.T) {
	if !usesDmaCoverageRule(model.ShapeFlowMeter) {
		t.Fatal("flow_meter should use dma coverage validation")
	}
	if got := outsideDmaCoverageMessage(model.ShapeFlowMeter); got != "point is outside the branch dma_boundary coverage" {
		t.Fatalf("flow_meter coverage message = %q", got)
	}
}
