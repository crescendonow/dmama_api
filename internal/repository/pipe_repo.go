package repository

import (
	"context"
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

// GetGeometry fetches pipe geometries by pipe_id and/or pwa_code. (Endpoint 2)
func (r *PipeRepo) GetGeometry(ctx context.Context, region int, pipeIDs []string, pwaCode string, format model.GeomFormat) ([]model.Pipe, error) {
	tbl := TableName("oracle", region, "pipe")
	geomExpr := model.SQLGeomExpr("wkb_geometry", format)

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

	if pwaCode != "" {
		conditions = append(conditions, fmt.Sprintf("pwa_code = $%d", argIdx))
		args = append(args, pwaCode)
	}

	if len(conditions) == 0 {
		return nil, nil
	}

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

// GetAssetsOnPipe finds meter/valve/leakpoint on a pipe's geometry. (Endpoint 12)
func (r *PipeRepo) GetAssetsOnPipe(ctx context.Context, region int, pipeWkbGeometry string, assetType string, format model.GeomFormat) ([]map[string]interface{}, error) {
	var tbl string
	var fields string
	geomExpr := model.SQLGeomExpr(assetType+".wkb_geometry", format)

	switch assetType {
	case "meter":
		tbl = TableName("oracle", region, "meter")
		fields = fmt.Sprintf("%s.ogc_fid, %s.custcode, %s", assetType, assetType, geomExpr)
	case "valve":
		tbl = TableName("oracle", region, "valve")
		fields = fmt.Sprintf("%s.ogc_fid, %s", assetType, geomExpr)
	case "leakpoint":
		tbl = TableName("oracle", region, "leakpoint")
		fields = fmt.Sprintf("%s.ogc_fid, %s", assetType, geomExpr)
	default:
		return nil, fmt.Errorf("invalid asset type: %s", assetType)
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM %s AS %s
		WHERE ST_Intersects($1, %s.wkb_geometry)`,
		fields, tbl, assetType, assetType)

	rows, err := r.pool.Query(ctx, query, pipeWkbGeometry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	fieldDescs := rows.FieldDescriptions()
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, fd := range fieldDescs {
			row[string(fd.Name)] = vals[i]
		}
		results = append(results, row)
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
