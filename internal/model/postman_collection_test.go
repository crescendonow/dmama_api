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

	for _, name := range []string{"features - validate step_test", "features - create step_test", "features - update step_test"} {
		assertStepTestPostmanBody(t, name, seen[name].Request.Body.Raw)
	}
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
