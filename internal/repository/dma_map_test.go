package repository

import (
	"context"
	"errors"
	"strings"
	"testing"

	"dmama_api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/text/encoding/charmap"
)

type fakeMapDB struct {
	rows  pgx.Rows
	query string
	args  []any
}

func (*fakeMapDB) QueryRow(context.Context, string, ...any) pgx.Row {
	return nil
}

func (f *fakeMapDB) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	f.query = query
	f.args = append([]any(nil), args...)
	if f.rows == nil {
		return nil, errors.New("unexpected Query")
	}
	return f.rows, nil
}

type fakeMapRows struct {
	scans []func(dest ...any) error
	index int
	err   error
}

func (*fakeMapRows) Close() {}

func (r *fakeMapRows) Err() error {
	return r.err
}

func (*fakeMapRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (*fakeMapRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *fakeMapRows) Next() bool {
	if r.index >= len(r.scans) {
		return false
	}
	r.index++
	return true
}

func (r *fakeMapRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.scans) {
		return errors.New("Scan called without a current row")
	}
	return r.scans[r.index-1](dest...)
}

func (*fakeMapRows) Values() ([]any, error) {
	return nil, errors.New("Values is not implemented")
}

func (*fakeMapRows) RawValues() [][]byte {
	return nil
}

func (*fakeMapRows) Conn() *pgx.Conn {
	return nil
}

var _ dmaDB = (*fakeMapDB)(nil)
var _ pgx.Rows = (*fakeMapRows)(nil)

func TestDMARepoGetMapDataReturnsSeparateDecodedDMAFields(t *testing.T) {
	const utf8Name = "DMA \u0e0a\u0e37\u0e48\u0e2d"
	encodedName, err := charmap.Windows874.NewEncoder().String(utf8Name)
	if err != nil {
		t.Fatalf("failed to encode fixture: %v", err)
	}
	dmaNo := "DMA-001"
	geometry := `{"type":"Polygon","coordinates":[]}`
	db := &fakeMapDB{
		rows: &fakeMapRows{
			scans: []func(dest ...any) error{
				func(dest ...any) error {
					if len(dest) != 6 {
						t.Fatalf("expected 6 scan destinations, got %d", len(dest))
					}
					*dest[0].(*string) = "5531011-1"
					*dest[1].(*string) = "5531011"
					*dest[2].(*string) = "1"
					*dest[3].(**string) = &dmaNo
					*dest[4].(**string) = &encodedName
					*dest[5].(**string) = &geometry
					return nil
				},
			},
		},
	}

	items, err := newDMARepo(db).GetMapData(context.Background(), model.FormatGeoJSON, nil, nil)
	if err != nil {
		t.Fatalf("GetMapData returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one item, got %#v", items)
	}
	item := items[0]
	if item.DmaNo == nil || *item.DmaNo != dmaNo {
		t.Fatalf("expected dma_no %q, got %#v", dmaNo, item.DmaNo)
	}
	if item.DmaName == nil || *item.DmaName != utf8Name {
		t.Fatalf("expected decoded dma_name %q, got %#v", utf8Name, item.DmaName)
	}
	query := strings.Join(strings.Fields(db.query), " ")
	if !strings.Contains(query, "SELECT concat(pwa_code, '-', dma_id) AS id, pwa_code, dma_id, dma_no, dma_name,") {
		t.Fatalf("expected map query to select dma_no and dma_name, got %s", query)
	}
	if strings.Contains(query, "dma_no AS name") {
		t.Fatalf("expected map query not to alias dma_no as name, got %s", query)
	}
}
