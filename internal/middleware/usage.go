package middleware

import (
	"time"

	"dmama_api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// UsageLogger records each request's response size, duration, and timestamps to auth_logs.dmama_use.
// Register on the authenticated /api group (after APIKeyAuth) so health checks and rejected (401)
// requests are excluded. Recording is async and never blocks or fails the request.
func UsageLogger(rec *repository.UsageRecorder) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		end := time.Now()

		sizeBytes := int64(len(c.Response().Body()))
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
