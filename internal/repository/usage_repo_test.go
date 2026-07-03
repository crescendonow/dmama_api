package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeUsageDB struct {
	rows    []fakeUsageRow
	execErr error
	queries []string
	execs   []string
	args    [][]any
}

func (f *fakeUsageDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	f.execs = append(f.execs, sql)
	f.args = append(f.args, arguments)
	if f.execErr != nil {
		return pgconn.CommandTag{}, f.execErr
	}
	return pgconn.CommandTag{}, nil
}

func (f *fakeUsageDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	f.queries = append(f.queries, sql)
	if len(f.rows) == 0 {
		return fakeUsageRow{err: errors.New("unexpected QueryRow")}
	}
	row := f.rows[0]
	f.rows = f.rows[1:]
	return row
}

type fakeUsageRow struct {
	exists bool
	ready  bool
	err    error
}

func (r fakeUsageRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != 2 {
		return fmt.Errorf("expected 2 scan destinations, got %d", len(dest))
	}
	exists, ok := dest[0].(*bool)
	if !ok {
		return fmt.Errorf("first scan destination is %T, want *bool", dest[0])
	}
	ready, ok := dest[1].(*bool)
	if !ok {
		return fmt.Errorf("second scan destination is %T, want *bool", dest[1])
	}
	*exists = r.exists
	*ready = r.ready
	return nil
}

func TestEnsureSchemaSkipsDDLWhenUsageTableReady(t *testing.T) {
	db := &fakeUsageDB{
		rows: []fakeUsageRow{{exists: true, ready: true}},
	}

	if err := newUsageRecorder(db, 1).EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema returned error: %v", err)
	}
	if len(db.execs) != 0 {
		t.Fatalf("expected no DDL when usage table is ready, got %d execs", len(db.execs))
	}
	if len(db.queries) != 1 || !strings.Contains(db.queries[0], "to_regclass('auth_logs.dmama_use')") {
		t.Fatalf("expected catalog readiness query, got %#v", db.queries)
	}
}

func TestEnsureSchemaCreatesUsageTableWhenMissing(t *testing.T) {
	db := &fakeUsageDB{
		rows: []fakeUsageRow{
			{exists: false, ready: false},
			{exists: true, ready: true},
		},
	}

	if err := newUsageRecorder(db, 1).EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema returned error: %v", err)
	}
	if len(db.execs) != len(usageSchemaDDL) {
		t.Fatalf("expected %d DDL execs, got %d", len(usageSchemaDDL), len(db.execs))
	}
	required := []string{
		"CREATE SCHEMA IF NOT EXISTS auth_logs",
		"CREATE TABLE IF NOT EXISTS auth_logs.dmama_use",
		"dmama_use_api_key_idx",
		"dmama_use_started_at_idx",
	}
	for _, want := range required {
		if !hasExecContaining(db.execs, want) {
			t.Fatalf("expected DDL containing %q, got %#v", want, db.execs)
		}
	}
}

func TestEnsureSchemaRejectsExistingUsageTableWithWrongSchema(t *testing.T) {
	db := &fakeUsageDB{
		rows: []fakeUsageRow{{exists: true, ready: false}},
	}

	err := newUsageRecorder(db, 1).EnsureSchema(context.Background())
	if err == nil {
		t.Fatal("expected EnsureSchema to fail")
	}
	if len(db.execs) != 0 {
		t.Fatalf("expected no DDL against existing table with wrong schema, got %d execs", len(db.execs))
	}
	for _, want := range []string{"does not match", "migrations/0002_auth_logs_dmama_use.sql", "GISDATA_URL"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %q", want, err.Error())
		}
	}
}
func TestEnsureSchemaDDLFailureMentionsMigrationAndGISDataURL(t *testing.T) {
	db := &fakeUsageDB{
		rows:    []fakeUsageRow{{exists: false, ready: false}},
		execErr: errors.New("permission denied"),
	}

	err := newUsageRecorder(db, 1).EnsureSchema(context.Background())
	if err == nil {
		t.Fatal("expected EnsureSchema to fail")
	}
	for _, want := range []string{"permission denied", "migrations/0002_auth_logs_dmama_use.sql", "GISDATA_URL"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %q", want, err.Error())
		}
	}
}

func TestUsageInsertWritesToDMAMAUse(t *testing.T) {
	db := &fakeUsageDB{}
	rec := newUsageRecorder(db, 1)

	rec.insert(context.Background(), UsageRecord{Method: "GET", Path: "/api/dma/stats"})

	if len(db.execs) != 1 {
		t.Fatalf("expected one insert exec, got %d", len(db.execs))
	}
	if !strings.Contains(db.execs[0], "INSERT INTO auth_logs.dmama_use") {
		t.Fatalf("expected insert into auth_logs.dmama_use, got %s", db.execs[0])
	}
	if len(db.args[0]) != 11 {
		t.Fatalf("expected 11 insert args, got %d", len(db.args[0]))
	}
}

func hasExecContaining(execs []string, want string) bool {
	for _, exec := range execs {
		if strings.Contains(exec, want) {
			return true
		}
	}
	return false
}
