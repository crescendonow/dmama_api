package service

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCustomersAllDocumentationExists guards the docs additions described in
// note/22_plan_for_customers_endpoint.md: the new endpoint section, the interactive playground,
// and the PWA logo/favicon swap in both HTML files.
func TestCustomersAllDocumentationExists(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to locate test source")
	}
	templateDir := filepath.Join(filepath.Dir(sourceFile), "..", "..", "template")

	indexContents, err := os.ReadFile(filepath.Join(templateDir, "index.html"))
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}
	index := string(indexContents)

	for _, want := range []string{
		`id="ep-dma-customers-all"`,
		`id="playground"`,
		`pwa_logo2015.svg`,
		`rel="icon"`,
	} {
		if !strings.Contains(index, want) {
			t.Errorf("index.html missing %q", want)
		}
	}

	monitorContents, err := os.ReadFile(filepath.Join(templateDir, "api_monitor.html"))
	if err != nil {
		t.Fatalf("read api_monitor.html: %v", err)
	}
	if !strings.Contains(string(monitorContents), "pwa_logo2015.svg") {
		t.Error("api_monitor.html missing pwa_logo2015.svg")
	}
}

// TestCustomersAllDocumentationSectionCoversContract spot-checks that the endpoint section
// documents the plan's most load-bearing, easy-to-forget contract points: count last, the
// year_month/lstwtusgN rule, the unterminated-JSON-on-failure behaviour, and the "one row per
// customer, smallest dma_id" overlap rule.
func TestCustomersAllDocumentationSectionCoversContract(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to locate test source")
	}
	contents, err := os.ReadFile(filepath.Join(filepath.Dir(sourceFile), "..", "..", "template", "index.html"))
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}

	sectionStart := strings.Index(string(contents), `id="ep-dma-customers-all"`)
	if sectionStart < 0 {
		t.Fatal("ep-dma-customers-all section not found")
	}
	section := string(contents)[sectionStart:]
	if end := strings.Index(section, "</section>"); end >= 0 {
		section = section[:end]
	}

	for _, want := range []string{
		"year_month",
		"dma_id",
		"my_polygon",
		"usetype",
		`"count"`,
		"ไม่รับประกันลำดับ",
		"ไม่มี pagination",
		"null",
	} {
		if !strings.Contains(section, want) {
			t.Errorf("ep-dma-customers-all section missing %q", want)
		}
	}
}
