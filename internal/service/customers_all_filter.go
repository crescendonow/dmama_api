package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"dmama_api/internal/model"
	"dmama_api/internal/repository"
)

// ParseCustomersAllQuery resolves and validates the GET query parameters for
// /api/dma/customers-all (see note/22_plan_for_customers_endpoint.md). It never touches the
// database: region/pwa_code cross-checks use the static prefix table in the repository package.
func ParseCustomersAllQuery(region, pwaCode, dmaID, usetype string) (model.CustomersAllFilter, error) {
	resolvedRegion, err := resolveCustomersAllRegion(region, pwaCode)
	if err != nil {
		return model.CustomersAllFilter{}, err
	}

	if dmaID != "" && pwaCode == "" {
		return model.CustomersAllFilter{}, fmt.Errorf("pwa_code is required when dma_id is provided")
	}

	dmaIDs, err := parseDmaIDList(dmaID)
	if err != nil {
		return model.CustomersAllFilter{}, err
	}

	usetypes, err := parseUsetypeList(usetype)
	if err != nil {
		return model.CustomersAllFilter{}, err
	}

	return model.CustomersAllFilter{
		Regions:  regionsFor(resolvedRegion),
		PwaCode:  pwaCode,
		DmaIDs:   dmaIDs,
		Usetypes: usetypes,
	}, nil
}

// customersAllRequestBody mirrors the JSON body accepted by POST /api/dma/customers-all.
// MyPolygon is kept as raw JSON so it can be validated as a whole GeoJSON geometry (type check
// here, ST_IsValid check against PostGIS in the repository layer) and passed through unmodified.
type customersAllRequestBody struct {
	Region    *int            `json:"region"`
	PwaCode   string          `json:"pwa_code"`
	DmaID     []int           `json:"dma_id"`
	Usetype   []string        `json:"usetype"`
	MyPolygon json.RawMessage `json:"my_polygon"`
}

// ParseCustomersAllBody resolves and validates the JSON body for POST /api/dma/customers-all.
// my_polygon is required; every other filter follows the same rules as the GET form.
func ParseCustomersAllBody(raw []byte) (model.CustomersAllFilter, error) {
	var body customersAllRequestBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return model.CustomersAllFilter{}, fmt.Errorf("request body must be valid JSON: %w", err)
	}

	if len(body.MyPolygon) == 0 || string(body.MyPolygon) == "null" {
		return model.CustomersAllFilter{}, fmt.Errorf("my_polygon is required")
	}
	var geometry struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body.MyPolygon, &geometry); err != nil {
		return model.CustomersAllFilter{}, fmt.Errorf("my_polygon must be a GeoJSON geometry object")
	}
	if geometry.Type != "Polygon" && geometry.Type != "MultiPolygon" {
		return model.CustomersAllFilter{}, fmt.Errorf("my_polygon.type must be Polygon or MultiPolygon")
	}

	regionStr := ""
	if body.Region != nil {
		regionStr = strconv.Itoa(*body.Region)
	}
	resolvedRegion, err := resolveCustomersAllRegion(regionStr, body.PwaCode)
	if err != nil {
		return model.CustomersAllFilter{}, err
	}

	if len(body.DmaID) > 0 && body.PwaCode == "" {
		return model.CustomersAllFilter{}, fmt.Errorf("pwa_code is required when dma_id is provided")
	}

	usetypes, err := validateUsetypes(body.Usetype)
	if err != nil {
		return model.CustomersAllFilter{}, err
	}

	return model.CustomersAllFilter{
		Regions:        regionsFor(resolvedRegion),
		PwaCode:        body.PwaCode,
		DmaIDs:         dedupeSortInts(body.DmaID),
		Usetypes:       usetypes,
		PolygonGeoJSON: string(body.MyPolygon),
	}, nil
}

// resolveCustomersAllRegion applies the flexible region/pwa_code hierarchy: region may be
// omitted and is resolved from pwa_code; if both are given they must match. Returns 0 when
// neither was supplied (meaning: every region).
func resolveCustomersAllRegion(regionStr, pwaCode string) (int, error) {
	resolvedRegion := 0
	if regionStr != "" {
		region, err := strconv.Atoi(regionStr)
		if err != nil || !repository.ValidRegion(region) {
			return 0, fmt.Errorf("region must be 1-10")
		}
		resolvedRegion = region
	}

	if pwaCode != "" {
		pwaRegion, err := repository.RegionFromPWACode(pwaCode)
		if err != nil {
			return 0, err
		}
		if resolvedRegion != 0 && resolvedRegion != pwaRegion {
			return 0, fmt.Errorf("pwa_code does not belong to the requested region")
		}
		resolvedRegion = pwaRegion
	}

	return resolvedRegion, nil
}

func regionsFor(resolvedRegion int) []int {
	if resolvedRegion == 0 {
		return repository.AllRegions
	}
	return []int{resolvedRegion}
}

// parseDmaIDList parses a comma-separated list of dma_id integers, trims whitespace around each
// entry, removes duplicates, and sorts the result ascending for deterministic query building.
func parseDmaIDList(raw string) ([]int, error) {
	if raw == "" {
		return nil, nil
	}
	seen := map[int]bool{}
	var result []int
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("dma_id must be a comma-separated list of integers")
		}
		if !seen[n] {
			seen[n] = true
			result = append(result, n)
		}
	}
	sort.Ints(result)
	return result, nil
}

func dedupeSortInts(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	seen := map[int]bool{}
	var result []int
	for _, n := range values {
		if !seen[n] {
			seen[n] = true
			result = append(result, n)
		}
	}
	sort.Ints(result)
	return result
}

// parseUsetypeList parses a comma-separated list of usetype codes; each code must be digits only.
func parseUsetypeList(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var parts []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parts = append(parts, part)
	}
	return validateUsetypes(parts)
}

func validateUsetypes(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		if value == "" || !isDigitsOnly(value) {
			return nil, fmt.Errorf("usetype must be a comma-separated list of numeric codes")
		}
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result, nil
}

func isDigitsOnly(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
