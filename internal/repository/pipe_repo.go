package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"dmama_api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PipeRepo struct {
	pool *pgxpool.Pool
}

func NewPipeRepo(pool *pgxpool.Pool) *PipeRepo {
	return &PipeRepo{pool: pool}
}

type pipeAssetSource struct {
	Schema string
	Entity string
	Alias  string
}

func pipeAssetTable(assetType string) (*pipeAssetSource, error) {
	switch assetType {
	case "meter":
		return &pipeAssetSource{Schema: "oracle", Entity: "meter", Alias: "meter"}, nil
	case "bl_customer":
		return &pipeAssetSource{Schema: "giswebm_stamp", Entity: "bl_customer", Alias: "bl_customer"}, nil
	case "valve":
		return &pipeAssetSource{Schema: "oracle", Entity: "valve", Alias: "valve"}, nil
	case "leakpoint":
		return &pipeAssetSource{Schema: "oracle", Entity: "leakpoint", Alias: "leakpoint"}, nil
	case "firehydrant":
		return &pipeAssetSource{Schema: "oracle", Entity: "firehydrant", Alias: "firehydrant"}, nil
	default:
		return nil, fmt.Errorf("invalid asset type: %s", assetType)
	}
}

// FindNearestPipe finds the nearest pipe within radius meters of a point. (Endpoint 1)
func (r *PipeRepo) FindNearestPipe(ctx context.Context, region int, lng, lat, radius float64) (*model.NearestPipe, error) {
	tbl := TableName("oracle", region, "pipe")
	query := fmt.Sprintf(`
		SELECT pipe_id, pwa_code
		FROM %s AS pipe
		WHERE ST_DWithin(ST_Point($1, $2)::geography, pipe.wkb_geometry::geography, $3)
		ORDER BY ST_Distance(ST_Point($1, $2)::geography, pipe.wkb_geometry::geography) ASC
		LIMIT 1`, tbl)

	var result model.NearestPipe
	err := r.pool.QueryRow(ctx, query, lng, lat, radius).Scan(&result.PipeID, &result.PwaCode)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

// GetGeometry fetches pipe geometries by pwa_code and optional pipe_id. (Endpoint 2)
func (r *PipeRepo) GetGeometry(ctx context.Context, region int, pipeIDs []string, pwaCode string, format model.GeomFormat) ([]model.Pipe, error) {
	tbl := TableName("oracle", region, "pipe")
	geomExpr := model.SQLGeomExpr("wkb_geometry", format)

	if pwaCode == "" {
		return nil, fmt.Errorf("pwa_code is required")
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	if len(pipeIDs) > 0 {
		placeholders := make([]string, len(pipeIDs))
		for i, id := range pipeIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, id)
			argIdx++
		}
		conditions = append(conditions, fmt.Sprintf("pipe_id IN (%s)", strings.Join(placeholders, ",")))
	}

	conditions = append(conditions, fmt.Sprintf("pwa_code = $%d", argIdx))
	args = append(args, pwaCode)

	query := fmt.Sprintf(`
		SELECT %s, pipe_id, pwa_code
		FROM %s
		WHERE %s`,
		geomExpr, tbl, strings.Join(conditions, " AND "))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pipes []model.Pipe
	for rows.Next() {
		var p model.Pipe
		if err := rows.Scan(&p.Geometry, &p.PipeID, &p.PwaCode); err != nil {
			return nil, err
		}
		pipes = append(pipes, p)
	}
	return pipes, nil
}

