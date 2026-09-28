package middleware

import (
	"time"

	"dmama_api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// usageRecorder is the subset of *repository.UsageRecorder that UsageLogger needs; declaring it
// as an interface (rather than depending on the concrete type) lets tests use a fake recorder.
type usageRecorder interface {
	Record(repository.UsageRecord)
}

// UsageLogger records each request's response size, duration, and timestamps to auth_logs.dmama_use.
// Register on the authenticated /api group (after APIKeyAuth) so health checks and rejected (401)
// requests are excluded. Recording is async and never blocks or fails the request.
func UsageLogger(rec usageRecorder) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		end := time.Now()

		// Streamed responses (SetBodyStreamWriter, e.g. /api/dma/customers-all) are written
		// directly to the socket after this middleware returns; c.Response().Body() would drain
		// that stream into memory first (multi-GB for the largest requests) just to measure it.
		// Size is recorded as 0 for those instead of buffering the whole response.
		var sizeBytes int64
		if !c.Response().IsBodyStream() {
			sizeBytes = int64(len(c.Response().Body()))
		}
		value, unit := repository.HumanizeBytes(sizeBytes)

		rec.Record(repository.UsageRecord{
			APIKey:     c.Get("X-API-Key"),
			Method:     c.Method(),
			Path:       c.Path(),
			Status:     c.Response().StatusCode(),
			SizeBytes:  sizeBytes,
			SizeValue:  value,
			SizeUnit:   unit,
			DurationMS: end.Sub(start).Milliseconds(),
			StartedAt:  start.UTC(),
			EndedAt:    end.UTC(),
			RequestID:  requestIDOf(c),
		})
		return err
	}
}

func requestIDOf(c *fiber.Ctx) string {
	if v, ok := c.Locals(requestid.ConfigDefault.ContextKey).(string); ok {
		return v
	}
	return ""
}
