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

type fakeTopoDB struct {
	rows      []fakeTopoRow
	execErr   error
	queries   []string
	queryArgs [][]any
	execs     []string
	execArgs  [][]any
}

func (f *fakeTopoDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	f.execs = append(f.execs, sql)
	f.execArgs = append(f.execArgs, arguments)
	if f.execErr != nil {
		return pgconn.CommandTag{}, f.execErr
	}
	return pgconn.CommandTag{}, nil
}

func (f *fakeTopoDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	f.queries = append(f.queries, sql)
	f.queryArgs = append(f.queryArgs, args)
	if len(f.rows) == 0 {
		return fakeTopoRow{err: errors.New("unexpected QueryRow")}
	}
	row := f.rows[0]
	f.rows = f.rows[1:]
	return row
}

// fakeTopoRow scans a fixed list of bool values, in order. It serves both the 1-column
// to_regnamespace probe and the 2-column (table_exists, schema_ready) probe.
type fakeTopoRow struct {
	values []bool
	err    error
}

func (r fakeTopoRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return fmt.Errorf("expected %d scan destinations, got %d", len(r.values), len(dest))
	}
	for i, d := range dest {
		ptr, ok := d.(*bool)
		if !ok {
			return fmt.Errorf("scan destination %d is %T, want *bool", i, d)
		}
		*ptr = r.values[i]
	}
	return nil
}

func hasQueryContaining(queries []string, want string) bool {
	for _, q := range queries {
		if strings.Contains(q, want) {
			return true
		}
	}
	return false
}

// TestEnsureSchemaSkipsDDLWhenTopologyTablesReady is the regression test for this outage: a
// least-privilege dmama with a fully-provisioned dmama_layer must not attempt any DDL (today's
// EnsureSchema fires 4 statements per table regardless, and 42501s before the IF NOT EXISTS
// short-circuit even runs).
func TestEnsureSchemaSkipsDDLWhenTopologyTablesReady(t *testing.T) {
	db := &fakeTopoDB{
		rows: []fakeTopoRow{
			{values: []bool{true}},       // schema exists
			{values: []bool{true, true}}, // dma_boundary
			{values: []bool{true, true}}, // flow_meter
			{values: []bool{true, true}}, // step_test
		},
	}

	if err := newTopologyRepo(db).EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema returned error: %v", err)
	}
	if len(db.execs) != 0 {
		t.Fatalf("expected no DDL when dmama_layer is ready, got %d execs: %#v", len(db.execs), db.execs)
	}
}

// TestEnsureSchemaProbesWithoutDDLPrivileges proves a least-privilege dmama (no DDL rights at
// all) works when the schema is already ready: every Exec would fail with 42501, but Exec must
// never be called.
func TestEnsureSchemaProbesWithoutDDLPrivileges(t *testing.T) {
	db := &fakeTopoDB{
		rows: []fakeTopoRow{
			{values: []bool{true}},
			{values: []bool{true, true}},
			{values: []bool{true, true}},
			{values: []bool{true, true}},
		},
		execErr: &pgconn.PgError{Code: "42501"},
	}

	if err := newTopologyRepo(db).EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema returned error: %v", err)
	}
	if len(db.execs) != 0 {
		t.Fatalf("expected zero DDL execs (least-privilege dmama must never need CREATE), got %d", len(db.execs))
	}
}

func TestEnsureSchemaCreatesOnlyMissingTables(t *testing.T) {
	db := &fakeTopoDB{
		rows: []fakeTopoRow{
			{values: []bool{true}},         // schema exists
			{values: []bool{true, true}},   // dma_boundary ready
			{values: []bool{false, false}}, // flow_meter missing
			{values: []bool{true, true}},   // step_test ready
			// re-probe after creating flow_meter
			{values: []bool{true}},
			{values: []bool{true, true}},
			{values: []bool{true, true}},
			{values: []bool{true, true}},
		},
	}

	if err := newTopologyRepo(db).EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema returned error: %v", err)
	}
	if hasExecContaining(db.execs, "CREATE SCHEMA") {
		t.Fatalf("expected no CREATE SCHEMA exec when the schema already exists, got %#v", db.execs)
	}
	for _, want := range []string{
		"CREATE TABLE IF NOT EXISTS dmama_layer.flow_meter",
		"flow_meter_geom_gix",
		"flow_meter_pwa_idx",
		"flow_meter_pwa_dma_idx",
	} {
		if !hasExecContaining(db.execs, want) {
			t.Fatalf("expected DDL containing %q, got %#v", want, db.execs)
		}
	}
	if hasExecContaining(db.execs, "dma_boundary") || hasExecContaining(db.execs, "step_test") {
		t.Fatalf("expected DDL only for the missing table, got %#v", db.execs)
	}
}

