package service

import (
	"testing"
	"time"

	"dmama_api/internal/model"
)

func TestResolveStatsColumnFromYearMonthBeforeBillingDay(t *testing.T) {
	now := time.Date(2026, time.June, 11, 0, 0, 0, 0, time.UTC)

	column, err := ResolveStatsColumn("2025", "6", "prswtusg", now)
	if err != nil {
		t.Fatalf("ResolveStatsColumn returned error: %v", err)
	}
	if column != "lstwtusg11" {
		t.Fatalf("expected lstwtusg11, got %s", column)
	}
}

func TestResolveStatsColumnClampsOlderThanTwelveMonths(t *testing.T) {
	now := time.Date(2026, time.June, 21, 0, 0, 0, 0, time.UTC)

	column, err := ResolveStatsColumn("2024", "1", "", now)
	if err != nil {
		t.Fatalf("ResolveStatsColumn returned error: %v", err)
	}
	if column != "lstwtusg12" {
		t.Fatalf("expected lstwtusg12, got %s", column)
	}
}

func TestResolveStatsColumnUsesYearMonthBeforeColumn(t *testing.T) {
	now := time.Date(2026, time.June, 21, 0, 0, 0, 0, time.UTC)

	column, err := ResolveStatsColumn("2026", "6", "lstwtusg12", now)
	if err != nil {
		t.Fatalf("ResolveStatsColumn returned error: %v", err)
	}
	if column != "prswtusg" {
		t.Fatalf("expected prswtusg, got %s", column)
	}
}

func TestResolveStatsColumnRejectsInvalidInput(t *testing.T) {
	now := time.Date(2026, time.June, 21, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		year   string
		month  string
		column string
	}{
		{name: "missing month", year: "2026"},
		{name: "bad month", year: "2026", month: "13"},
		{name: "future billing cycle", year: "2026", month: "7"},
		{name: "invalid column", column: "prswtusg;drop table"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ResolveStatsColumn(tt.year, tt.month, tt.column, now); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestPrepareDMAStatsResponseUsesRequestMetadata(t *testing.T) {
	input := &model.DMAStats{
		PwaCode: "5532011",
		DmaID:   "2",
		Column:  "lstwtusg1",
		Usage: model.DMAUsage{
			Total: 46493,
		},
		Population: model.DMAPopulationStats{
			Total: 1986,
		},
	}

	result := prepareDMAStatsResponse(input, "5532013", "6", "prswtusg")

	if result.PwaCode != "5532013" {
		t.Fatalf("expected pwa_code from request, got %s", result.PwaCode)
	}
	if result.DmaID != "6" {
		t.Fatalf("expected dma_id from request, got %s", result.DmaID)
	}
	if result.Column != "prswtusg" {
		t.Fatalf("expected column from request, got %s", result.Column)
	}
	if input.PwaCode != "5532011" || input.DmaID != "2" || input.Column != "lstwtusg1" {
		t.Fatal("prepareDMAStatsResponse mutated input stats")
	}
	if result.Usage.Total != 46493 || result.Population.Total != 1986 {
		t.Fatal("prepareDMAStatsResponse changed numeric stats")
	}
}

func TestResolveStatsRegionColumnDefaultsToPresentUsage(t *testing.T) {
	column, err := ResolveStatsRegionColumn("")
	if err != nil {
		t.Fatalf("ResolveStatsRegionColumn returned error: %v", err)
	}
	if column != "prswtusg" {
		t.Fatalf("expected prswtusg, got %s", column)
	}
}

func TestResolveStatsRegionColumnAllowsOnlyContractColumns(t *testing.T) {
	allowed := []string{"prswtusg", "lstwtusg1"}
	for _, input := range allowed {
		t.Run(input, func(t *testing.T) {
			column, err := ResolveStatsRegionColumn(input)
			if err != nil {
				t.Fatalf("expected %s to be allowed: %v", input, err)
			}
			if column != input {
				t.Fatalf("expected %s, got %s", input, column)
			}
		})
	}
}

func TestResolveStatsRegionColumnRejectsUnsupportedColumns(t *testing.T) {
	rejected := []string{"ltwtusg1", "lstwtusg2", "prswtusg;drop table"}
	for _, input := range rejected {
		t.Run(input, func(t *testing.T) {
			if _, err := ResolveStatsRegionColumn(input); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
