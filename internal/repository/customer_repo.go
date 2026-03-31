package repository

import (
	"context"
	"fmt"

	"dmama_api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Allowed columns for usage SUM queries to prevent SQL injection.
var allowedUsageColumns = map[string]bool{
	"prswtusg": true, "use_water": true,
	"use_jan": true, "use_feb": true, "use_mar": true, "use_apr": true,
	"use_may": true, "use_jun": true, "use_jul": true, "use_aug": true,
	"use_sep": true, "use_oct": true, "use_nov": true, "use_dec": true,
}

type CustomerRepo struct {
	pool *pgxpool.Pool
}

func NewCustomerRepo(pool *pgxpool.Pool) *CustomerRepo {
	return &CustomerRepo{pool: pool}
}

// ValidateColumn checks if the column name is in the allowlist.
func ValidateColumn(col string) error {
	if !allowedUsageColumns[col] {
		return fmt.Errorf("invalid column: %s", col)
	}
	return nil
}

// SumUsageInDMA calculates usage sums by customer type within a DMA boundary. (Endpoint 4)
func (r *CustomerRepo) SumUsageInDMA(ctx context.Context, region int, pwaCode, dmaWkbGeometry, column string) (*model.DMAUsage, error) {
	if err := ValidateColumn(column); err != nil {
		return nil, err
	}

	tbl := TableName("giswebm_stamp", region, "bl_customer")
	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(%s), 0) AS c,
			COALESCE(SUM(CASE WHEN usetype IN ('11','12','13','14','15') THEN %s ELSE 0 END), 0) AS c_house,
			COALESCE(SUM(CASE WHEN usetype IN ('21','22','24','25','27') THEN %s ELSE 0 END), 0) AS c_government,
			COALESCE(SUM(CASE WHEN usetype IN ('23','26','28','29') THEN %s ELSE 0 END), 0) AS c_business_small,
			COALESCE(SUM(CASE WHEN usetype LIKE '3%%' THEN %s ELSE 0 END), 0) AS c_business_large
		FROM %s AS bl
		WHERE ST_Contains($1::geometry, bl.wkb_geometry) AND bl.pwa_code = $2`,
		column, column, column, column, column, tbl)

	var result model.DMAUsage
	err := r.pool.QueryRow(ctx, query, dmaWkbGeometry, pwaCode).Scan(
		&result.Total, &result.House, &result.Government, &result.BusinessSmall, &result.BusinessLarge,
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CountPopulationInDMA counts customers by type within a DMA boundary. (Endpoint 5)
func (r *CustomerRepo) CountPopulationInDMA(ctx context.Context, region int, pwaCode, dmaWkbGeometry, column string) (*model.DMAPopulation, error) {
	if err := ValidateColumn(column); err != nil {
		return nil, err
	}

	tbl := TableName("giswebm_stamp", region, "bl_customer")
	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(CASE WHEN %s > -1 THEN 1 ELSE 0 END), 0) AS c,
			COALESCE(SUM(CASE WHEN usetype LIKE '1%%' THEN 1 ELSE 0 END), 0) AS c_house,
			COALESCE(SUM(CASE WHEN usetype LIKE '2%%' THEN 1 ELSE 0 END), 0) AS c_government,
			COALESCE(SUM(CASE WHEN usetype LIKE '3%%' THEN 1 ELSE 0 END), 0) AS c_business
		FROM %s AS bl
		WHERE ST_Intersects($1::geometry, bl.wkb_geometry) AND bl.pwa_code = $2`,
		column, tbl)

	var result model.DMAPopulation
	err := r.pool.QueryRow(ctx, query, dmaWkbGeometry, pwaCode).Scan(
		&result.Total, &result.House, &result.Government, &result.Business,
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
