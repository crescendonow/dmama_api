package middleware

import (
	"bufio"
	"io"
	"net/http/httptest"
	"testing"

	"dmama_api/internal/config"
	"dmama_api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

func TestAPIKeyAuthRejectsMissingKeyBeforeDownstreamMiddleware(t *testing.T) {
	app := fiber.New()
	downstreamCalled := false

	app.Use(APIKeyAuth(&config.Config{DmamaKey: "secret"}))
	app.Use(func(c *fiber.Ctx) error {
		downstreamCalled = true
		return c.Next()
	})
	app.Get("/api/dma/stats", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	req := httptest.NewRequest("GET", "/api/dma/stats?pwa_code=5521027&dma_id=11&column=prswtusg", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected missing API key to return 401, got %d", resp.StatusCode)
	}
	if downstreamCalled {
		t.Fatal("expected missing API key to stop before downstream middleware")
	}
}

func TestAPIKeyAuthAllowsValidKeyThroughDownstreamMiddleware(t *testing.T) {
	app := fiber.New()
	downstreamCalled := false

	app.Use(APIKeyAuth(&config.Config{DmamaKey: "secret"}))
	app.Use(func(c *fiber.Ctx) error {
		downstreamCalled = true
		return c.Next()
	})
	app.Get("/api/dma/stats", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	req := httptest.NewRequest("GET", "/api/dma/stats?pwa_code=5521027&dma_id=11&column=prswtusg", nil)
	req.Header.Set("X-API-Key", "secret")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected valid API key to continue to route, got %d", resp.StatusCode)
	}
	if !downstreamCalled {
		t.Fatal("expected valid API key to continue through downstream middleware")
	}
}

// fakeUsageRecorder is an in-memory usageRecorder used to test UsageLogger without a database.
type fakeUsageRecorder struct {
	records []repository.UsageRecord
}

func (f *fakeUsageRecorder) Record(rec repository.UsageRecord) {
	f.records = append(f.records, rec)
}

// TestUsageLoggerDoesNotDrainStreamedResponseBody guards the fix for c.Response().Body() buffering
// a SetBodyStreamWriter response (e.g. /api/dma/customers-all, which can stream several GB) into
// memory just to measure its size. Streamed responses must record size 0 and the client must still
// receive the full body.
func TestUsageLoggerDoesNotDrainStreamedResponseBody(t *testing.T) {
	app := fiber.New()
	rec := &fakeUsageRecorder{}
	app.Use(UsageLogger(rec))
	app.Get("/api/dma/customers-all", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "application/json")
		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			w.WriteString(`{"success":true,"data":[1,2,3],"count":3}`)
			w.Flush()
		})
		return nil
	})

	req := httptest.NewRequest("GET", "/api/dma/customers-all", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}
	if string(body) != `{"success":true,"data":[1,2,3],"count":3}` {
		t.Fatalf("expected client to receive the full streamed body, got %q", body)
	}
	if len(rec.records) != 1 {
		t.Fatalf("expected exactly one usage record, got %d", len(rec.records))
	}
	if rec.records[0].SizeBytes != 0 {
		t.Fatalf("expected streamed response to record size 0 (not drained), got %d", rec.records[0].SizeBytes)
	}
}

// TestUsageLoggerRecordsSizeForNonStreamedResponses guards against a regression where the
// IsBodyStream check above accidentally also skips size recording for ordinary JSON responses.
func TestUsageLoggerRecordsSizeForNonStreamedResponses(t *testing.T) {
	app := fiber.New()
	rec := &fakeUsageRecorder{}
	app.Use(UsageLogger(rec))
	app.Get("/api/dma/stats-region", func(c *fiber.Ctx) error {
		return c.SendString(`{"success":true}`)
	})

	req := httptest.NewRequest("GET", "/api/dma/stats-region", nil)
	if _, err := app.Test(req); err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if len(rec.records) != 1 {
		t.Fatalf("expected exactly one usage record, got %d", len(rec.records))
	}
	if rec.records[0].SizeBytes != int64(len(`{"success":true}`)) {
		t.Fatalf("expected non-streamed response size to be recorded, got %d", rec.records[0].SizeBytes)
	}
}
