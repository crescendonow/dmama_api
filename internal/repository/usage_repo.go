package repository

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UsageRecord is one logged API request — a data-pull "credit" entry attributed to an API key.
type UsageRecord struct {
	APIKey     string
	Method     string
	Path       string
	Status     int
	SizeBytes  int64
	SizeValue  float64
	SizeUnit   string
	DurationMS int64
	StartedAt  time.Time
	EndedAt    time.Time
	RequestID  string
}

// UsageRecorder persists UsageRecords to auth_logs.dmama_use on PostgreSQL 16. Records are queued on
// a buffered channel and written by a single background worker so request latency is unaffected; a
// full queue drops the record (with a warning) rather than blocking.
type UsageRecorder struct {
	pool *pgxpool.Pool
	ch   chan UsageRecord
}

func NewUsageRecorder(pool *pgxpool.Pool, buffer int) *UsageRecorder {
	if buffer <= 0 {
		buffer = 1024
	}
	return &UsageRecorder{pool: pool, ch: make(chan UsageRecord, buffer)}
}

// EnsureSchema creates the auth_logs schema, the dmama_use table, and its indexes (idempotent).
func (r *UsageRecorder) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE SCHEMA IF NOT EXISTS auth_logs`,
		`CREATE TABLE IF NOT EXISTS auth_logs.dmama_use (
			id          bigserial PRIMARY KEY,
			api_key     text,
			method      text,
			path        text,
			status      integer,
			size_bytes  bigint,
			size_value  double precision,
			size_unit   text,
			duration_ms bigint,
			started_at  timestamptz,
			ended_at    timestamptz,
			request_id  text,
			created_at  timestamptz NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS dmama_use_api_key_idx ON auth_logs.dmama_use (api_key)`,
		`CREATE INDEX IF NOT EXISTS dmama_use_started_at_idx ON auth_logs.dmama_use (started_at)`,
	}
	for _, s := range stmts {
		if _, err := r.pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("ensure usage schema: %w", err)
		}
	}
	return nil
}

// Start launches the background writer; call once after EnsureSchema. It exits when ctx is done.
func (r *UsageRecorder) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case rec := <-r.ch:
				r.insert(ctx, rec)
			}
		}
	}()
}

// Record queues a usage record. Non-blocking: drops (with a warning) when the buffer is full.
func (r *UsageRecorder) Record(rec UsageRecord) {
	select {
	case r.ch <- rec:
	default:
		log.Printf("WARN: usage log buffer full; dropping record for %s %s", rec.Method, rec.Path)
	}
}

func (r *UsageRecorder) insert(ctx context.Context, rec UsageRecord) {
	const q = `
		INSERT INTO auth_logs.dmama_use
			(api_key, method, path, status, size_bytes, size_value, size_unit, duration_ms, started_at, ended_at, request_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`
	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := r.pool.Exec(writeCtx, q, rec.APIKey, rec.Method, rec.Path, rec.Status,
		rec.SizeBytes, rec.SizeValue, rec.SizeUnit, rec.DurationMS, rec.StartedAt, rec.EndedAt, rec.RequestID); err != nil {
		log.Printf("WARN: usage log insert failed: %v", err)
	}
}

// HumanizeBytes converts a byte count into a KB/MB/GB value + unit (1024 steps, 2 decimals).
func HumanizeBytes(n int64) (value float64, unit string) {
	const (
		kb = 1024.0
		mb = kb * 1024
		gb = mb * 1024
	)
	f := float64(n)
	switch {
	case f >= gb:
		return math.Round(f/gb*100) / 100, "GB"
	case f >= mb:
		return math.Round(f/mb*100) / 100, "MB"
	default:
		return math.Round(f/kb*100) / 100, "KB"
	}
}
