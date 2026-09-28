package repository

import (
	"strings"
	"testing"

	"dmama_api/internal/model"

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

func TestValidateStatsRegionColumnAllowsOnlyBillingColumns(t *testing.T) {
	allowed := []string{
		"prswtusg",
		"lstwtusg1", "lstwtusg2", "lstwtusg3", "lstwtusg4",
		"lstwtusg5", "lstwtusg6", "lstwtusg7", "lstwtusg8",
		"lstwtusg9", "lstwtusg10", "lstwtusg11", "lstwtusg12",
	}
	for _, column := range allowed {
		t.Run("allows_"+column, func(t *testing.T) {
			if err := ValidateStatsRegionColumn(column); err != nil {
				t.Fatalf("expected %s to be allowed: %v", column, err)
			}
		})
	}

	for _, column := range []string{"use_jan", "lstwtusg13", "prswtusg;drop table"} {
		t.Run("rejects_"+column, func(t *testing.T) {
			if err := ValidateStatsRegionColumn(column); err == nil {
				t.Fatalf("expected %s to be rejected", column)
			}
		})
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
	query, prefix, err := dmaStatsRegionQuery(9, "prswtusg", "")
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
	query, _, err := dmaStatsRegionQuery(1, "lstwtusg1", "")
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
func TestDMAStatsRegionQueryUsesExactPWACodeWhenSupplied(t *testing.T) {
	query, pwaCode, err := dmaStatsRegionQuery(1, "prswtusg", "5531011")
	if err != nil {
		t.Fatalf("dmaStatsRegionQuery returned error: %v", err)
	}
	if pwaCode != "5531011" {
		t.Fatalf("bound pwa_code = %q, want 5531011", pwaCode)
	}
	if !strings.Contains(query, "WHERE dma.pwa_code = $1") {
		t.Fatalf("expected exact pwa_code filter, query was %s", query)
	}
	if strings.Contains(query, "WHERE dma.pwa_code LIKE $1") {
		t.Fatalf("did not expect region prefix filter, query was %s", query)
	}
}
func TestDMAStatsRegionQueryRejectsPWACodeFromAnotherRegion(t *testing.T) {
	_, _, err := dmaStatsRegionQuery(1, "prswtusg", "5541011")
	if err == nil {
		t.Fatal("expected pwa_code from region 2 to be rejected for region 1")
	}
}

// ---- customersAllQuery ----

func TestCustomersAllQueryUsesRegionTable(t *testing.T) {
	query, _ := customersAllQuery(8, model.CustomersAllFilter{})
	if !strings.Contains(query, "giswebm_stamp.r8_bl_customer") {
		t.Fatalf("expected region 8 customer table, query was %s", query)
	}
}

func TestCustomersAllQueryHasLateralJoinShape(t *testing.T) {
	query, _ := customersAllQuery(1, model.CustomersAllFilter{})
	required := []string{
		"LEFT JOIN LATERAL",
		"ORDER BY b.dma_id",
		"LIMIT 1",
		"FROM pwa_dma.dma_boundary b",
	}
	for _, fragment := range required {
		if !strings.Contains(query, fragment) {
			t.Fatalf("expected query to contain %q, query was %s", fragment, query)
		}
	}
}

func TestCustomersAllQueryNoFiltersHasNoOptionalClauses(t *testing.T) {
	query, args := customersAllQuery(1, model.CustomersAllFilter{})
	if len(args) != 0 {
		t.Fatalf("expected no bound args, got %v", args)
	}
	forbidden := []string{"ANY(", "d.dma_id IS NOT NULL", "ST_GeomFromGeoJSON", "c.pwa_code ="}
	for _, fragment := range forbidden {
		if strings.Contains(query, fragment) {
			t.Fatalf("did not expect query to contain %q, query was %s", fragment, query)
		}
	}
	if !strings.Contains(query, "WHERE TRUE") {
		t.Fatalf("expected base WHERE TRUE, query was %s", query)
	}
}

func TestCustomersAllQueryDmaIDsAddAnyFilterAndNotNullFilter(t *testing.T) {
	query, args := customersAllQuery(1, model.CustomersAllFilter{PwaCode: "5531011", DmaIDs: []int{1, 2, 10}})
	if !strings.Contains(query, "AND b.dma_id = ANY($1::int[])") {
		t.Fatalf("expected dma_id ANY filter inside LATERAL join, query was %s", query)
	}
	if !strings.Contains(query, "d.dma_id IS NOT NULL") {
		t.Fatalf("expected d.dma_id IS NOT NULL filter when dma_id given without polygon, query was %s", query)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 bound args (dma ids, pwa_code), got %d: %v", len(args), args)
	}
	ids, ok := args[0].([]int32)
	if !ok || len(ids) != 3 {
		t.Fatalf("expected first arg to be []int32 of length 3, got %#v", args[0])
	}
}

func TestCustomersAllQueryPolygonOmitsNotNullFilterEvenWithDmaIDs(t *testing.T) {
	query, args := customersAllQuery(1, model.CustomersAllFilter{
		DmaIDs:         []int{1, 2},
		PolygonGeoJSON: `{"type":"Polygon","coordinates":[]}`,
	})
	if strings.Contains(query, "d.dma_id IS NOT NULL") {
		t.Fatalf("did not expect d.dma_id IS NOT NULL when a polygon is present, query was %s", query)
	}
	if !strings.Contains(query, "ST_GeomFromGeoJSON") || !strings.Contains(query, "ST_Intersects(ST_SetSRID(ST_GeomFromGeoJSON") {
		t.Fatalf("expected polygon clause, query was %s", query)
	}
	if !strings.Contains(query, "AND b.dma_id = ANY($1::int[])") {
		t.Fatalf("expected dma_id to still restrict the reported DMA, query was %s", query)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 bound args (dma ids, polygon), got %d: %v", len(args), args)
	}
}

func TestCustomersAllQueryUsetypesUseTextArrayFilter(t *testing.T) {
	query, args := customersAllQuery(1, model.CustomersAllFilter{Usetypes: []string{"22", "35"}})
	if !strings.Contains(query, "c.usetype = ANY($1::text[])") {
		t.Fatalf("expected usetype ANY filter, query was %s", query)
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 bound arg, got %d: %v", len(args), args)
	}
	values, ok := args[0].([]string)
	if !ok || len(values) != 2 {
		t.Fatalf("expected first arg to be []string of length 2, got %#v", args[0])
	}
}

func TestCustomersAllQueryArgOrderMatchesPlaceholders(t *testing.T) {
	query, args := customersAllQuery(1, model.CustomersAllFilter{
		PwaCode:  "5531011",
		DmaIDs:   []int{1},
		Usetypes: []string{"22"},
	})
	if len(args) != 3 {
		t.Fatalf("expected 3 bound args (dma ids, pwa_code, usetype), got %d: %v", len(args), args)
	}
	if !strings.Contains(query, "ANY($1::int[])") {
		t.Fatalf("expected dma_id bound to $1, query was %s", query)
	}
	if !strings.Contains(query, "c.pwa_code = $2") {
		t.Fatalf("expected pwa_code bound to $2, query was %s", query)
	}
	if !strings.Contains(query, "c.usetype = ANY($3::text[])") {
		t.Fatalf("expected usetype bound to $3, query was %s", query)
	}
}