// GetPipeAtLeakpoint finds the pipe at a leakpoint's geometry. (Endpoint 11)
func (r *PipeRepo) GetPipeAtLeakpoint(ctx context.Context, region int, lng, lat float64) (*model.PipeInfo, error) {
	tbl := TableName("oracle", region, "pipe")
	query := fmt.Sprintf(`
		SELECT pipe_size, pipe_type, yearinstall,
			ST_DistanceSphere(
				ST_ClosestPoint(wkb_geometry, ST_SetSRID(ST_MakePoint($1,$2),4326)),
				ST_SetSRID(ST_MakePoint($1,$2),4326)
			) AS distance
		FROM %s AS pipe
		WHERE ST_Intersects(
			ST_Buffer(ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, 10)::geometry,
			wkb_geometry
		)
		ORDER BY distance ASC
		LIMIT 1`, tbl)

	var result model.PipeInfo
	err := r.pool.QueryRow(ctx, query, lng, lat).Scan(
		&result.PipeSize, &result.PipeType, &result.YearInstall, &result.Distance,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

// GetAssetsOnPipe finds assets on a pipe's geometry. (Endpoint 12)
func (r *PipeRepo) GetAssetsOnPipe(ctx context.Context, region int, pipeWkbGeometry string, lng, lat *float64, assetType string, format model.GeomFormat) (interface{}, error) {
	source, err := pipeAssetTable(assetType)
	if err != nil {
		return nil, err
	}

	pipeTbl := TableName("oracle", region, "pipe")
	assetTbl := TableName(source.Schema, region, source.Entity)

	var pipeCTE string
	var args []interface{}
	if pipeWkbGeometry != "" {
		pipeCTE = `
		pipe AS (
			SELECT $1::geometry AS geom
		)`
		args = append(args, pipeWkbGeometry)
	} else {
		if lng == nil || lat == nil {
			return nil, fmt.Errorf("wkb_geometry is required, or provide both lng and lat")
		}
		pipeCTE = fmt.Sprintf(`
		pipe AS (
			SELECT pipe.wkb_geometry AS geom
			FROM %s AS pipe
			WHERE ST_Intersects(
				ST_Buffer(ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, 10)::geometry,
				pipe.wkb_geometry
			)
			ORDER BY ST_DistanceSphere(
				ST_ClosestPoint(pipe.wkb_geometry, ST_SetSRID(ST_MakePoint($1, $2), 4326)),
				ST_SetSRID(ST_MakePoint($1, $2), 4326)
			) ASC
			LIMIT 1
		)`, pipeTbl)
		args = append(args, *lng, *lat)
	}

	if format == model.FormatGeoJSON {
		query := fmt.Sprintf(`
			WITH %s
			SELECT COALESCE(
				json_agg(
					json_build_object(
						'type', 'Feature',
						'geometry', ST_AsGeoJSON(%s.wkb_geometry)::json,
						'properties', row_to_json(%s)
					)
				),
				'[]'::json
			)
			FROM %s AS %s
			CROSS JOIN pipe
			WHERE pipe.geom IS NOT NULL
				AND ST_Intersects(pipe.geom, %s.wkb_geometry)`,
			pipeCTE, source.Alias, source.Alias, assetTbl, source.Alias, source.Alias)

		var featuresRaw []byte
		if err := r.pool.QueryRow(ctx, query, args...).Scan(&featuresRaw); err != nil {
			return nil, err
		}

		var features []map[string]interface{}
		if err := json.Unmarshal(featuresRaw, &features); err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"type":     "FeatureCollection",
			"features": features,
		}, nil
	}

	query := fmt.Sprintf(`
		WITH %s
		SELECT COALESCE(
			json_agg(
				row_to_json(row_data)
			),
			'[]'::json
		)
		FROM (
			SELECT %s.*, ST_AsText(%s.wkb_geometry) AS geometry
			FROM %s AS %s
			CROSS JOIN pipe
			WHERE pipe.geom IS NOT NULL
				AND ST_Intersects(pipe.geom, %s.wkb_geometry)
		) AS row_data`,
		pipeCTE, source.Alias, source.Alias, assetTbl, source.Alias, source.Alias)

	var rowsRaw []byte
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&rowsRaw); err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(rowsRaw, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// SumPipeLengthInDMA calculates total pipe length within a DMA boundary. (Endpoint 6)
func (r *PipeRepo) SumPipeLengthInDMA(ctx context.Context, region int, dmaWkbGeometry string) (*model.DMAPipeLength, error) {
	tbl := TableName("oracle", region, "pipe")
	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(pipe_long), 0) AS c
		FROM %s AS mt
		WHERE ST_Contains($1::geometry, mt.wkb_geometry)`, tbl)

	var result model.DMAPipeLength
	err := r.pool.QueryRow(ctx, query, dmaWkbGeometry).Scan(&result.TotalLength)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ClippedPipeLengthInDMA calculates pipe length clipped to DMA boundary in km. (Endpoint 16)
func (r *PipeRepo) ClippedPipeLengthInDMA(ctx context.Context, region int, pwaCode string, dmaNo string) (*model.DMAPipeLengthClipped, error) {
	tbl := TableName("oracle", region, "pipe")
	query := fmt.Sprintf(`
		SELECT COALESCE(TRUNC(SUM(cl.p_long)::DECIMAL/1000, 2), 0) AS sum_long
		FROM (
			SELECT ST_Length(
				(ST_Dump(ST_Intersection(dma.wkb_geometry, pp.wkb_geometry))).geom::geography
			) AS p_long
			FROM (SELECT * FROM %s
				  WHERE pwa_code = $1 AND pipe_func NOT IN (5,6)) pp,
				 (SELECT * FROM pwa_dma.dma_boundary
				  WHERE pwa_code = $1 AND dma_id = $2) dma
		) cl`, tbl)

	var result model.DMAPipeLengthClipped
	err := r.pool.QueryRow(ctx, query, pwaCode, dmaNo).Scan(&result.SumLongKm)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
