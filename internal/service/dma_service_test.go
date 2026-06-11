package service

import (
	"testing"
	"time"
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
