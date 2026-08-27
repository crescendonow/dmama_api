package router

import (
	"strings"
	"testing"

	"dmama_api/internal/config"

	"github.com/gofiber/fiber/v2"
)

func TestDocsShowFeatureCollectionRequestsForDmaBoundaryAndFlowMeter(t *testing.T) {
	app := fiber.New()
	Setup(app, nil, nil, nil, nil, nil, nil, &config.Config{})

	body := requestBody(t, app, "/docs/", fiber.StatusOK)
	for _, want := range []string{"dma_boundary FeatureCollection", "flow_meter FeatureCollection"} {
		if !strings.Contains(body, want) {
			t.Fatalf("docs did not include %q request example", want)
		}
	}
}
