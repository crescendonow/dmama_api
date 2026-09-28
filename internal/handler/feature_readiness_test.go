package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dmama_api/internal/model"
	"dmama_api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

// fakeSchemaProber fakes repository.TopologyRepo's SchemaStatus for tests that must not touch a
// live PostgreSQL connection.
type fakeSchemaProber struct {
	status repository.TopologySchemaStatus
	err    error
	calls  int
}

func (f *fakeSchemaProber) SchemaStatus(ctx context.Context) (repository.TopologySchemaStatus, error) {
	f.calls++
	return f.status, f.err
}

func TestFeatureGateNilReturns503WithReason(t *testing.T) {
	var gate *FeatureGate
	app := fiber.New()
	app.Get("/x", gate.Middleware(), func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest("GET", "/x", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("expected a nil gate to answer 503, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}
	var got model.APIResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Success {
		t.Fatal("expected success=false")
	}
	if got.Error == "" {
		t.Fatal("expected a non-empty reason")
	}
}

func TestFeatureGateReportsSchemaLegSeparatelyFromPoolLeg(t *testing.T) {
	cases := []struct {
		name       string
		topo       schemaProber
		featureDB  *mongo.Database
		wantPG     bool
		wantMongo  bool
		wantReason string
	}{
		{
			name:       "postgres not configured",
			topo:       nil,
			featureDB:  &mongo.Database{},
			wantReason: "GISDATA_URL",
		},
		{
			name:       "mongo not configured",
			topo:       &fakeSchemaProber{status: repository.TopologySchemaStatus{Ready: true}},
			featureDB:  nil,
			wantPG:     true,
			wantReason: "MONGO_URI",
		},
		{
			name:       "schema probe errored",
			topo:       &fakeSchemaProber{err: errors.New("boom")},
			featureDB:  &mongo.Database{},
			wantPG:     true,
			wantMongo:  true,
			wantReason: "schema check failed",
		},
		{
			name:       "schema not ready",
			topo:       &fakeSchemaProber{status: repository.TopologySchemaStatus{MissingTables: []string{"flow_meter"}}},
			featureDB:  &mongo.Database{},
			wantPG:     true,
			wantMongo:  true,
			wantReason: "flow_meter",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gate := newFeatureGate(tc.topo, tc.featureDB, time.Minute, time.Now)
			status := gate.Status(context.Background())
			if status.Ready {
				t.Fatal("expected Ready=false")
			}
			if status.Postgres != tc.wantPG {
				t.Errorf("Postgres = %v, want %v", status.Postgres, tc.wantPG)
			}
			if status.Mongo != tc.wantMongo {
				t.Errorf("Mongo = %v, want %v", status.Mongo, tc.wantMongo)
			}
			if status.Schema {
				t.Error("expected Schema=false")
			}
			if !strings.Contains(status.Reason, tc.wantReason) {
				t.Errorf("Reason = %q, want to contain %q", status.Reason, tc.wantReason)
			}
			if status.CheckedAt == "" {
				t.Error("expected CheckedAt to be set")
			}
		})
	}
}

func TestFeatureGateRecoversWithoutRestart(t *testing.T) {
	prober := &fakeSchemaProber{status: repository.TopologySchemaStatus{MissingTables: []string{"flow_meter"}}}
	current := time.Unix(0, 0)
	clock := func() time.Time { return current }
	gate := newFeatureGate(prober, &mongo.Database{}, 30*time.Second, clock)

	if gate.Status(context.Background()).Ready {
		t.Fatal("expected not ready before the schema is fixed")
	}

	prober.status = repository.TopologySchemaStatus{Ready: true}
	current = current.Add(31 * time.Second)

	if !gate.Status(context.Background()).Ready {
		t.Fatal("expected ready once the TTL elapses and the schema is fixed, without a restart")
	}
}

func TestFeatureGateDoesNotReprobeOnceReady(t *testing.T) {
	prober := &fakeSchemaProber{status: repository.TopologySchemaStatus{Ready: true}}
	gate := newFeatureGate(prober, &mongo.Database{}, 30*time.Second, time.Now)

	for i := 0; i < 3; i++ {
		if !gate.Status(context.Background()).Ready {
			t.Fatalf("request %d: expected ready", i)
		}
	}
	if prober.calls != 1 {
		t.Fatalf("expected exactly 1 probe over 3 requests once ready, got %d", prober.calls)
	}
}

func TestFeatureGateReprobeIsRateLimitedByTTL(t *testing.T) {
	prober := &fakeSchemaProber{status: repository.TopologySchemaStatus{MissingTables: []string{"flow_meter"}}}
	current := time.Unix(0, 0)
	clock := func() time.Time { return current }
	gate := newFeatureGate(prober, &mongo.Database{}, 30*time.Second, clock)

	gate.Status(context.Background())
	gate.Status(context.Background())
	if prober.calls != 1 {
		t.Fatalf("expected the TTL to suppress the second probe, got %d calls", prober.calls)
	}

	current = current.Add(31 * time.Second)
	gate.Status(context.Background())
	if prober.calls != 2 {
		t.Fatalf("expected a re-probe once the TTL elapsed, got %d calls", prober.calls)
	}
}
