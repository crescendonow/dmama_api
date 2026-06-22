package repository

import (
	"context"
	"errors"
	"fmt"

	"dmama_api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TopologyRepo runs PostGIS-16 topology validation and maintains the dmama_layer mirror.
// Geometries are passed in as GeoJSON strings in EPSG:4326 (the CRS used by the frontend).
// The mirror tables double as the GiST-indexed neighbour source for set-based topology checks;
// MongoDB remains the source of truth, the mirror is rebuildable via SyncFromMongo.
type TopologyRepo struct {
	pool *pgxpool.Pool
}

func NewTopologyRepo(pool *pgxpool.Pool) *TopologyRepo {
	return &TopologyRepo{pool: pool}
}

const schemaName = "dmama_layer"

// mirrorTables maps a shape to its fully-qualified mirror table. Using a fixed whitelist keeps
// the table name out of any user-controlled SQL string.
var mirrorTables = map[string]string{
	model.ShapeDmaBoundary: schemaName + ".dma_boundary",
	model.ShapeFlowMeter:   schemaName + ".flow_meter",
	model.ShapeStepTest:    schemaName + ".step_test",
}

func tableFor(shape string) (string, error) {
	tbl, ok := mirrorTables[shape]
	if !ok {
		return "", fmt.Errorf("unknown shape: %s", shape)
	}
	return tbl, nil
}

// EnsureSchema creates the dmama_layer schema, the per-shape mirror tables, and their indexes.
// It is idempotent. PostGIS is assumed to be installed in the target database.
func (r *TopologyRepo) EnsureSchema(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS `+schemaName); err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}
	for _, name := range []string{"dma_boundary", "flow_meter", "step_test"} {
		full := schemaName + "." + name
		stmts := []string{
			fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
				mongo_id   text PRIMARY KEY,
				pwa_code   text NOT NULL,
				dma_id     integer,
				properties jsonb,
				geom       geometry(Geometry,4326),
				created_at timestamptz NOT NULL DEFAULT now(),
				updated_at timestamptz NOT NULL DEFAULT now()
			)`, full),
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s_geom_gix ON %s USING GIST (geom)`, name, full),
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s_pwa_idx ON %s (pwa_code)`, name, full),
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s_pwa_dma_idx ON %s (pwa_code, dma_id)`, name, full),
		}
		for _, s := range stmts {
			if _, err := r.pool.Exec(ctx, s); err != nil {
				return fmt.Errorf("ensure table %s: %w", full, err)
			}
		}
	}
	return nil
}

// CheckValidity reports whether a geometry is OGC-valid (covers self-intersection, ring
// orientation, etc.). Returns the reason when invalid.
func (r *TopologyRepo) CheckValidity(ctx context.Context, geomGeoJSON string) (valid bool, reason string, err error) {
	const q = `
		WITH g AS (SELECT ST_SetSRID(ST_GeomFromGeoJSON($1), 4326) AS geom)
		SELECT ST_IsValid(geom), ST_IsValidReason(geom) FROM g`
	if err := r.pool.QueryRow(ctx, q, geomGeoJSON).Scan(&valid, &reason); err != nil {
		return false, "", err
	}
	return valid, reason, nil
}

// OverlapsExisting reports whether the geometry improperly overlaps a sibling of the same shape
// in the same branch. Touching along a shared boundary is allowed; interior overlap is not.
// excludeMongoID (may be empty) skips the row being updated.
func (r *TopologyRepo) OverlapsExisting(ctx context.Context, shape, pwaCode, geomGeoJSON, excludeMongoID string) (bool, error) {
	tbl, err := tableFor(shape)
	if err != nil {
		return false, err
	}
	q := fmt.Sprintf(`
		WITH input AS (SELECT ST_MakeValid(ST_SetSRID(ST_GeomFromGeoJSON($1), 4326)) AS g)
		SELECT EXISTS (
			SELECT 1 FROM %s t, input
			WHERE t.pwa_code = $2
			  AND ($3 = '' OR t.mongo_id <> $3)
			  AND ST_Intersects(t.geom, input.g)
			  AND NOT ST_Touches(t.geom, input.g)
		)`, tbl)

	var overlaps bool
	if err := r.pool.QueryRow(ctx, q, geomGeoJSON, pwaCode, excludeMongoID).Scan(&overlaps); err != nil {
		return false, err
	}
	return overlaps, nil
}

// WithinDmaCoverage reports whether a geometry is covered by the union of the branch's dma_boundary
// polygons. hasCoverage is false when the branch has no dma_boundary mirrored yet (rule skipped).
func (r *TopologyRepo) WithinDmaCoverage(ctx context.Context, pwaCode, geomGeoJSON string) (within bool, hasCoverage bool, err error) {
	const q = `
		WITH input AS (SELECT ST_SetSRID(ST_GeomFromGeoJSON($1), 4326) AS g),
		cov AS (
			SELECT ST_Union(ST_MakeValid(geom)) AS g
			FROM dmama_layer.dma_boundary
			WHERE pwa_code = $2
		)
		SELECT (cov.g IS NOT NULL), COALESCE(ST_CoveredBy(input.g, cov.g), false)
		FROM input, cov`

	err = r.pool.QueryRow(ctx, q, geomGeoJSON, pwaCode).Scan(&hasCoverage, &within)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return within, hasCoverage, nil
}

// Upsert mirrors a feature into its dmama_layer table. dmaID is nil for non-boundary shapes.
func (r *TopologyRepo) Upsert(ctx context.Context, shape, mongoID, pwaCode string, dmaID *int, propertiesJSON, geomGeoJSON string) error {
	tbl, err := tableFor(shape)
	if err != nil {
		return err
	}
	q := fmt.Sprintf(`
		INSERT INTO %s (mongo_id, pwa_code, dma_id, properties, geom, updated_at)
		VALUES ($1, $2, $3, $4::jsonb, ST_SetSRID(ST_GeomFromGeoJSON($5), 4326), now())
		ON CONFLICT (mongo_id) DO UPDATE SET
			pwa_code   = EXCLUDED.pwa_code,
			dma_id     = EXCLUDED.dma_id,
			properties = EXCLUDED.properties,
			geom       = EXCLUDED.geom,
			updated_at = now()`, tbl)
	_, err = r.pool.Exec(ctx, q, mongoID, pwaCode, dmaID, propertiesJSON, geomGeoJSON)
	return err
}

// Delete removes a feature's mirror row. A missing row is not an error.
func (r *TopologyRepo) Delete(ctx context.Context, shape, mongoID string) error {
	tbl, err := tableFor(shape)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE mongo_id = $1`, tbl), mongoID)
	return err
}
