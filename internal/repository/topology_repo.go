package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"dmama_api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const topologySchemaMigrationHint = "run migrations/0001_dmama_layer.sql on the database configured by GISDATA_URL"

const topologySchemaExistsSQL = `SELECT to_regnamespace($1::text) IS NOT NULL`

// topologyTableStatusSQL checks one dmama_layer mirror table against its expected shape. The
// predicate is LIKE, not =, because format_type schema-qualifies the PostGIS type (e.g.
// public.geometry(...)) whenever postgis is off the connection's search_path -- an exact match
// would report a correctly-provisioned table as broken.
const topologyTableStatusSQL = `
WITH target AS (
	SELECT to_regclass($1::text) AS oid
),
expected(column_name, expected_type, expected_not_null) AS (
	VALUES
		('mongo_id', 'text', true),
		('pwa_code', 'text', true),
		('dma_id', 'integer', false),
		('properties', 'jsonb', false),
		('geom', '%geometry(Geometry,4326)', false),
		('created_at', 'timestamp with time zone', true),
		('updated_at', 'timestamp with time zone', true)
)
SELECT
	target.oid IS NOT NULL AS table_exists,
	target.oid IS NOT NULL
		AND bool_and(
			a.attname IS NOT NULL
			AND format_type(a.atttypid, a.atttypmod) LIKE expected.expected_type
			AND a.attnotnull = expected.expected_not_null
		) AS schema_ready
FROM target
CROSS JOIN expected
LEFT JOIN pg_attribute a
	ON a.attrelid = target.oid
	AND a.attname = expected.column_name
	AND a.attnum > 0
	AND NOT a.attisdropped
GROUP BY target.oid`

type topoDB interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// TopologyRepo runs PostGIS-16 topology validation and maintains the dmama_layer mirror.
// Geometries are passed in as GeoJSON strings in EPSG:4326 (the CRS used by the frontend).
// The mirror tables double as the GiST-indexed neighbour source for set-based topology checks;
// MongoDB remains the source of truth, the mirror is rebuildable via SyncFromMongo.
type TopologyRepo struct {
	db topoDB
}

// NewTopologyRepo wraps a PostgreSQL 16 pool. A nil pool yields a repo with no db handle rather
// than a non-nil *TopologyRepo whose interior interface secretly holds a nil pointer -- callers
// (e.g. the feature readiness gate) rely on being able to tell "not configured" apart from
// "configured but broken".
func NewTopologyRepo(pool *pgxpool.Pool) *TopologyRepo {
	if pool == nil {
		return &TopologyRepo{}
	}
	return newTopologyRepo(pool)
}

