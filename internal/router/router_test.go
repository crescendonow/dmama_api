package router

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"dmama_api/internal/config"

	"github.com/gofiber/fiber/v2"
)

func TestSetupRegistersDMAStatsRegion(t *testing.T) {
	app := fiber.New()
	Setup(app, nil, nil, nil, nil, nil, nil, &config.Config{})

	req := httptest.NewRequest("GET", "/api/dma/stats-region?column=prswtusg", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected registered stats-region route to reject missing region with 400, got %d", resp.StatusCode)
	}
}
func TestSetupRegistersDMACustomersAllGetAndPost(t *testing.T) {
	app := fiber.New()
	Setup(app, nil, nil, nil, nil, nil, nil, &config.Config{})

	getReq := httptest.NewRequest("GET", "/api/dma/customers-all?region=99", nil)
	getResp, err := app.Test(getReq)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if getResp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected registered GET customers-all route to reject region=99 with 400, got %d", getResp.StatusCode)
	}

	postReq := httptest.NewRequest("POST", "/api/dma/customers-all", strings.NewReader(`{}`))
	postReq.Header.Set("Content-Type", "application/json")
	postResp, err := app.Test(postReq)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if postResp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected registered POST customers-all route to reject missing my_polygon with 400, got %d", postResp.StatusCode)
	}
}

func TestSetupServesMonitorFromTemplateStaticPath(t *testing.T) {
	app := fiber.New()
	Setup(app, nil, nil, nil, nil, nil, nil, &config.Config{})

	body := requestBody(t, app, "/template/api_monitor.html", fiber.StatusOK)
	if !strings.Contains(body, "DMAMA API Monitor") {
		t.Fatalf("expected template monitor page, got body: %q", body)
	}
}

func TestSetupServesDocsMonitorAliases(t *testing.T) {
	app := fiber.New()
	Setup(app, nil, nil, nil, nil, nil, nil, &config.Config{})

	for _, path := range []string{"/docs/api-monitor", "/docs/api_monitor.html"} {
		body := requestBody(t, app, path, fiber.StatusOK)
		if !strings.Contains(body, "DMAMA API Monitor") {
			t.Fatalf("expected monitor page at %s, got body: %q", path, body)
		}
	}
}

func TestDocsIndexUsesPrefixSafeMonitorLink(t *testing.T) {
	app := fiber.New()
	Setup(app, nil, nil, nil, nil, nil, nil, &config.Config{})

	body := requestBody(t, app, "/docs/", fiber.StatusOK)
	if !strings.Contains(body, `href="./api_monitor.html"`) {
		t.Fatalf("expected docs index to link to monitor with a relative URL")
	}
	if strings.Contains(body, `href="/docs/api-monitor"`) {
		t.Fatalf("expected docs index not to link to monitor with a root-relative URL")
	}
}

func TestMonitorPageUsesPrefixSafeLinks(t *testing.T) {
	app := fiber.New()
	Setup(app, nil, nil, nil, nil, nil, nil, &config.Config{})

	body := requestBody(t, app, "/docs/api_monitor.html", fiber.StatusOK)
	for _, want := range []string{
		`href="./static/css/api_monitor.css`,
		`href="./index.html"`,
		`src="./static/js/api_monitor.js`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected monitor page to contain %q", want)
		}
	}
	for _, bad := range []string{
		`href="/docs`,
		`src="/docs`,
	} {
		if strings.Contains(body, bad) {
			t.Fatalf("expected monitor page not to contain root-relative docs link %q", bad)
		}
	}
}

// PATCH has never been a feature route; with the gate attached per-route instead of as a group
// Use, Fiber's own router now answers (405, with an Allow header) as it normally would for an
// unmapped verb on a path it does recognise, instead of the pre-fix `api.All("/features/*", ...)`
// catch-all's 503 implying the whole feature backend -- or a PATCH route -- exists.
func TestSetupPatchOnFeaturesReturns404NotServiceUnavailable(t *testing.T) {
	app := fiber.New()
	Setup(app, nil, nil, nil, nil, nil, nil, &config.Config{})

	req := httptest.NewRequest("PATCH", "/api/features/dma_boundary/5521040", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode == fiber.StatusServiceUnavailable {
		t.Fatalf("expected PATCH /api/features/* not to 503 (that's the bug this fix removes), got %d", resp.StatusCode)
	}
	if resp.StatusCode != fiber.StatusMethodNotAllowed {
		t.Fatalf("expected Fiber's default 405 for an unmapped verb on a known path, got %d", resp.StatusCode)
	}
}

func requestBody(t *testing.T, app *fiber.App, path string, wantStatus int) string {
	t.Helper()

	req := httptest.NewRequest("GET", path, nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}
	body := string(bodyBytes)
	if resp.StatusCode != wantStatus {
		t.Fatalf("expected %s to return %d, got %d with body %q", path, wantStatus, resp.StatusCode, body)
	}
	return body
}
