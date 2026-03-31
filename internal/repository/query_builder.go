package repository

import (
	"fmt"
	"strings"
)

var AllRegions = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

// ZonePrefixes maps district/region number to pwa_code 4-digit prefix.
var ZonePrefixes = map[int]string{
	1: "5531", 2: "5541", 3: "5542", 4: "5551", 5: "5552",
	6: "5521", 7: "5522", 8: "5532", 9: "5511", 10: "5512",
}

// prefixToRegion is the reverse lookup: pwa_code prefix => region.
var prefixToRegion map[string]int

func init() {
	prefixToRegion = make(map[string]int, len(ZonePrefixes))
	for region, prefix := range ZonePrefixes {
		prefixToRegion[prefix] = region
	}
}

// TableName builds a fully-qualified table name.
// Example: TableName("oracle", 3, "pipe") => "oracle.r3_pipe"
func TableName(schema string, region int, entity string) string {
	return fmt.Sprintf("%s.r%d_%s", schema, region, entity)
}

// QueryFunc receives the full table name and region, returns a SELECT statement.
type QueryFunc func(tableName string, region int) string

// UnionAll builds a UNION ALL across the given regions.
func UnionAll(regions []int, schema, entity string, queryFn QueryFunc) string {
	parts := make([]string, 0, len(regions))
	for _, r := range regions {
		tbl := TableName(schema, r, entity)
		parts = append(parts, "("+queryFn(tbl, r)+")")
	}
	return strings.Join(parts, "\nUNION ALL\n")
}

// ResolveRegions returns the list of region IDs to query.
// If region is specified (1-10), return just that one. Otherwise return all.
func ResolveRegions(region *int) []int {
	if region != nil && *region >= 1 && *region <= 10 {
		return []int{*region}
	}
	return AllRegions
}

// RegionFromPWACode resolves a pwa_code to its region using the first 4 digits.
// Returns 0 and error if the prefix is unknown.
func RegionFromPWACode(pwaCode string) (int, error) {
	if len(pwaCode) < 4 {
		return 0, fmt.Errorf("invalid pwa_code: too short")
	}
	prefix := pwaCode[:4]
	if r, ok := prefixToRegion[prefix]; ok {
		return r, nil
	}
	return 0, fmt.Errorf("unknown pwa_code prefix: %s", prefix)
}

// ValidRegion checks if region is between 1 and 10.
func ValidRegion(region int) bool {
	return region >= 1 && region <= 10
}

// ZonePrefix returns the pwa_code prefix for a given region.
func ZonePrefix(region int) (string, error) {
	if prefix, ok := ZonePrefixes[region]; ok {
		return prefix, nil
	}
	return "", fmt.Errorf("invalid region: %d", region)
}
