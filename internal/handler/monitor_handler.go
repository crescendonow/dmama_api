package handler

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"dmama_api/internal/model"
	"dmama_api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MonitorHandler struct {
	repo *repository.UsageMonitorRepo
}

func NewMonitorHandler(pool *pgxpool.Pool) *MonitorHandler {
	return &MonitorHandler{repo: repository.NewUsageMonitorRepo(pool)}
}

func (h *MonitorHandler) Usage(c *fiber.Ctx) error {
	filters, err := parseMonitorUsageFilters(c, time.Now())
	if err != nil {
		return c.Status(400).JSON(model.ErrorResponse(err.Error()))
	}

	result, err := h.repo.GetUsage(c.Context(), filters)
	if err != nil {
		return c.Status(500).JSON(model.ErrorResponse(err.Error()))
	}
	return c.JSON(model.SuccessResponse(result))
}

func MonitorUnavailable(c *fiber.Ctx) error {
	return c.Status(503).JSON(model.ErrorResponse("usage logging unavailable: GISDATA_URL is not configured or auth_logs.dmama_use is not ready"))
}

func parseMonitorUsageFilters(c *fiber.Ctx, now time.Time) (model.MonitorUsageFilters, error) {
	preset := strings.TrimSpace(c.Query("preset", "24h"))
	if preset == "" {
		preset = "24h"
	}

	now = now.UTC()
	from, to, err := monitorTimeRange(preset, c, now)
	if err != nil {
		return model.MonitorUsageFilters{}, err
	}

	limit, err := monitorLimit(c.Query("limit"))
	if err != nil {
		return model.MonitorUsageFilters{}, err
	}

	status, err := monitorStatus(c.Query("status"))
	if err != nil {
		return model.MonitorUsageFilters{}, err
	}

	return model.MonitorUsageFilters{
		Preset: preset,
		From:   from,
		To:     to,
		APIKey: strings.TrimSpace(c.Query("api_key")),
		Method: strings.ToUpper(strings.TrimSpace(c.Query("method"))),
		Path:   strings.TrimSpace(c.Query("path")),
		Status: status,
		Limit:  limit,
	}, nil
}

func monitorTimeRange(preset string, c *fiber.Ctx, now time.Time) (time.Time, time.Time, error) {
	switch preset {
	case "1h":
		return now.Add(-1 * time.Hour), now, nil
	case "6h":
		return now.Add(-6 * time.Hour), now, nil
	case "24h":
		return now.Add(-24 * time.Hour), now, nil
	case "7d":
		return now.AddDate(0, 0, -7), now, nil
	case "custom":
		fromRaw := strings.TrimSpace(c.Query("from"))
		toRaw := strings.TrimSpace(c.Query("to"))
		if fromRaw == "" || toRaw == "" {
			return time.Time{}, time.Time{}, errors.New("from and to are required for custom preset")
		}
		from, err := time.Parse(time.RFC3339, fromRaw)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("from must be RFC3339")
		}
		to, err := time.Parse(time.RFC3339, toRaw)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("to must be RFC3339")
		}
		from = from.UTC()
		to = to.UTC()
		if !to.After(from) {
			return time.Time{}, time.Time{}, errors.New("to must be after from")
		}
		return from, to, nil
	default:
		return time.Time{}, time.Time{}, errors.New("preset must be one of 1h, 6h, 24h, 7d, or custom")
	}
}

func monitorLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 100, nil
	}
	limit, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || limit < 1 || limit > 500 {
		return 0, errors.New("limit must be 1-500")
	}
	return limit, nil
}

func monitorStatus(raw string) (*int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	status, err := strconv.Atoi(raw)
	if err != nil || status < 100 || status > 599 {
		return nil, errors.New("status must be 100-599")
	}
	return &status, nil
}
