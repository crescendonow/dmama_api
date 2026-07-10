package repository

import (
	"context"
	"fmt"

	"dmama_api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type dmaDB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type DMARepo struct {
	db dmaDB
}

func NewDMARepo(pool *pgxpool.Pool) *DMARepo {
	return newDMARepo(pool)
}

func newDMARepo(db dmaDB) *DMARepo {
	return &DMARepo{db: db}
}

// GetAtLeakpoint finds the DMA boundary intersecting a leakpoint coordinate.
func (r *DMARepo) GetAtLeakpoint(ctx context.Context, lng, lat float64, pwaCode string) (*model.DMABoundary, error) {
	query := `
		SELECT pwa_code, dma_id, dma_no, dma_name
		FROM pwa_dma.dma_boundary
		WHERE ST_Intersects(
			wkb_geometry,
			ST_SetSRID(ST_MakePoint($1, $2), 4326)
		)`

	args := []interface{}{lng, lat}
	if pwaCode != "" {
		query += " AND pwa_code = $3"
		args = append(args, pwaCode)
	}

	query += " LIMIT 1"

	var result model.DMABoundary
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&result.PwaCode, &result.DmaID, &result.DmaNo, &result.DmaName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

// GetBoundary fetches a DMA boundary by pwa_code and dma_id. (Endpoint 3)
func (r *DMARepo) GetBoundary(ctx context.Context, pwaCode, dmaID string, format model.GeomFormat) (*model.DMABoundary, error) {
	geomExpr := model.SQLGeomExpr("wkb_geometry", format)
	query := fmt.Sprintf(`
		SELECT pwa_code, dma_id, dma_no, dma_name, %s, ST_AsText(wkb_geometry) AS wkb_raw
		FROM pwa_dma.dma_boundary
		WHERE pwa_code = $1 AND dma_id = $2`, geomExpr)

	var result model.DMABoundary
	err := r.db.QueryRow(ctx, query, pwaCode, dmaID).Scan(
		&result.PwaCode, &result.DmaID, &result.DmaNo, &result.DmaName, &result.Geometry, &result.WkbGeometry,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

// GetBoundaryRaw fetches DMA boundary raw geometry for spatial queries (internal use).
func (r *DMARepo) GetBoundaryRaw(ctx context.Context, pwaCode, dmaID string) (string, error) {
	query := `
		SELECT ST_AsEWKT(wkb_geometry)
		FROM pwa_dma.dma_boundary
		WHERE pwa_code = $1 AND dma_id = $2`

	var wkt string
	err := r.db.QueryRow(ctx, query, pwaCode, dmaID).Scan(&wkt)
	if err != nil {
		return "", err
	}
	return wkt, nil
}

// GetMapData fetches DMA boundaries for map display. (Endpoint 7)
func (r *DMARepo) GetMapData(ctx context.Context, format model.GeomFormat, pwaCode *string, ids []string) ([]model.DMAMapItem, error) {
	geomExpr := model.SQLGeomExpr("wkb_geometry", format)
	query := fmt.Sprintf(`
		SELECT concat(pwa_code, '-', dma_id) AS id, pwa_code, dma_id, dma_no AS name, %s
		FROM pwa_dma.dma_boundary`, geomExpr)

	var args []interface{}
	argIdx := 1

	if pwaCode != nil {
		query += fmt.Sprintf(" WHERE pwa_code = $%d", argIdx)
		args = append(args, *pwaCode)
		argIdx++
	}

	if len(ids) > 0 {
		if pwaCode != nil {
			query += " AND"
		} else {
			query += " WHERE"
		}
		placeholders := make([]string, len(ids))
		for i, id := range ids {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, id)
			argIdx++
		}
		query += fmt.Sprintf(" concat(pwa_code, '-', dma_id) IN (%s)", joinStrings(placeholders, ","))
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.DMAMapItem
	for rows.Next() {
		var item model.DMAMapItem
		if err := rows.Scan(&item.ID, &item.PwaCode, &item.DmaID, &item.Name, &item.Geometry); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// GetUsageV2 fetches DMA usage via direct spatial join. (Endpoint 14)
func (r *DMARepo) GetUsageV2(ctx context.Context, region int, pwaCode, dmaNo string) (*model.DMAUsageV2, error) {
	tbl := TableName("giswebm_stamp", region, "bl_customer")
	query := fmt.Sprintf(`
		SELECT dv.pwa_code, dv.dma_id, dv.dma_no, COALESCE(SUM(mt.prswtusg), 0) AS prswtusg
		FROM pwa_dma.dma_boundary dv
		LEFT JOIN %s mt
			ON ST_Intersects(dv.wkb_geometry, mt.wkb_geometry)
			AND mt.pwa_code = dv.pwa_code
		WHERE dv.pwa_code = $1 AND dv.dma_id = $2
		GROUP BY dv.pwa_code, dv.dma_id, dv.dma_no`, tbl)

	var result model.DMAUsageV2
	err := r.db.QueryRow(ctx, query, pwaCode, dmaNo).Scan(
		&result.PwaCode, &result.DmaID, &result.DmaNo, &result.Prswtusg,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
