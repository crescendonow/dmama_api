package repository

import (
	"context"
	"fmt"

	"dmama_api/internal/model"

	"github.com/jackc/pgx/v5"
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
func (r *WaterworksRepo) GetByType(ctx context.Context, stationType *int, format model.GeomFormat) ([]model.Waterworks, error) {
	geomExpr := model.SQLGeomExpr("wkb_geometry", format)

	query := fmt.Sprintf(`SELECT pwa_code, name, %s FROM pwa_office.pwa_waterworks_2024`, geomExpr)
	if stationType != nil {
		query += " WHERE pwa_station = $1"
	}
	query += " ORDER BY pwa_code::INTEGER, pwa_station"

	var (
		rows pgx.Rows
		err  error
	)
	if stationType != nil {
		rows, err = r.pool.Query(ctx, query, *stationType)
	} else {
		rows, err = r.pool.Query(ctx, query)
	}
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
		w.Name = decodeDBText(w.Name)
		results = append(results, w)
	}
	return results, nil
}
