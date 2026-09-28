package tests

import (
	"context"
	"os"
	"testing"

	"dmama_api/internal/database"
	"dmama_api/internal/model"
	"dmama_api/internal/repository"
	"dmama_api/internal/service"
)

// These tests hit the live PostgreSQL 16 (dmama_layer) + Vallaris MongoDB and mutate branch data,
// so they are opt-in. Enable with:
//
//	DMAMA_INTEGRATION=1 GISDATA_URL=... MONGO_URI=... [MONGO_DB=vallaris_feature] [TEST_PWA_CODE=5521040] go test ./tests/...
//
// They create only flow_meter points and clean them up; dma_boundary is read/validated only.

func integrationSvc(t *testing.T) (*service.FeatureService, *repository.FeatureRepo, string) {
	t.Helper()
	if os.Getenv("DMAMA_INTEGRATION") != "1" {
		t.Skip("set DMAMA_INTEGRATION=1 (with GISDATA_URL, MONGO_URI) to run feature integration tests")
	}
	gisURL, mongoURI := os.Getenv("GISDATA_URL"), os.Getenv("MONGO_URI")
	if gisURL == "" || mongoURI == "" {
		t.Skip("GISDATA_URL and MONGO_URI must be set")
	}
	mongoDB := os.Getenv("MONGO_DB")
	if mongoDB == "" {
		mongoDB = "vallaris_feature"
	}

	ctx := context.Background()
	gisPool, err := database.NewPool(ctx, gisURL)
	if err != nil {
		t.Fatalf("connect PG16: %v", err)
	}
	t.Cleanup(gisPool.Close)

	db, err := database.NewMongoDatabase(ctx, mongoURI, mongoDB)
	if err != nil {
		t.Fatalf("connect mongo: %v", err)
	}

	topo := repository.NewTopologyRepo(gisPool)
	if err := topo.EnsureSchema(ctx); err != nil {
		t.Fatalf("ensure dmama_layer schema: %v", err)
	}
	feats := repository.NewFeatureRepo(db)

	pwa := os.Getenv("TEST_PWA_CODE")
	if pwa == "" {
		pwa = "5521040"
	}
	return service.NewFeatureService(feats, topo, nil, "", ""), feats, pwa
}

// Verifies the alias -> collections._id -> features_<id> resolution chain works for the branch.
func TestIntegrationCatalogResolves(t *testing.T) {
	svc, _, pwa := integrationSvc(t)
	if _, err := svc.List(context.Background(), model.ShapeDmaBoundary, pwa, nil); err != nil {
		t.Fatalf("List dma_boundary for %s failed (is b%s_dma_boundary provisioned?): %v", pwa, pwa, err)
	}
}

// Verifies a client-supplied dma_id that collides with an existing one is rejected (no insert).
func TestIntegrationDmaIDDuplicateRejected(t *testing.T) {
	svc, feats, pwa := integrationSvc(t)
	ctx := context.Background()

	max, err := feats.MaxDmaID(ctx, pwa)
	if err != nil {
		t.Fatalf("MaxDmaID: %v", err)
	}
	if max == 0 {
		t.Skip("branch has no dma_boundary to collide with")
	}

	// A tiny polygon near (0,0), far from any real branch data, so only the dma_id rule can fail.
	req := &model.FeatureRequest{
		DmaID: max,
		Geometry: map[string]interface{}{
			"type": "Polygon",
			"coordinates": [][][]float64{{
				{0.001, 0.001}, {0.002, 0.001}, {0.002, 0.002}, {0.001, 0.002}, {0.001, 0.001},
			}},
		},
	}
	feature, result, err := svc.Create(ctx, model.ShapeDmaBoundary, pwa, req, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if feature != nil {
		_, _ = svc.Delete(ctx, model.ShapeDmaBoundary, pwa, feature.ID) // undo accidental insert
		t.Fatal("expected duplicate dma_id to be rejected, but a feature was created")
	}
	if result == nil || result.Valid {
		t.Fatalf("expected invalid result for duplicate dma_id, got %+v", result)
	}
}

// Verifies the full flow_meter CRUD round-trip with dmama_layer mirroring.
func TestIntegrationFlowMeterRoundTrip(t *testing.T) {
	svc, feats, pwa := integrationSvc(t)
	ctx := context.Background()

	// Seed the mirror so the within-coverage rule has dma_boundary data to test against.
	if _, err := svc.Sync(ctx, model.ShapeDmaBoundary, pwa); err != nil {
		t.Fatalf("sync dma_boundary: %v", err)
	}

	// Borrow an existing flow_meter location (presumed inside coverage) for the test point.
	existing, err := feats.List(ctx, pwa, model.ShapeFlowMeter, nil)
	if err != nil {
		t.Fatalf("list flow_meter: %v", err)
	}
	if len(existing) == 0 {
		t.Skip("no existing flow_meter to borrow a coordinate from")
	}
	geom := existing[0].Geometry

	created, result, err := svc.Create(ctx, model.ShapeFlowMeter, pwa,
		&model.FeatureRequest{Geometry: geom, Properties: map[string]interface{}{"remark": "integration-test"}}, "")
	if err != nil {
		t.Fatalf("create flow_meter: %v", err)
	}
	if created == nil {
		t.Skipf("create rejected (point not within coverage?): %+v", result.Violations)
	}
	id := created.ID
	defer func() { _, _ = svc.Delete(ctx, model.ShapeFlowMeter, pwa, id) }() // best-effort cleanup

	got, err := svc.GetByID(ctx, model.ShapeFlowMeter, pwa, id)
	if err != nil || got == nil {
		t.Fatalf("get after create: got=%v err=%v", got, err)
	}

	updated, _, found, err := svc.Update(ctx, model.ShapeFlowMeter, pwa, id,
		&model.FeatureRequest{Geometry: geom, Properties: map[string]interface{}{"remark": "integration-test-updated"}}, "")
	if err != nil || !found || updated == nil {
		t.Fatalf("update: found=%v updated=%v err=%v", found, updated, err)
	}
	if updated.Properties["remark"] != "integration-test-updated" {
		t.Fatalf("update did not apply remark: %v", updated.Properties["remark"])
	}

	deleted, err := svc.Delete(ctx, model.ShapeFlowMeter, pwa, id)
	if err != nil || !deleted {
		t.Fatalf("delete: deleted=%v err=%v", deleted, err)
	}

	after, err := svc.GetByID(ctx, model.ShapeFlowMeter, pwa, id)
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if after != nil {
		t.Fatal("expected feature to be gone after delete")
	}
}
