package repository

import (
	"context"
	"fmt"

	"dmama_api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WaterworksRepo struct {
	pool *pgxpool.Pool
}

func NewWaterworksRepo(pool *pgxpool.Pool) *WaterworksRepo {
	return &WaterworksRepo{pool: pool}
}

// GetByType fetches waterworks markers by station type across all regions. (Endpoint 9)
// type: 3=โรงผลิตน้ำ, 7=โรงสูบน้ำ, etc.
func (r *WaterworksRepo) GetByType(ctx context.Context, stationType int, format model.GeomFormat) ([]model.Waterworks, error) {
	geomExpr := model.SQLGeomExpr("wkb_geometry", format)

	query := UnionAll(AllRegions, "public", "pwa_waterworks", func(tbl string, region int) string {
		return fmt.Sprintf(
			`SELECT pwa_code, name, %s FROM %s WHERE pwa_station = $1`,
			geomExpr, tbl)
	})

	rows, err := r.pool.Query(ctx, query, stationType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.Waterworks
	for rows.Next() {
		var w model.Waterworks
		if err := rows.Scan(&w.PwaCode, &w.Name, &w.Geometry); err != nil {
			return nil, err
		}
		results = append(results, w)
	}
	return results, nil
}
