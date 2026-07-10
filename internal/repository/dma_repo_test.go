package repository

import (
	"context"
	"errors"
	"strings"
	"testing"

	"dmama_api/internal/model"

	"github.com/jackc/pgx/v5"
)

type fakeDMADB struct {
	row   pgx.Row
	query string
	args  []any
}

func (f *fakeDMADB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	f.query = query
	f.args = append([]any(nil), args...)
	return f.row
}

func (*fakeDMADB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected Query")
}

type fakeDMARow struct {
	scan func(dest ...any) error
}

func (r fakeDMARow) Scan(dest ...any) error {
	return r.scan(dest...)
}

var (
	_ dmaDB   = (*fakeDMADB)(nil)
	_ pgx.Row = fakeDMARow{}
)

func TestDMARepoGetBoundaryReturnsDMAName(t *testing.T) {
	dmaName := "DMA Zone One"
	dmaNo := "DMA-001"
	geometry := `{"type":"Polygon","coordinates":[]}`
	wkbGeometry := "POLYGON EMPTY"
	db := &fakeDMADB{
		row: fakeDMARow{scan: func(dest ...any) error {
			if len(dest) != 6 {
				t.Fatalf("expected 6 scan destinations, got %d", len(dest))
			}
			*dest[0].(*string) = "5531011"
			*dest[1].(*string) = "1"
			*dest[2].(**string) = &dmaNo
			*dest[3].(**string) = &dmaName
			*dest[4].(**string) = &geometry
			*dest[5].(**string) = &wkbGeometry
			return nil
		}},
	}

	result, err := newDMARepo(db).GetBoundary(context.Background(), "5531011", "1", model.FormatGeoJSON)
	if err != nil {
		t.Fatalf("GetBoundary returned error: %v", err)
	}
	if result == nil || result.DmaName == nil || *result.DmaName != dmaName {
		t.Fatalf("expected dma_name %q, got %#v", dmaName, result)
	}
	query := strings.Join(strings.Fields(db.query), " ")
	if !strings.Contains(query, "SELECT pwa_code, dma_id, dma_no, dma_name,") {
		t.Fatalf("expected GetBoundary to select dma_name, query was %s", query)
	}
}

func TestDMARepoGetAtLeakpointPreservesNullDMAName(t *testing.T) {
	dmaNo := "DMA-001"
	db := &fakeDMADB{
		row: fakeDMARow{scan: func(dest ...any) error {
			if len(dest) != 4 {
				t.Fatalf("expected 4 scan destinations, got %d", len(dest))
			}
			*dest[0].(*string) = "5531011"
			*dest[1].(*string) = "1"
			*dest[2].(**string) = &dmaNo
			*dest[3].(**string) = nil
			return nil
		}},
	}

	result, err := newDMARepo(db).GetAtLeakpoint(context.Background(), 101.6893, 13.9849, "5531011")
	if err != nil {
		t.Fatalf("GetAtLeakpoint returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected a DMA boundary, got nil")
	}
	if result.DmaName != nil {
		t.Fatalf("expected nil dma_name, got %q", *result.DmaName)
	}
	query := strings.Join(strings.Fields(db.query), " ")
	if !strings.Contains(query, "SELECT pwa_code, dma_id, dma_no, dma_name") {
		t.Fatalf("expected GetAtLeakpoint to select dma_name, query was %s", query)
	}
}