func newTopologyRepo(db topoDB) *TopologyRepo {
	return &TopologyRepo{db: db}
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

// mirrorTableNames is the fixed, ordered list of dmama_layer mirror tables. A slice (not the
// mirrorTables map) keeps SchemaStatus/EnsureSchema output order stable across runs.
var mirrorTableNames = []string{"dma_boundary", "flow_meter", "step_test"}

// TopologySchemaStatus is the read-only result of probing dmama_layer against its expected shape.
// Ready means EnsureSchema needs to issue zero DDL.
type TopologySchemaStatus struct {
	SchemaExists  bool
	MissingTables []string
	Mismatched    []string
	Ready         bool
}

// SchemaStatus probes dmama_layer and its three mirror tables using read-only catalog lookups
// only (to_regnamespace/to_regclass + pg_attribute) -- it never issues DDL, so it is safe to call
// on every request (see handler.FeatureGate).
func (r *TopologyRepo) SchemaStatus(ctx context.Context) (TopologySchemaStatus, error) {
	var status TopologySchemaStatus
	if err := r.db.QueryRow(ctx, topologySchemaExistsSQL, schemaName).Scan(&status.SchemaExists); err != nil {
		return TopologySchemaStatus{}, err
	}

	ready := status.SchemaExists
	for _, name := range mirrorTableNames {
		var exists, tableReady bool
		full := schemaName + "." + name
		if err := r.db.QueryRow(ctx, topologyTableStatusSQL, full).Scan(&exists, &tableReady); err != nil {
			return TopologySchemaStatus{}, err
		}
		switch {
		case !exists:
			status.MissingTables = append(status.MissingTables, name)
			ready = false
		case !tableReady:
			status.Mismatched = append(status.Mismatched, name)
			ready = false
		}
	}
	status.Ready = ready
	return status, nil
}

// EnsureSchema verifies dmama_layer, then creates only what's genuinely missing. Production
// should provision this via migrations/0001_dmama_layer.sql so the app role only needs USAGE +
// DML; EnsureSchema still self-heals a genuinely missing schema/table (created by dmama, so its
// own CREATE INDEX later passes the ownership check).
func (r *TopologyRepo) EnsureSchema(ctx context.Context) error {
	status, err := r.SchemaStatus(ctx)
	if err != nil {
		return topologySchemaError("check dmama_layer schema", err)
	}
	if status.Ready {
		return nil
	}
	if len(status.Mismatched) > 0 {
		return fmt.Errorf("dmama_layer.%s exists but does not match the required schema; %s",
			strings.Join(status.Mismatched, ", "), topologySchemaMigrationHint)
	}

	if !status.SchemaExists {
		if _, err := r.db.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS `+schemaName); err != nil {
			return topologySchemaError("create dmama_layer schema", err)
		}
	}
	for _, name := range status.MissingTables {
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
			if _, err := r.db.Exec(ctx, s); err != nil {
				return topologySchemaError("create table "+full, err)
			}
		}
	}

	status, err = r.SchemaStatus(ctx)
	if err != nil {
		return topologySchemaError("validate dmama_layer schema", err)
	}
	if !status.Ready {
		return fmt.Errorf("dmama_layer still does not match the required schema after create; %s", topologySchemaMigrationHint)
	}
	return nil
}

func topologySchemaError(action string, err error) error {
	return fmt.Errorf("%s: %w; %s", action, err, topologySchemaMigrationHint)
}

// CheckValidity reports whether a geometry is OGC-valid (covers self-intersection, ring
// orientation, etc.). Returns the reason when invalid.
func (r *TopologyRepo) CheckValidity(ctx context.Context, geomGeoJSON string) (valid bool, reason string, err error) {
	const q = `
		WITH g AS (SELECT ST_SetSRID(ST_GeomFromGeoJSON($1), 4326) AS geom)
		SELECT ST_IsValid(geom), ST_IsValidReason(geom) FROM g`
	if err := r.db.QueryRow(ctx, q, geomGeoJSON).Scan(&valid, &reason); err != nil {
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
	if err := r.db.QueryRow(ctx, q, geomGeoJSON, pwaCode, excludeMongoID).Scan(&overlaps); err != nil {
		return false, err
	}
	return overlaps, nil
}

// CollectionOverlaps returns every pair of zero-based indexes whose interiors intersect.
// Features that only touch along their boundaries are intentionally omitted.
func (r *TopologyRepo) CollectionOverlaps(ctx context.Context, geometries []string) ([][2]int, error) {
	if len(geometries) < 2 {
		return nil, nil
	}
	payload, err := json.Marshal(geometries)
	if err != nil {
		return nil, fmt.Errorf("marshal collection geometries: %w", err)
	}

	const q = `
		WITH inputs AS (
			SELECT ordinality::int - 1 AS feature_index,
			       ST_MakeValid(ST_SetSRID(ST_GeomFromGeoJSON(value), 4326)) AS geom
			FROM jsonb_array_elements_text($1::jsonb) WITH ORDINALITY AS item(value, ordinality)
		), overlap_pairs AS (
			SELECT left_item.feature_index AS left_index,
			       right_item.feature_index AS right_index
			FROM inputs left_item
			JOIN inputs right_item ON left_item.feature_index < right_item.feature_index
			WHERE ST_Intersects(left_item.geom, right_item.geom)
			  AND NOT ST_Touches(left_item.geom, right_item.geom)
			ORDER BY left_item.feature_index, right_item.feature_index
		)
		SELECT COALESCE(
			jsonb_agg(jsonb_build_array(left_index, right_index)),
			'[]'::jsonb
		) FROM overlap_pairs`

	var raw []byte
	if err := r.db.QueryRow(ctx, q, payload).Scan(&raw); err != nil {
		return nil, err
	}
	var pairs [][2]int
	if err := json.Unmarshal(raw, &pairs); err != nil {
		return nil, fmt.Errorf("decode collection overlap pairs: %w", err)
	}
	return pairs, nil
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

	err = r.db.QueryRow(ctx, q, geomGeoJSON, pwaCode).Scan(&hasCoverage, &within)
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
	_, err = r.db.Exec(ctx, q, mongoID, pwaCode, dmaID, propertiesJSON, geomGeoJSON)
	return err
}

// Delete removes a feature's mirror row. A missing row is not an error.
func (r *TopologyRepo) Delete(ctx context.Context, shape, mongoID string) error {
	tbl, err := tableFor(shape)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE mongo_id = $1`, tbl), mongoID)
	return err
}
