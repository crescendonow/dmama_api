package repository

import (
	"context"
	"fmt"

	"dmama_api/internal/model"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Allowed columns for usage SUM queries to prevent SQL injection.
var allowedUsageColumns = map[string]bool{
	"prswtusg": true, "use_water": true,
	"use_jan": true, "use_feb": true, "use_mar": true, "use_apr": true,
	"use_may": true, "use_jun": true, "use_jul": true, "use_aug": true,
	"use_sep": true, "use_oct": true, "use_nov": true, "use_dec": true,
	"lstwtusg1": true, "lstwtusg2": true, "lstwtusg3": true, "lstwtusg4": true,
	"lstwtusg5": true, "lstwtusg6": true, "lstwtusg7": true, "lstwtusg8": true,
	"lstwtusg9": true, "lstwtusg10": true, "lstwtusg11": true, "lstwtusg12": true,
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

func dmaCustomersQuery(region int) string {
	tbl := TableName("giswebm_stamp", region, "bl_customer")
	return fmt.Sprintf(`
		SELECT
			d.dma_id,
			d.dma_name,
			d.pwa_code,
			c.is_customer::text,
			c.custstat::text,
			c.meterstat::text,
			c.usetype::text,
			c.custname,
			ST_Y(c.wkb_geometry) AS latitude,
			ST_X(c.wkb_geometry) AS longitude,
			c.custaddr,
			c.custcode,
			c.meterno,
			c.mtrrdroute,
			c.mtrseq,
			c.metermake,
			c.metersize,
			c.prswtusg
		FROM (
			SELECT *
			FROM pwa_dma.dma_boundary
			WHERE dma_id = $1
				AND pwa_code = $2
		) d
		JOIN %s c
			ON c.pwa_code = d.pwa_code
			AND d.wkb_geometry && c.wkb_geometry
			AND ST_Intersects(d.wkb_geometry, c.wkb_geometry)
		WHERE c.usetype IN ('22','35')`, tbl)
}

// GetCustomersInDMA returns customer rows inside a DMA boundary for selected use types.
func (r *CustomerRepo) GetCustomersInDMA(ctx context.Context, region int, pwaCode, dmaID string) ([]model.DMACustomer, error) {
	rows, err := r.pool.Query(ctx, dmaCustomersQuery(region), dmaID, pwaCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	customers := make([]model.DMACustomer, 0)
	for rows.Next() {
		var customer model.DMACustomer
		var dmaName, isCustomer, custstat, meterstat, usetype, custname pgtype.Text
		var custaddr, custcode, meterno, mtrrdroute, mtrseq, metermake, metersize pgtype.Text
		var latitude, longitude, prswtusg pgtype.Float8

		if err := rows.Scan(
			&customer.DmaID,
			&dmaName,
			&customer.PwaCode,
			&isCustomer,
			&custstat,
			&meterstat,
			&usetype,
			&custname,
			&latitude,
			&longitude,
			&custaddr,
			&custcode,
			&meterno,
			&mtrrdroute,
			&mtrseq,
			&metermake,
			&metersize,
			&prswtusg,
		); err != nil {
			return nil, err
		}

		customer.DmaName = textPtr(dmaName)
		customer.IsCustomer = textPtr(isCustomer)
		customer.Custstat = textPtr(custstat)
		customer.Meterstat = textPtr(meterstat)
		customer.Usetype = textPtr(usetype)
		customer.Custname = textPtr(custname)
		customer.Latitude = float8Ptr(latitude)
		customer.Longitude = float8Ptr(longitude)
		customer.Custaddr = textPtr(custaddr)
		customer.Custcode = textPtr(custcode)
		customer.Meterno = textPtr(meterno)
		customer.Mtrrdroute = textPtr(mtrrdroute)
		customer.Mtrseq = textPtr(mtrseq)
		customer.Metermake = textPtr(metermake)
		customer.Metersize = textPtr(metersize)
		customer.Prswtusg = float8Ptr(prswtusg)
		customers = append(customers, customer)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return customers, nil
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	decoded := decodeDBText(value.String)
	return &decoded
}

func float8Ptr(value pgtype.Float8) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}

// SumUsageInDMA calculates usage sums by customer type within a DMA boundary. (Endpoint 4)
func (r *CustomerRepo) SumUsageInDMA(ctx context.Context, region int, pwaCode, dmaWkbGeometry, column string) (*model.DMAUsage, error) {
	if err := ValidateColumn(column); err != nil {
		return nil, err
	}

	tbl := TableName("giswebm_stamp", region, "bl_customer")
	query := fmt.Sprintf(`
		WITH dma AS (
			SELECT ST_GeomFromEWKT($1) AS geom
		)
		SELECT
			COALESCE(SUM(%s), 0) AS c,
			COALESCE(SUM(CASE WHEN usetype IN ('11','12','13','14','15') THEN %s ELSE 0 END), 0) AS c_house,
			COALESCE(SUM(CASE WHEN usetype IN ('21','22','24','25','27') THEN %s ELSE 0 END), 0) AS c_government,
			COALESCE(SUM(CASE WHEN usetype IN ('23','26','28','29') THEN %s ELSE 0 END), 0) AS c_business_small,
			COALESCE(SUM(CASE WHEN usetype LIKE '3%%' THEN %s ELSE 0 END), 0) AS c_business_large
		FROM %s AS bl
		CROSS JOIN dma
		WHERE ST_Contains(
			dma.geom,
			CASE
				WHEN ST_SRID(bl.wkb_geometry) = ST_SRID(dma.geom) THEN bl.wkb_geometry
				WHEN ST_SRID(bl.wkb_geometry) = 0 THEN ST_SetSRID(bl.wkb_geometry, ST_SRID(dma.geom))
				ELSE ST_Transform(bl.wkb_geometry, ST_SRID(dma.geom))
			END
		) AND bl.pwa_code = $2`,
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
		WITH dma AS (
			SELECT ST_GeomFromEWKT($1) AS geom
		)
		SELECT
			COALESCE(SUM(CASE WHEN %s > -1 THEN 1 ELSE 0 END), 0) AS c,
			COALESCE(SUM(CASE WHEN usetype LIKE '1%%' THEN 1 ELSE 0 END), 0) AS c_house,
			COALESCE(SUM(CASE WHEN usetype LIKE '2%%' THEN 1 ELSE 0 END), 0) AS c_government,
			COALESCE(SUM(CASE WHEN usetype LIKE '3%%' THEN 1 ELSE 0 END), 0) AS c_business
		FROM %s AS bl
		CROSS JOIN dma
		WHERE ST_Intersects(
			dma.geom,
			CASE
				WHEN ST_SRID(bl.wkb_geometry) = ST_SRID(dma.geom) THEN bl.wkb_geometry
				WHEN ST_SRID(bl.wkb_geometry) = 0 THEN ST_SetSRID(bl.wkb_geometry, ST_SRID(dma.geom))
				ELSE ST_Transform(bl.wkb_geometry, ST_SRID(dma.geom))
			END
		) AND bl.pwa_code = $2`,
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

// GetStatsInDMA returns merged usage sums and population counts within a DMA via direct DB-side join. (Endpoint stats)
func (r *CustomerRepo) GetStatsInDMA(ctx context.Context, region int, pwaCode, dmaID, column string) (*model.DMAStats, error) {
	if err := ValidateColumn(column); err != nil {
		return nil, err
	}

	tbl := TableName("giswebm_stamp", region, "bl_customer")
	query := fmt.Sprintf(`
		SELECT
			COUNT(dma.dma_id),
			COALESCE(SUM(bl.%s), 0),
			COALESCE(SUM(CASE WHEN bl.usetype IN ('11','12','13','14','15') THEN bl.%s ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN bl.usetype IN ('21','22','24','25','27') THEN bl.%s ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN bl.usetype IN ('23','26','28','29') THEN bl.%s ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN bl.usetype LIKE '3%%' THEN bl.%s ELSE 0 END), 0),
			COALESCE(COUNT(bl.pwa_code) FILTER (WHERE bl.%s > -1), 0),
			COALESCE(COUNT(bl.pwa_code) FILTER (WHERE bl.usetype LIKE '1%%' AND bl.%s > -1), 0),
			COALESCE(COUNT(bl.pwa_code) FILTER (WHERE bl.usetype IN ('21','22','24','25','27') AND bl.%s > -1), 0),
			COALESCE(COUNT(bl.pwa_code) FILTER (WHERE bl.usetype IN ('23','26','28','29') AND bl.%s > -1), 0),
			COALESCE(COUNT(bl.pwa_code) FILTER (WHERE bl.usetype LIKE '3%%' AND bl.%s > -1), 0)
		FROM pwa_dma.dma_boundary AS dma
		LEFT JOIN %s AS bl
			ON bl.pwa_code = dma.pwa_code
			AND ST_Intersects(
				dma.wkb_geometry,
				CASE
					WHEN ST_SRID(bl.wkb_geometry) = ST_SRID(dma.wkb_geometry) THEN bl.wkb_geometry
					WHEN ST_SRID(bl.wkb_geometry) = 0 THEN ST_SetSRID(bl.wkb_geometry, ST_SRID(dma.wkb_geometry))
					ELSE ST_Transform(bl.wkb_geometry, ST_SRID(dma.wkb_geometry))
				END
			)
		WHERE dma.pwa_code = $1
			AND dma.dma_id = $2`,
		column, column, column, column, column, column, column, column, column, column, tbl)

	var result model.DMAStats
	var dmaCount int
	err := r.pool.QueryRow(ctx, query, pwaCode, dmaID).Scan(
		&dmaCount,
		&result.Usage.Total, &result.Usage.House, &result.Usage.Government, &result.Usage.BusinessSmall, &result.Usage.BusinessLarge,
		&result.Population.Total, &result.Population.House, &result.Population.Government, &result.Population.BusinessSmall, &result.Population.BusinessLarge,
	)
	if err != nil {
		return nil, err
	}
	if dmaCount == 0 {
		return nil, nil
	}
	result.PwaCode = pwaCode
	result.DmaID = dmaID
	result.Column = column
	return &result, nil
}

// CountPopulationByDMA counts customers by type within a DMA via direct DB-side join.
// This avoids round-tripping the DMA geometry through the application layer.
func (r *CustomerRepo) CountPopulationByDMA(ctx context.Context, region int, pwaCode, dmaID, column string) (*model.DMAPopulation, error) {
	if err := ValidateColumn(column); err != nil {
		return nil, err
	}

	tbl := TableName("giswebm_stamp", region, "bl_customer")
	query := fmt.Sprintf(`
		SELECT
			COALESCE(COUNT(*) FILTER (WHERE %s > -1), 0) AS c,
			COALESCE(COUNT(*) FILTER (WHERE usetype LIKE '1%%'), 0) AS c_house,
			COALESCE(COUNT(*) FILTER (WHERE usetype LIKE '2%%'), 0) AS c_government,
			COALESCE(COUNT(*) FILTER (WHERE usetype LIKE '3%%'), 0) AS c_business
		FROM pwa_dma.dma_boundary AS dma
		JOIN %s AS bl
			ON bl.pwa_code = dma.pwa_code
		WHERE dma.pwa_code = $1
			AND dma.dma_id = $2
			AND ST_Intersects(
				dma.wkb_geometry,
				CASE
					WHEN ST_SRID(bl.wkb_geometry) = ST_SRID(dma.wkb_geometry) THEN bl.wkb_geometry
					WHEN ST_SRID(bl.wkb_geometry) = 0 THEN ST_SetSRID(bl.wkb_geometry, ST_SRID(dma.wkb_geometry))
					ELSE ST_Transform(bl.wkb_geometry, ST_SRID(dma.wkb_geometry))
				END
			)`,
		column, tbl)

	var result model.DMAPopulation
	err := r.pool.QueryRow(ctx, query, pwaCode, dmaID).Scan(
		&result.Total, &result.House, &result.Government, &result.Business,
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
