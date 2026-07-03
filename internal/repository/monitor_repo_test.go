package repository

import (
	"strings"
	"testing"
	"time"

	"dmama_api/internal/model"
)

func TestMonitorUsageWhereUsesParameterizedFilters(t *testing.T) {
	status := 500
	filters := model.MonitorUsageFilters{
		From:   time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
		To:     time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC),
		APIKey: "secret",
		Method: "GET",
		Path:   "/api/dma",
		Status: &status,
	}

	where, args := monitorUsageWhere(filters)
	for _, want := range []string{
		"started_at >= $1",
		"started_at <= $2",
		"api_key = $3",
		"method = $4",
		"path ILIKE $5",
		"status = $6",
	} {
		if !strings.Contains(where, want) {
			t.Fatalf("expected where clause to contain %q, got %s", want, where)
		}
	}
	if len(args) != 6 {
		t.Fatalf("expected 6 args, got %d", len(args))
	}
	if args[4] != "%/api/dma%" {
		t.Fatalf("expected path wildcard arg, got %#v", args[4])
	}
}
