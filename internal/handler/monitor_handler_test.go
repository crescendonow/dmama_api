package handler

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestParseMonitorUsageFiltersDefaultsAndNormalizes(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)

	filters, err := parseMonitorUsageFiltersForTarget(t, "/monitor?method=get&path=/api/dma&status=500&limit=25", now)
	if err != nil {
		t.Fatalf("parseMonitorUsageFilters returned error: %v", err)
	}
	if filters.Preset != "24h" {
		t.Fatalf("expected default preset 24h, got %q", filters.Preset)
	}
	if !filters.From.Equal(now.Add(-24*time.Hour)) || !filters.To.Equal(now) {
		t.Fatalf("expected 24h range ending at now, got from=%s to=%s", filters.From, filters.To)
	}
	if filters.Method != "GET" {
		t.Fatalf("expected method GET, got %q", filters.Method)
	}
	if filters.Path != "/api/dma" {
		t.Fatalf("expected path filter, got %q", filters.Path)
	}
	if filters.Status == nil || *filters.Status != 500 {
		t.Fatalf("expected status 500, got %#v", filters.Status)
	}
	if filters.Limit != 25 {
		t.Fatalf("expected limit 25, got %d", filters.Limit)
	}
}

func TestParseMonitorUsageFiltersRejectsInvalidStatus(t *testing.T) {
	_, err := parseMonitorUsageFiltersForTarget(t, "/monitor?status=abc", time.Now())
	if err == nil || !strings.Contains(err.Error(), "status must be 100-599") {
		t.Fatalf("expected status validation error, got %v", err)
	}
}

func TestParseMonitorUsageFiltersRejectsLimitOverMax(t *testing.T) {
	_, err := parseMonitorUsageFiltersForTarget(t, "/monitor?limit=501", time.Now())
	if err == nil || !strings.Contains(err.Error(), "limit must be 1-500") {
		t.Fatalf("expected limit validation error, got %v", err)
	}
}

func TestParseMonitorUsageFiltersRejectsInvalidCustomRange(t *testing.T) {
	_, err := parseMonitorUsageFiltersForTarget(t, "/monitor?preset=custom&from=2026-07-03T10:00:00Z&to=2026-07-03T09:00:00Z", time.Now())
	if err == nil || !strings.Contains(err.Error(), "to must be after from") {
		t.Fatalf("expected custom range validation error, got %v", err)
	}
}

func parseMonitorUsageFiltersForTarget(t *testing.T, target string, now time.Time) (filters monitorUsageFilters, err error) {
	t.Helper()

	app := fiber.New()
	app.Get("/monitor", func(c *fiber.Ctx) error {
		filters, err = parseMonitorUsageFilters(c, now)
		return nil
	})

	req := httptest.NewRequest("GET", target, nil)
	resp, testErr := app.Test(req)
	if testErr != nil {
		t.Fatalf("app.Test returned error: %v", testErr)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected test route status 200, got %d", resp.StatusCode)
	}
	return filters, err
}
