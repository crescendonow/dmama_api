package service

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDMAStatsDocumentationDescribesYearMonthContract(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to locate test source")
	}
	contents, err := os.ReadFile(filepath.Join(filepath.Dir(sourceFile), "..", "..", "template", "index.html"))
	if err != nil {
		t.Fatalf("read API documentation: %v", err)
	}

	for _, testCase := range []struct {
		name     string
		start    string
		required []string
	}{
		{"stats", "<!-- EP5: DMA Stats -->", []string{"year_month", "Buddhist Era", "YYYYMM", "days 1-15", "day 16 through month end", "use_water", "use_jan..use_dec", `year_month: ""`, "lstwtusg1", "256907"}},
		{"stats-region", "<!-- EP19: DMA Stats Region -->", []string{"year_month", "Buddhist Era", "YYYYMM", "days 1-15", "day 16 through month end", "lstwtusg1..12", "prswtusg", "256908"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			sectionStart := strings.Index(string(contents), testCase.start)
			if sectionStart < 0 {
				t.Fatalf("documentation section %q not found", testCase.start)
			}
			section := string(contents[sectionStart:])
			if sectionEnd := strings.Index(section, "</section>"); sectionEnd >= 0 {
				section = section[:sectionEnd]
			}
			for _, text := range testCase.required {
				if !strings.Contains(section, text) {
					t.Errorf("missing %q", text)
				}
			}
		})
	}
}
