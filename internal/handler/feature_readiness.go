package handler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"dmama_api/internal/model"
	"dmama_api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

// schemaProber is the read-only probe FeatureGate needs from *repository.TopologyRepo. Narrowing
// to an interface (rather than depending on the concrete type) lets tests fake the PostgreSQL leg
// without a live connection.
type schemaProber interface {
	SchemaStatus(ctx context.Context) (repository.TopologySchemaStatus, error)
}

// FeatureGate decides whether the feature CRUD routes are safe to serve and, when not, which
// backend leg is at fault. It never issues DDL -- probe calls only SchemaStatus, so no request
// path can ever fire schema-changing SQL.
//
// Caching is deliberately asymmetric: once Ready is true it is never re-probed (the hot path is a
// single mutex + bool; a backend that dies afterwards surfaces as a 500 from the query itself --
// this is a readiness gate, not a liveness gate). While not ready, a re-probe happens at most once
// per ttl, so a fix (e.g. running the migration) recovers feature CRUD without a restart.
type FeatureGate struct {
	topo      schemaProber
	featureDB *mongo.Database
	ttl       time.Duration

	mu        sync.Mutex
	status    model.FeatureStatus
	lastCheck time.Time
	now       func() time.Time
}

// NewFeatureGate builds a gate. topo/featureDB may be nil when the corresponding backend was
// unreachable or unconfigured at startup; the gate reports that as an unready leg instead of
// panicking.
func NewFeatureGate(topo *repository.TopologyRepo, featureDB *mongo.Database, ttl time.Duration) *FeatureGate {
	// A nil *repository.TopologyRepo must become a nil schemaProber, not an interface value that
	// merely wraps a nil pointer -- the same gotcha NewTopologyRepo itself guards against.
	var prober schemaProber
	if topo != nil {
		prober = topo
	}
	return newFeatureGate(prober, featureDB, ttl, time.Now)
}

func newFeatureGate(topo schemaProber, featureDB *mongo.Database, ttl time.Duration, now func() time.Time) *FeatureGate {
	return &FeatureGate{topo: topo, featureDB: featureDB, ttl: ttl, now: now}
}

// Status returns the current readiness, re-probing per the asymmetric TTL rule above.
func (g *FeatureGate) Status(ctx context.Context) model.FeatureStatus {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.status.Ready {
		return g.status
	}
	if !g.lastCheck.IsZero() && g.now().Sub(g.lastCheck) < g.ttl {
		return g.status
	}

	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	g.status = g.probe(probeCtx)
	g.lastCheck = g.now()
	return g.status
}

// Middleware gates every feature route with a 503 until Status reports ready. A nil receiver
// (backends never wired up, e.g. router tests) always answers 503 without touching topo/featureDB.
func (g *FeatureGate) Middleware() fiber.Handler {
	if g == nil {
		reason := "feature CRUD unavailable: feature gate not configured"
		return func(c *fiber.Ctx) error {
			return c.Status(503).JSON(model.APIResponse{
				Success: false,
				Error:   reason,
				Data:    model.FeatureStatus{Reason: reason},
			})
		}
	}
	return func(c *fiber.Ctx) error {
		status := g.Status(c.Context())
		if !status.Ready {
			return c.Status(503).JSON(model.APIResponse{Success: false, Error: status.Reason, Data: status})
		}
		return c.Next()
	}
}

// Readiness answers GET /api/monitor/readiness with the full per-leg status. Always 200: this is
// a diagnostics endpoint for the dashboard, not a load-balancer health probe.
func (g *FeatureGate) Readiness(c *fiber.Ctx) error {
	if g == nil {
		return c.JSON(model.SuccessResponse(model.FeatureStatus{Reason: "feature gate not configured"}))
	}
	return c.JSON(model.SuccessResponse(g.Status(c.Context())))
}

func (g *FeatureGate) probe(ctx context.Context) model.FeatureStatus {
	checkedAt := g.now().UTC().Format(time.RFC3339)

	if g.topo == nil {
		return model.FeatureStatus{
			Reason:    "PostgreSQL 16 (GISDATA_URL) is not configured or was unreachable at startup",
			CheckedAt: checkedAt,
		}
	}
	if g.featureDB == nil {
		return model.FeatureStatus{
			Postgres:  true,
			Reason:    "MongoDB (MONGO_URI) is not configured or was unreachable at startup",
			CheckedAt: checkedAt,
		}
	}

	st, err := g.topo.SchemaStatus(ctx)
	if err != nil {
		return model.FeatureStatus{
			Postgres:  true,
			Mongo:     true,
			Reason:    fmt.Sprintf("dmama_layer schema check failed: %v; run migrations/0001_dmama_layer.sql", err),
			CheckedAt: checkedAt,
		}
	}
	if !st.Ready {
		missing := append(append([]string{}, st.MissingTables...), st.Mismatched...)
		return model.FeatureStatus{
			Postgres:  true,
			Mongo:     true,
			Reason:    fmt.Sprintf("dmama_layer not ready (missing: %s); run migrations/0001_dmama_layer.sql", strings.Join(missing, ", ")),
			CheckedAt: checkedAt,
		}
	}

	return model.FeatureStatus{
		Ready:     true,
		Postgres:  true,
		Mongo:     true,
		Schema:    true,
		CheckedAt: checkedAt,
	}
}
