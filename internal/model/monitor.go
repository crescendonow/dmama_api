package model

import "time"

type MonitorUsageFilters struct {
	Preset string    `json:"preset"`
	From   time.Time `json:"from"`
	To     time.Time `json:"to"`
	APIKey string    `json:"api_key,omitempty"`
	Method string    `json:"method,omitempty"`
	Path   string    `json:"path,omitempty"`
	Status *int      `json:"status,omitempty"`
	Limit  int       `json:"limit"`
}

type MonitorUsageResponse struct {
	Latest    []MonitorLatestRequest   `json:"latest"`
	Endpoints []MonitorEndpointSummary `json:"endpoints"`
	Statuses  []MonitorStatusSummary   `json:"statuses"`
	APIKeys   []MonitorAPIKeySummary   `json:"api_keys"`
	Filters   MonitorUsageFilters      `json:"filters"`
}

type MonitorLatestRequest struct {
	StartedAt  time.Time `json:"started_at"`
	APIKey     string    `json:"api_key"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Status     int       `json:"status"`
	DurationMS int64     `json:"duration_ms"`
	SizeValue  float64   `json:"size_value"`
	SizeUnit   string    `json:"size_unit"`
	RequestID  string    `json:"request_id"`
}

type MonitorEndpointSummary struct {
	Path   string `json:"path"`
	Method string `json:"method"`
	Status int    `json:"status"`
	Calls  int64  `json:"calls"`
	AvgMS  int64  `json:"avg_ms"`
}

type MonitorStatusSummary struct {
	Status int   `json:"status"`
	Calls  int64 `json:"calls"`
}

type MonitorAPIKeySummary struct {
	APIKey     string `json:"api_key"`
	Calls      int64  `json:"calls"`
	TotalBytes int64  `json:"total_bytes"`
	AvgMS      int64  `json:"avg_ms"`
}