func TestEnsureSchemaCreatesSchemaWhenNamespaceMissing(t *testing.T) {
	db := &fakeTopoDB{
		rows: []fakeTopoRow{
			{values: []bool{false}},        // schema missing
			{values: []bool{false, false}}, // dma_boundary missing
			{values: []bool{false, false}}, // flow_meter missing
			{values: []bool{false, false}}, // step_test missing
			// re-probe after create
			{values: []bool{true}},
			{values: []bool{true, true}},
			{values: []bool{true, true}},
			{values: []bool{true, true}},
		},
	}

	if err := newTopologyRepo(db).EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema returned error: %v", err)
	}
	if !hasExecContaining(db.execs, "CREATE SCHEMA IF NOT EXISTS dmama_layer") {
		t.Fatalf("expected a CREATE SCHEMA exec, got %#v", db.execs)
	}
	for _, name := range []string{"dma_boundary", "flow_meter", "step_test"} {
		if !hasExecContaining(db.execs, "CREATE TABLE IF NOT EXISTS dmama_layer."+name) {
			t.Fatalf("expected CREATE TABLE for %s, got %#v", name, db.execs)
		}
	}
}

func TestEnsureSchemaRejectsMismatchedTable(t *testing.T) {
	db := &fakeTopoDB{
		rows: []fakeTopoRow{
			{values: []bool{true}},
			{values: []bool{true, true}},  // dma_boundary ready
			{values: []bool{true, false}}, // flow_meter exists but wrong shape
			{values: []bool{true, true}},  // step_test ready
		},
	}

	err := newTopologyRepo(db).EnsureSchema(context.Background())
	if err == nil {
		t.Fatal("expected EnsureSchema to fail")
	}
	if len(db.execs) != 0 {
		t.Fatalf("expected no DDL against a mismatched table, got %d execs: %#v", len(db.execs), db.execs)
	}
	for _, want := range []string{"flow_meter", "does not match", "migrations/0001_dmama_layer.sql", "GISDATA_URL"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %q", want, err.Error())
		}
	}
}

func TestSchemaStatusQueriesEachMirrorTable(t *testing.T) {
	db := &fakeTopoDB{
		rows: []fakeTopoRow{
			{values: []bool{true}},
			{values: []bool{true, true}},   // dma_boundary ready
			{values: []bool{false, false}}, // flow_meter missing
			{values: []bool{true, false}},  // step_test mismatched
		},
	}

	status, err := newTopologyRepo(db).SchemaStatus(context.Background())
	if err != nil {
		t.Fatalf("SchemaStatus returned error: %v", err)
	}
	if !status.SchemaExists {
		t.Fatal("expected SchemaExists=true")
	}
	if status.Ready {
		t.Fatal("expected Ready=false")
	}
	if len(status.MissingTables) != 1 || status.MissingTables[0] != "flow_meter" {
		t.Fatalf("expected MissingTables=[flow_meter], got %#v", status.MissingTables)
	}
	if len(status.Mismatched) != 1 || status.Mismatched[0] != "step_test" {
		t.Fatalf("expected Mismatched=[step_test], got %#v", status.Mismatched)
	}
	if len(db.queries) != 4 {
		t.Fatalf("expected 4 catalog queries (schema + 3 mirror tables), got %d: %#v", len(db.queries), db.queries)
	}
	if !hasQueryContaining(db.queries, "to_regnamespace") {
		t.Fatalf("expected a to_regnamespace query, got %#v", db.queries)
	}
	if !hasQueryContaining(db.queries, "to_regclass") {
		t.Fatalf("expected to_regclass queries, got %#v", db.queries)
	}
}
