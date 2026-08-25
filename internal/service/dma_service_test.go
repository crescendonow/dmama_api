package service

import (
	"testing"
	"time"

	"dmama_api/internal/model"
)

func TestResolveStatsColumnFromYearMonthBeforeBillingDay(t *testing.T) {
	now := time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC)

	column, err := ResolveStatsColumn("2025", "6", "prswtusg", now)
	if err != nil {
		t.Fatalf("ResolveStatsColumn returned error: %v", err)
	}
	if column != "lstwtusg11" {
		t.Fatalf("expected lstwtusg11, got %s", column)
	}
}

func TestResolveStatsColumnFromYearMonthOnOrAfterBillingDay(t *testing.T) {
	now := time.Date(2026, time.June, 16, 0, 0, 0, 0, time.UTC)

	column, err := ResolveStatsColumn("2026", "6", "", now)
	if err != nil {
		t.Fatalf("ResolveStatsColumn returned error: %v", err)
	}
	if column != "prswtusg" {
		t.Fatalf("expected prswtusg, got %s", column)
	}
}

func TestStatsYearMonthPresentUsageAtCutoff(t *testing.T) {
	beforeCutoff := time.Date(2026, time.August, 15, 0, 0, 0, 0, time.UTC)
	afterCutoff := time.Date(2026, time.August, 16, 0, 0, 0, 0, time.UTC)

	if got, err := ResolveStatsYearMonth("prswtusg", beforeCutoff); err != nil || got != "256907" {
		t.Fatalf("day 15 year_month = %q, %v; want 256907, nil", got, err)
	}
	if got, err := ResolveStatsYearMonth("prswtusg", afterCutoff); err != nil || got != "256908" {
		t.Fatalf("day 16 year_month = %q, %v; want 256908, nil", got, err)
	}
}

func TestStatsYearMonthAppliesHistoryColumnOffset(t *testing.T) {
	now := time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		column string
		want   string
	}{
		{column: "lstwtusg1", want: "256907"},
		{column: "lstwtusg2", want: "256906"},
		{column: "lstwtusg12", want: "256808"},
	}

	for _, tt := range tests {
		t.Run(tt.column, func(t *testing.T) {
			if got, err := ResolveStatsYearMonth(tt.column, now); err != nil || got != tt.want {
				t.Fatalf("ResolveStatsYearMonth(%q) = %q, %v; want %q, nil", tt.column, got, err, tt.want)
			}
		})
	}
}

func TestStatsYearMonthHandlesShorterPreviousMonth(t *testing.T) {
	now := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)

	if got, err := ResolveStatsYearMonth("lstwtusg1", now); err != nil || got != "256902" {
		t.Fatalf("ResolveStatsYearMonth(lstwtusg1) = %q, %v; want 256902, nil", got, err)
	}
}

func TestResolveStatsYearMonthDistinguishesLegacyFromInvalidColumns(t *testing.T) {
	now := time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC)
	for _, column := range []string{"use_water", "use_jan", "use_feb", "use_mar", "use_apr", "use_may", "use_jun", "use_jul", "use_aug", "use_sep", "use_oct", "use_nov", "use_dec"} {
		t.Run("legacy_"+column, func(t *testing.T) {
			got, err := ResolveStatsYearMonth(column, now)
			if err != nil || got != "" {
				t.Fatalf("ResolveStatsYearMonth(%q) = %q, %v; want empty, nil", column, got, err)
			}
		})
	}
	for _, column := range []string{"lstwtusg", "lstwtusg0", "lstwtusg13", "lstwtusg01", "prswtusg;drop table"} {
		t.Run("invalid_"+column, func(t *testing.T) {
			if _, err := ResolveStatsYearMonth(column, now); err == nil {
				t.Fatalf("expected invalid column %q to return an error", column)
			}
		})
	}
}

func TestStatsCacheExpiresAtBillingBoundary(t *testing.T) {
	beforeBoundary := time.Date(2026, time.August, 15, 23, 59, 0, 0, time.UTC)
	if got, want := statsCacheExpiresAt(beforeBoundary), time.Date(2026, time.August, 16, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("pre-boundary expiry = %s, want %s", got, want)
	}

	afterBoundary := time.Date(2026, time.August, 16, 0, 0, 0, 0, time.UTC)
	if got, want := statsCacheExpiresAt(afterBoundary), afterBoundary.Add(statsCacheTTL); !got.Equal(want) {
		t.Fatalf("post-boundary expiry = %s, want %s", got, want)
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
	yearMonth := "256908"
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

	result := prepareDMAStatsResponse(input, "5532013", "6", "prswtusg", yearMonth)

	if result.PwaCode != "5532013" {
		t.Fatalf("expected pwa_code from request, got %s", result.PwaCode)
	}
	if result.DmaID != "6" {
		t.Fatalf("expected dma_id from request, got %s", result.DmaID)
	}
	if result.Column != "prswtusg" {
		t.Fatalf("expected column from request, got %s", result.Column)
	}
	if result.YearMonth != "256908" {
		t.Fatalf("expected year_month 256908, got %s", result.YearMonth)
	}
	if input.PwaCode != "5532011" || input.DmaID != "2" || input.Column != "lstwtusg1" {
		t.Fatal("prepareDMAStatsResponse mutated input stats")
	}
	if result.Usage.Total != 46493 || result.Population.Total != 1986 {
		t.Fatal("prepareDMAStatsResponse changed numeric stats")
	}
}

func TestPrepareDMAStatsRegionResponseAddsYearMonthToEveryItem(t *testing.T) {
	yearMonth := "256906"
	input := []model.DMAStats{
		{PwaCode: "5541011", DmaID: "1", Column: "prswtusg", Usage: model.DMAUsage{Total: 10}, Population: model.DMAPopulationStats{Total: 2}},
		{PwaCode: "5541011", DmaID: "2", Column: "prswtusg", Usage: model.DMAUsage{Total: 20}, Population: model.DMAPopulationStats{Total: 3}},
	}

	result := prepareDMAStatsRegionResponse(input, "lstwtusg2", yearMonth)
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	for i := range result {
		if result[i].Column != "lstwtusg2" {
			t.Fatalf("item %d column = %q, want lstwtusg2", i, result[i].Column)
		}
		if result[i].YearMonth != "256906" {
			t.Fatalf("item %d year_month = %q, want 256906", i, result[i].YearMonth)
		}
	}
	if result[0].Usage.Total != 10 || result[0].Population.Total != 2 ||
		result[1].Usage.Total != 20 || result[1].Population.Total != 3 {
		t.Fatal("response preparation changed numeric statistics")
	}
	if input[0].Column != "prswtusg" || input[0].YearMonth != "" {
		t.Fatal("response preparation mutated input items")
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
	allowed := []string{"prswtusg", "lstwtusg1", "lstwtusg2", "lstwtusg12"}
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
	rejected := []string{"ltwtusg1", "lstwtusg13", "use_jan", "prswtusg;drop table"}
	for _, input := range rejected {
		t.Run(input, func(t *testing.T) {
			if _, err := ResolveStatsRegionColumn(input); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
