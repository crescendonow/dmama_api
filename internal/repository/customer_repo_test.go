package repository

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/text/encoding/charmap"
)

func TestValidateColumnAllowsStatsColumns(t *testing.T) {
	columns := []string{"prswtusg", "lstwtusg1", "lstwtusg11", "lstwtusg12"}

	for _, column := range columns {
		t.Run(column, func(t *testing.T) {
			if err := ValidateColumn(column); err != nil {
				t.Fatalf("expected %s to be allowed: %v", column, err)
			}
		})
	}
}

func TestValidateColumnRejectsUnsafeColumn(t *testing.T) {
	if err := ValidateColumn("lstwtusg13"); err == nil {
		t.Fatal("expected lstwtusg13 to be rejected")
	}
	if err := ValidateColumn("prswtusg;drop table"); err == nil {
		t.Fatal("expected SQL-like column to be rejected")
	}
}

func TestDMACustomersQueryUsesRegionTable(t *testing.T) {
	query := dmaCustomersQuery(8)

	if !strings.Contains(query, "giswebm_stamp.r8_bl_customer") {
		t.Fatalf("expected region 8 customer table, query was %s", query)
	}
}

func TestDMACustomersQueryIncludesSpatialAndCustomerFilters(t *testing.T) {
	query := dmaCustomersQuery(1)

	required := []string{
		"c.pwa_code = d.pwa_code",
		"d.wkb_geometry && c.wkb_geometry",
		"ST_Intersects(d.wkb_geometry, c.wkb_geometry)",
		"c.usetype IN ('22','35')",
	}
	for _, fragment := range required {
		if !strings.Contains(query, fragment) {
			t.Fatalf("expected query to contain %q, query was %s", fragment, query)
		}
	}
}

func TestTextPtrDecodesWindows874Text(t *testing.T) {
	encoded, err := charmap.Windows874.NewEncoder().String("ผู้ใช้น้ำ")
	if err != nil {
		t.Fatalf("failed to encode fixture: %v", err)
	}

	result := textPtr(pgtype.Text{String: encoded, Valid: true})
	if result == nil {
		t.Fatal("expected decoded string, got nil")
	}
	if *result != "ผู้ใช้น้ำ" {
		t.Fatalf("expected decoded Thai text, got %q", *result)
	}
}

func TestDMAStatsRegionQueryUsesRegionTableAndPrefix(t *testing.T) {
	query, prefix, err := dmaStatsRegionQuery(9, "prswtusg")
	if err != nil {
		t.Fatalf("dmaStatsRegionQuery returned error: %v", err)
	}
	if prefix != "5511%" {
		t.Fatalf("expected region 9 prefix 5511%%, got %s", prefix)
	}
	if !strings.Contains(query, "giswebm_stamp.r9_bl_customer") {
		t.Fatalf("expected region 9 customer table, query was %s", query)
	}
	if !strings.Contains(query, "WHERE dma.pwa_code LIKE $1") {
		t.Fatalf("expected region prefix filter, query was %s", query)
	}
}

func TestDMAStatsRegionQueryIncludesGroupedSpatialLeftJoin(t *testing.T) {
	query, _, err := dmaStatsRegionQuery(1, "lstwtusg1")
	if err != nil {
		t.Fatalf("dmaStatsRegionQuery returned error: %v", err)
	}

	required := []string{
		"LEFT JOIN giswebm_stamp.r1_bl_customer AS bl",
		"ST_Intersects(",
		"GROUP BY dma.pwa_code, dma.dma_id",
		"ORDER BY dma.pwa_code, dma.dma_id",
		"SUM(bl.lstwtusg1)",
	}
	for _, fragment := range required {
		if !strings.Contains(query, fragment) {
			t.Fatalf("expected query to contain %q, query was %s", fragment, query)
		}
	}
}
