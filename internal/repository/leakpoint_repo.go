package repository

import (
	"context"
	"fmt"
	"time"

	"dmama_api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LeakpointRepo struct {
	pool *pgxpool.Pool
}

func NewLeakpointRepo(pool *pgxpool.Pool) *LeakpointRepo {
	return &LeakpointRepo{pool: pool}
}

// GetLeakpointsBySizeInDMA counts leakpoints by pipe size within a DMA (last 5 years). (Endpoint 15)
func (r *LeakpointRepo) GetLeakpointsBySizeInDMA(ctx context.Context, region int, pwaCode, dmaID string) (*model.DMALeakpointsBySize, error) {
	tbl := TableName("oracle", region, "leakpoint")
	yearBE := time.Now().Year() + 543
	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(CASE WHEN pipe_size >= 100 AND pipe_size < 200 THEN 1 ELSE 0 END), 0) AS psize_100,
			COALESCE(SUM(CASE WHEN pipe_size >= 200 AND pipe_size < 300 THEN 1 ELSE 0 END), 0) AS psize_200,
			COALESCE(SUM(CASE WHEN pipe_size >= 300 AND pipe_size < 400 THEN 1 ELSE 0 END), 0) AS psize_300,
			COALESCE(SUM(CASE WHEN pipe_size >= 400 AND pipe_size < 500 THEN 1 ELSE 0 END), 0) AS psize_400,
			COALESCE(SUM(CASE WHEN pipe_size >= 500 THEN 1 ELSE 0 END), 0) AS psize_500
		FROM %s lk, pwa_dma.dma_boundary dma
		WHERE lk.pwa_code = dma.pwa_code
			AND ST_Within(lk.wkb_geometry, dma.wkb_geometry)
			AND dma.pwa_code = $1
			AND dma.dma_id = $2
			AND (%d - 2500 - year_from_leakdate) <= 5`, tbl, yearBE)

	var result model.DMALeakpointsBySize
	err := r.pool.QueryRow(ctx, query, pwaCode, dmaID).Scan(
		&result.PSize100, &result.PSize200, &result.PSize300, &result.PSize400, &result.PSize500,
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
