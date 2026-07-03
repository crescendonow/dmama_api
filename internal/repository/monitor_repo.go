package repository

import (
	"context"
	"fmt"
	"strings"

	"dmama_api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type usageMonitorDB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type UsageMonitorRepo struct {
	db usageMonitorDB
}

func NewUsageMonitorRepo(pool *pgxpool.Pool) *UsageMonitorRepo {
	return &UsageMonitorRepo{db: pool}
}

func (r *UsageMonitorRepo) GetUsage(ctx context.Context, filters model.MonitorUsageFilters) (*model.MonitorUsageResponse, error) {
	latest, err := r.LatestRequests(ctx, filters)
	if err != nil {
		return nil, err
	}
	endpoints, err := r.EndpointSummary(ctx, filters)
	if err != nil {
		return nil, err
	}
	statuses, err := r.StatusSummary(ctx, filters)
	if err != nil {
		return nil, err
	}
	apiKeys, err := r.APIKeySummary(ctx, filters)
	if err != nil {
		return nil, err
	}

	return &model.MonitorUsageResponse{
		Latest:    latest,
		Endpoints: endpoints,
		Statuses:  statuses,
		APIKeys:   apiKeys,
		Filters:   filters,
	}, nil
}

func (r *UsageMonitorRepo) LatestRequests(ctx context.Context, filters model.MonitorUsageFilters) ([]model.MonitorLatestRequest, error) {
	where, args := monitorUsageWhere(filters)
	args, limitRef := appendMonitorLimit(args, filters.Limit)
	query := fmt.Sprintf(`
		SELECT started_at, COALESCE(api_key, ''), COALESCE(method, ''), COALESCE(path, ''),
			COALESCE(status, 0), COALESCE(duration_ms, 0), COALESCE(size_value, 0),
			COALESCE(size_unit, ''), COALESCE(request_id, '')
		FROM auth_logs.dmama_use
		WHERE %s
		ORDER BY started_at DESC
		LIMIT %s`, where, limitRef)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.MonitorLatestRequest
	for rows.Next() {
		var item model.MonitorLatestRequest
		if err := rows.Scan(&item.StartedAt, &item.APIKey, &item.Method, &item.Path, &item.Status,
			&item.DurationMS, &item.SizeValue, &item.SizeUnit, &item.RequestID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *UsageMonitorRepo) EndpointSummary(ctx context.Context, filters model.MonitorUsageFilters) ([]model.MonitorEndpointSummary, error) {
	where, args := monitorUsageWhere(filters)
	args, limitRef := appendMonitorLimit(args, filters.Limit)
	query := fmt.Sprintf(`
		SELECT COALESCE(path, ''), COALESCE(method, ''), COALESCE(status, 0),
			count(*) AS calls, COALESCE(avg(duration_ms)::int, 0) AS avg_ms
		FROM auth_logs.dmama_use
		WHERE %s
		GROUP BY path, method, status
		ORDER BY calls DESC
		LIMIT %s`, where, limitRef)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.MonitorEndpointSummary
	for rows.Next() {
		var item model.MonitorEndpointSummary
		if err := rows.Scan(&item.Path, &item.Method, &item.Status, &item.Calls, &item.AvgMS); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *UsageMonitorRepo) StatusSummary(ctx context.Context, filters model.MonitorUsageFilters) ([]model.MonitorStatusSummary, error) {
	where, args := monitorUsageWhere(filters)
	query := fmt.Sprintf(`
		SELECT COALESCE(status, 0), count(*) AS calls
		FROM auth_logs.dmama_use
		WHERE %s
		GROUP BY status
		ORDER BY status`, where)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.MonitorStatusSummary
	for rows.Next() {
		var item model.MonitorStatusSummary
		if err := rows.Scan(&item.Status, &item.Calls); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *UsageMonitorRepo) APIKeySummary(ctx context.Context, filters model.MonitorUsageFilters) ([]model.MonitorAPIKeySummary, error) {
	where, args := monitorUsageWhere(filters)
	args, limitRef := appendMonitorLimit(args, filters.Limit)
	query := fmt.Sprintf(`
		SELECT COALESCE(api_key, ''), count(*) AS calls, COALESCE(sum(size_bytes), 0)::bigint AS total_bytes,
			COALESCE(avg(duration_ms)::int, 0) AS avg_ms
		FROM auth_logs.dmama_use
		WHERE %s
		GROUP BY api_key
		ORDER BY calls DESC
		LIMIT %s`, where, limitRef)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.MonitorAPIKeySummary
	for rows.Next() {
		var item model.MonitorAPIKeySummary
		if err := rows.Scan(&item.APIKey, &item.Calls, &item.TotalBytes, &item.AvgMS); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func monitorUsageWhere(filters model.MonitorUsageFilters) (string, []any) {
	conditions := []string{"started_at >= $1", "started_at <= $2"}
	args := []any{filters.From, filters.To}

	add := func(condition string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(condition, len(args)))
	}
	if filters.APIKey != "" {
		add("api_key = $%d", filters.APIKey)
	}
	if filters.Method != "" {
		add("method = $%d", filters.Method)
	}
	if filters.Path != "" {
		add("path ILIKE $%d", "%"+filters.Path+"%")
	}
	if filters.Status != nil {
		add("status = $%d", *filters.Status)
	}

	return strings.Join(conditions, " AND "), args
}

func appendMonitorLimit(args []any, limit int) ([]any, string) {
	next := append([]any{}, args...)
	next = append(next, limit)
	return next, fmt.Sprintf("$%d", len(next))
}
