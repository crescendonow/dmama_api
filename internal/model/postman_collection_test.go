package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPostmanCollectionHasStepTestCRUD(t *testing.T) {
	path := filepath.Join("..", "..", "note", "dmama_api.postman_collection.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read postman collection: %v", err)
	}

	var collection struct {
		Item []postmanItem `json:"item"`
	}
	if err := json.Unmarshal(raw, &collection); err != nil {
		t.Fatalf("postman collection must be valid JSON: %v", err)
	}

	expected := map[string]string{
		"features - validate step_test":    "POST",
		"features - create step_test":      "POST",
		"features - list step_test":        "GET",
		"features - get one step_test":     "GET",
		"features - update step_test":      "PUT",
		"features - delete step_test":      "DELETE",
		"features - sync step_test mirror": "POST",
	}
	seen := map[string]postmanItem{}
	for _, item := range collection.Item {
		if _, ok := expected[item.Name]; ok {
			seen[item.Name] = item
		}
	}
	if len(seen) != len(expected) {
		t.Fatalf("found %d step_test items, want %d", len(seen), len(expected))
	}

	for name, method := range expected {
		item := seen[name]
		if item.Request.Method != method {
			t.Fatalf("%s method = %s, want %s", name, item.Request.Method, method)
		}
		if !hasHeader(item, "X-API-Key") {
			t.Fatalf("%s missing X-API-Key header", name)
		}
		if method == "POST" || method == "PUT" || method == "DELETE" {
			if !hasHeader(item, "X-User-Id") {
				t.Fatalf("%s missing X-User-Id header", name)
			}
		}
	}

	for _, name := range []string{"features - get one step_test", "features - update step_test", "features - delete step_test"} {
		if got := seen[name].Request.URL.Raw; !strings.Contains(got, "{{step_test_id}}") {
			t.Fatalf("%s raw URL = %q, want {{step_test_id}}", name, got)
		}
	}

	for _, name := range []string{"features - validate step_test", "features - create step_test"} {
		assertStepTestCollectionPostmanBody(t, name, seen[name].Request.Body.Raw)
	}
	assertStepTestPostmanBody(t, "features - update step_test", seen["features - update step_test"].Request.Body.Raw)
}

type postmanItem struct {
	Name    string `json:"name"`
	Request struct {
		Method string `json:"method"`
		Header []struct {
			Key string `json:"key"`
		} `json:"header"`
		Body struct {
			Raw string `json:"raw"`
		} `json:"body"`
		URL struct {
			Raw string `json:"raw"`
		} `json:"url"`
	} `json:"request"`
}

func hasHeader(item postmanItem, key string) bool {
	for _, header := range item.Request.Header {
		if header.Key == key {
			return true
		}
	}
	return false
}

func assertStepTestPostmanBody(t *testing.T, name, raw string) {
	t.Helper()
	var body stepTestFixture
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("%s body must be JSON: %v", name, err)
	}
	if got, _ := body.Geometry["type"].(string); got != "Polygon" {
		t.Fatalf("%s geometry type = %q, want Polygon", name, got)
	}
	if got := body.Properties["pwaCode"]; got != "5521040" {
		t.Fatalf("%s pwaCode = %v, want 5521040", name, got)
	}
	for _, key := range []string{"_id", "_createdAt", "_createdBy", "_updatedAt", "_updatedBy"} {
		if _, ok := body.Properties[key]; ok {
			t.Fatalf("%s body still has server-managed property %s", name, key)
		}
	}
}

func TestPostmanCollectionHasCustomersAllEndpoints(t *testing.T) {
	path := filepath.Join("..", "..", "note", "dmama_api.postman_collection.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read postman collection: %v", err)
	}

	var collection struct {
		Item []postmanItem `json:"item"`
	}
	if err := json.Unmarshal(raw, &collection); err != nil {
		t.Fatalf("postman collection must be valid JSON: %v", err)
	}

	expectedMethods := map[string]string{
		"dma - customers-all (all regions)": "GET",
		"dma - customers-all (region)":      "GET",
		"dma - customers-all (branch)":      "GET",
		"dma - customers-all (dma ids)":     "GET",
		"dma - customers-all (polygon)":     "POST",
	}
	seen := map[string]postmanItem{}
	for _, item := range collection.Item {
		if _, ok := expectedMethods[item.Name]; ok {
			seen[item.Name] = item
		}
	}
	if len(seen) != len(expectedMethods) {
		t.Fatalf("found %d customers-all items, want %d (seen: %#v)", len(seen), len(expectedMethods), seen)
	}

	for name, method := range expectedMethods {
		item := seen[name]
		if item.Request.Method != method {
			t.Fatalf("%s method = %s, want %s", name, item.Request.Method, method)
		}
		if !hasHeader(item, "X-API-Key") {
			t.Fatalf("%s missing X-API-Key header", name)
		}
	}

	polygonItem := seen["dma - customers-all (polygon)"]
	var body struct {
		MyPolygon struct {
			Type string `json:"type"`
		} `json:"my_polygon"`
	}
	if err := json.Unmarshal([]byte(polygonItem.Request.Body.Raw), &body); err != nil {
		t.Fatalf("polygon item body must be JSON: %v", err)
	}
	if body.MyPolygon.Type != "Polygon" && body.MyPolygon.Type != "MultiPolygon" {
		t.Fatalf("polygon item my_polygon.type = %q, want Polygon or MultiPolygon", body.MyPolygon.Type)
	}

	dmaIDsItem := seen["dma - customers-all (dma ids)"]
	if !strings.Contains(dmaIDsItem.Request.URL.Raw, "dma_id=") {
		t.Fatalf("dma ids item raw URL = %q, want a dma_id query param", dmaIDsItem.Request.URL.Raw)
	}
}

func assertStepTestCollectionPostmanBody(t *testing.T, name, raw string) {
	t.Helper()
	var body struct {
		Type     string            `json:"type"`
		Features []stepTestFixture `json:"features"`
	}
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("%s body must be JSON: %v", name, err)
	}
	if body.Type != "FeatureCollection" || len(body.Features) < 2 {
		t.Fatalf("%s body = %#v, want FeatureCollection with at least two features", name, body)
	}
	for i := range body.Features {
		if got, _ := body.Features[i].Geometry["type"].(string); got != "Polygon" {
			t.Fatalf("%s feature %d geometry type = %q, want Polygon", name, i, got)
		}
		for _, key := range []string{"_id", "_createdAt", "_createdBy", "_updatedAt", "_updatedBy"} {
			if _, ok := body.Features[i].Properties[key]; ok {
				t.Fatalf("%s feature %d still has server-managed property %s", name, i, key)
			}
		}
	}
}
