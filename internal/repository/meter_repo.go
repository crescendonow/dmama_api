package repository

import (
	"context"
	"fmt"

	"dmama_api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MeterRepo struct {
	pool *pgxpool.Pool
}

func NewMeterRepo(pool *pgxpool.Pool) *MeterRepo {
	return &MeterRepo{pool: pool}
}

// CountDailyMetersInDMA counts active meters within a DMA boundary.
func (r *MeterRepo) CountDailyMetersInDMA(ctx context.Context, region int, pwaCode, dmaID string) (*model.DMADailyMeterCount, error) {
	tbl := TableName("oracle", region, "meter")
	query := fmt.Sprintf(`
		SELECT COALESCE(COUNT(*), 0) AS meter_count
		FROM pwa_dma.dma_boundary AS dma
		JOIN %s AS meter
			ON meter.pwa_code = dma.pwa_code
			AND meter.custstat IN ('1', '2', '3', '4')
			AND ST_Intersects(
				dma.wkb_geometry,
				CASE
					WHEN ST_SRID(meter.wkb_geometry) = ST_SRID(dma.wkb_geometry) THEN meter.wkb_geometry
					WHEN ST_SRID(meter.wkb_geometry) = 0 THEN ST_SetSRID(meter.wkb_geometry, ST_SRID(dma.wkb_geometry))
					ELSE ST_Transform(meter.wkb_geometry, ST_SRID(dma.wkb_geometry))
				END
			)
		WHERE dma.pwa_code = $1
			AND dma.dma_id = $2`, tbl)

	var result model.DMADailyMeterCount
	err := r.pool.QueryRow(ctx, query, pwaCode, dmaID).Scan(&result.MeterCount)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
