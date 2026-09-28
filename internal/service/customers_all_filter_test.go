package service

import (
	"reflect"
	"strings"
	"testing"

	"dmama_api/internal/repository"
)

func TestParseCustomersAllQueryDefaultsToAllRegions(t *testing.T) {
	filter, err := ParseCustomersAllQuery("", "", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !reflect.DeepEqual(filter.Regions, repository.AllRegions) {
		t.Fatalf("expected all regions, got %v", filter.Regions)
	}
}

func TestParseCustomersAllQueryRegionOnly(t *testing.T) {
	filter, err := ParseCustomersAllQuery("3", "", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !reflect.DeepEqual(filter.Regions, []int{3}) {
		t.Fatalf("expected region [3], got %v", filter.Regions)
	}
}

func TestParseCustomersAllQueryPWACodeResolvesRegion(t *testing.T) {
	filter, err := ParseCustomersAllQuery("", "5531011", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !reflect.DeepEqual(filter.Regions, []int{1}) {
		t.Fatalf("expected region [1] resolved from pwa_code, got %v", filter.Regions)
	}
	if filter.PwaCode != "5531011" {
		t.Fatalf("expected pwa_code to be kept, got %q", filter.PwaCode)
	}
}

func TestParseCustomersAllQueryRejectsMismatchedRegionAndPWACode(t *testing.T) {
	_, err := ParseCustomersAllQuery("2", "5531011", "", "")
	if err == nil {
		t.Fatal("expected error for region/pwa_code mismatch")
	}
	if !strings.Contains(err.Error(), "does not belong to the requested region") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestParseCustomersAllQueryRejectsDmaIDWithoutPWACode(t *testing.T) {
	_, err := ParseCustomersAllQuery("1", "", "1,2", "")
	if err == nil {
		t.Fatal("expected error when dma_id given without pwa_code")
	}
	if !strings.Contains(err.Error(), "pwa_code is required when dma_id is provided") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestParseCustomersAllQueryDedupesAndSortsDmaIDs(t *testing.T) {
	filter, err := ParseCustomersAllQuery("1", "5531011", "1, 2,2,10", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !reflect.DeepEqual(filter.DmaIDs, []int{1, 2, 10}) {
		t.Fatalf("expected [1 2 10], got %v", filter.DmaIDs)
	}
}

func TestParseCustomersAllQueryRejectsNonIntegerDmaID(t *testing.T) {
	_, err := ParseCustomersAllQuery("1", "5531011", "a", "")
	if err == nil {
		t.Fatal("expected error for non-integer dma_id")
	}
	if !strings.Contains(err.Error(), "dma_id must be a comma-separated list of integers") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestParseCustomersAllQueryAllowsValidUsetypeList(t *testing.T) {
	filter, err := ParseCustomersAllQuery("", "", "", "22,35")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !reflect.DeepEqual(filter.Usetypes, []string{"22", "35"}) {
		t.Fatalf("expected [22 35], got %v", filter.Usetypes)
	}
}

func TestParseCustomersAllQueryRejectsNonNumericUsetype(t *testing.T) {
	_, err := ParseCustomersAllQuery("", "", "", "2x")
	if err == nil {
		t.Fatal("expected error for non-numeric usetype")
	}
	if !strings.Contains(err.Error(), "usetype must be a comma-separated list of numeric codes") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestParseCustomersAllQueryRejectsInvalidRegion(t *testing.T) {
	_, err := ParseCustomersAllQuery("11", "", "", "")
	if err == nil {
		t.Fatal("expected error for region 11")
	}
	if !strings.Contains(err.Error(), "region must be 1-10") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestParseCustomersAllQueryRejectsUnknownPWACodePrefix(t *testing.T) {
	_, err := ParseCustomersAllQuery("", "9999011", "", "")
	if err == nil {
		t.Fatal("expected error for unknown pwa_code prefix")
	}
	if !strings.Contains(err.Error(), "unknown pwa_code prefix") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// ---- POST body parsing ----

func TestParseCustomersAllBodyRequiresMyPolygon(t *testing.T) {
	_, err := ParseCustomersAllBody([]byte(`{"region":1}`))
	if err == nil {
		t.Fatal("expected error for missing my_polygon")
	}
	if !strings.Contains(err.Error(), "my_polygon is required") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestParseCustomersAllBodyRejectsInvalidJSON(t *testing.T) {
	_, err := ParseCustomersAllBody([]byte(`{not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseCustomersAllBodyRejectsWrongGeometryType(t *testing.T) {
	_, err := ParseCustomersAllBody([]byte(`{"my_polygon":{"type":"Point","coordinates":[100,13]}}`))
	if err == nil {
		t.Fatal("expected error for Point geometry")
	}
	if !strings.Contains(err.Error(), "Polygon or MultiPolygon") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestParseCustomersAllBodyAcceptsMultiPolygon(t *testing.T) {
	body := `{"my_polygon":{"type":"MultiPolygon","coordinates":[[[[100,13],[101,13],[101,14],[100,13]]]]}}`
	filter, err := ParseCustomersAllBody([]byte(body))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if filter.PolygonGeoJSON == "" {
		t.Fatal("expected PolygonGeoJSON to be set")
	}
}

func TestParseCustomersAllBodyParsesArraysAndFilters(t *testing.T) {
	body := `{
		"region": 1,
		"pwa_code": "5531011",
		"dma_id": [10, 2, 2, 1],
		"usetype": ["22", "35"],
		"my_polygon": {"type": "Polygon", "coordinates": [[[101.68,13.98],[101.70,13.98],[101.70,14.00],[101.68,13.98]]]}
	}`
	filter, err := ParseCustomersAllBody([]byte(body))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !reflect.DeepEqual(filter.Regions, []int{1}) {
		t.Fatalf("expected region [1], got %v", filter.Regions)
	}
	if filter.PwaCode != "5531011" {
		t.Fatalf("expected pwa_code 5531011, got %q", filter.PwaCode)
	}
	if !reflect.DeepEqual(filter.DmaIDs, []int{1, 2, 10}) {
		t.Fatalf("expected dma_id [1 2 10], got %v", filter.DmaIDs)
	}
	if !reflect.DeepEqual(filter.Usetypes, []string{"22", "35"}) {
		t.Fatalf("expected usetype [22 35], got %v", filter.Usetypes)
	}
}

func TestParseCustomersAllBodyRejectsDmaIDWithoutPWACode(t *testing.T) {
	body := `{"dma_id":[1],"my_polygon":{"type":"Polygon","coordinates":[[[100,13],[101,13],[101,14],[100,13]]]}}`
	_, err := ParseCustomersAllBody([]byte(body))
	if err == nil {
		t.Fatal("expected error when dma_id given without pwa_code")
	}
	if !strings.Contains(err.Error(), "pwa_code is required when dma_id is provided") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestParseCustomersAllBodyRejectsMismatchedRegionAndPWACode(t *testing.T) {
	body := `{"region":2,"pwa_code":"5531011","my_polygon":{"type":"Polygon","coordinates":[[[100,13],[101,13],[101,14],[100,13]]]}}`
	_, err := ParseCustomersAllBody([]byte(body))
	if err == nil {
		t.Fatal("expected error for region/pwa_code mismatch")
	}
	if !strings.Contains(err.Error(), "does not belong to the requested region") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
