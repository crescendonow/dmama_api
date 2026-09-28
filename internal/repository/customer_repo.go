package repository

import (
	"context"
	"fmt"
	"strings"

	"dmama_api/internal/model"

	"github.com/jackc/pgx/v5"
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

// ValidateStatsRegionColumn restricts the stats-region endpoint to its public contract.
func ValidateStatsRegionColumn(col string) error {
	switch col {
	case "prswtusg",
		"lstwtusg1", "lstwtusg2", "lstwtusg3", "lstwtusg4",
		"lstwtusg5", "lstwtusg6", "lstwtusg7", "lstwtusg8",
		"lstwtusg9", "lstwtusg10", "lstwtusg11", "lstwtusg12":
		return nil
	default:
		return fmt.Errorf("invalid stats-region column: %s", col)
	}
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

func dmaStatsRegionQuery(region int, column, pwaCode string) (string, string, error) {
	prefix, err := ZonePrefix(region)
	if err != nil {
		return "", "", err
	}

	tbl := TableName("giswebm_stamp", region, "bl_customer")
	filter := "dma.pwa_code LIKE $1"
	value := prefix + "%"
	if pwaCode != "" {
		pwaRegion, err := RegionFromPWACode(pwaCode)
		if err != nil {
			return "", "", err
		}
		if pwaRegion != region {
			return "", "", fmt.Errorf("pwa_code %s does not belong to region %d", pwaCode, region)
		}
		filter = "dma.pwa_code = $1"
		value = pwaCode
	}
	query := fmt.Sprintf(`
		SELECT
			dma.pwa_code,
			dma.dma_id,
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
		WHERE %s
		GROUP BY dma.pwa_code, dma.dma_id
		ORDER BY dma.pwa_code, dma.dma_id`,
		column, column, column, column, column, column, column, column, column, column, tbl, filter)

	return query, value, nil
}

// GetStatsRegion returns merged usage and population statistics for every DMA in a region.
func (r *CustomerRepo) GetStatsRegion(ctx context.Context, region int, column, pwaCode string) ([]model.DMAStats, error) {
	if err := ValidateStatsRegionColumn(column); err != nil {
		return nil, err
	}

	query, filterValue, err := dmaStatsRegionQuery(region, column, pwaCode)
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, query, filterValue)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]model.DMAStats, 0)
	for rows.Next() {
		var item model.DMAStats
		if err := rows.Scan(
			&item.PwaCode,
			&item.DmaID,
			&item.Usage.Total, &item.Usage.House, &item.Usage.Government, &item.Usage.BusinessSmall, &item.Usage.BusinessLarge,
			&item.Population.Total, &item.Population.House, &item.Population.Government, &item.Population.BusinessSmall, &item.Population.BusinessLarge,
		); err != nil {
			return nil, err
		}
		item.Column = column
		stats = append(stats, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return stats, nil
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

// customersAllQuery builds the SELECT for one region table of /api/dma/customers-all
// (see note/22_plan_for_customers_endpoint.md). It returns positional arguments alongside the
// query text; the only interpolated identifier is the table name from TableName with a region
// that has already been validated by the caller. Every filter value is bound, never interpolated.
//
// Customers keep exactly one row even when several DMA boundaries overlap them: the LATERAL
// join orders by dma_id and takes the smallest match. Customers outside every matching DMA get
// dma_id/dma_name = NULL because the join is LEFT.
func customersAllQuery(region int, f model.CustomersAllFilter) (string, []any) {
	tbl := TableName("giswebm_stamp", region, "bl_customer")

	var args []any
	bind := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	dmaIDFilter := ""
	if len(f.DmaIDs) > 0 {
		ids := make([]int32, len(f.DmaIDs))
		for i, id := range f.DmaIDs {
			ids[i] = int32(id)
		}
		dmaIDFilter = fmt.Sprintf("\n      AND b.dma_id = ANY(%s::int[])", bind(ids))
	}

	var whereClauses []string
	if f.PwaCode != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("c.pwa_code = %s", bind(f.PwaCode)))
	}
	if len(f.Usetypes) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("c.usetype = ANY(%s::text[])", bind(f.Usetypes)))
	}
	if f.PolygonGeoJSON != "" {
		placeholder := bind(f.PolygonGeoJSON)
		whereClauses = append(whereClauses, fmt.Sprintf(
			"c.wkb_geometry && ST_SetSRID(ST_GeomFromGeoJSON(%s), 4326)\n  AND ST_Intersects(ST_SetSRID(ST_GeomFromGeoJSON(%s), 4326), c.wkb_geometry)",
			placeholder, placeholder))
	}
	// dma_id only narrows which DMA is *reported*, not which customers are returned, once a
	// polygon is present -- customers outside the selected DMAs but inside the polygon still come
	// back, with dma_id/dma_name null (plan decision: "dma_id + my_polygon").
	if len(f.DmaIDs) > 0 && f.PolygonGeoJSON == "" {
		whereClauses = append(whereClauses, "d.dma_id IS NOT NULL")
	}

	where := "WHERE TRUE"
	for _, clause := range whereClauses {
		where += "\n  AND " + clause
	}

	lstColumns := make([]string, 12)
	for i := 1; i <= 12; i++ {
		lstColumns[i-1] = fmt.Sprintf("c.lstwtusg%d::float8", i)
	}

	query := fmt.Sprintf(`SELECT
    d.dma_id::text, d.dma_name, c.pwa_code,
    c.is_customer::text, c.custstat::text, c.meterstat::text, c.usetype::text, c.custname,
    ST_Y(c.wkb_geometry), ST_X(c.wkb_geometry),
    c.custaddr, c.custcode, c.meterno, c.mtrrdroute::text, c.mtrseq::text,
    c.metermake, c.metersize,
    c.prswtusg::float8, %s
FROM %s c
LEFT JOIN LATERAL (
    SELECT b.dma_id, b.dma_name
    FROM pwa_dma.dma_boundary b
    WHERE b.pwa_code = c.pwa_code
      AND b.wkb_geometry && c.wkb_geometry
      AND ST_Intersects(b.wkb_geometry, c.wkb_geometry)%s
    ORDER BY b.dma_id
    LIMIT 1
) d ON TRUE
%s`, strings.Join(lstColumns, ", "), tbl, dmaIDFilter, where)

	return query, args
}

// ValidatePolygon checks that a GeoJSON geometry parses and is a valid PostGIS geometry before
// any streaming starts, so a bad my_polygon becomes a normal 400 JSON response.
func (r *CustomerRepo) ValidatePolygon(ctx context.Context, geojson string) error {
	var isValid bool
	var reason string
	err := r.pool.QueryRow(ctx,
		`SELECT ST_IsValid(g), ST_IsValidReason(g) FROM (SELECT ST_SetSRID(ST_GeomFromGeoJSON($1), 4326) g) s`,
		geojson,
	).Scan(&isValid, &reason)
	if err != nil {
		return fmt.Errorf("invalid my_polygon: %v", err)
	}
	if !isValid {
		return fmt.Errorf("invalid my_polygon: %s", reason)
	}
	return nil
}

// CustomerAllRows lazily scans one region's rows for /api/dma/customers-all. Callers must Close it.
type CustomerAllRows struct {
	rows pgx.Rows
}

// OpenCustomersAll opens (without reading) the customer rows for one region. Opening the first
// region's rows before starting the HTTP response stream is what lets a DB error surface as a
// normal 500 JSON response instead of a broken stream (see the handler).
func (r *CustomerRepo) OpenCustomersAll(ctx context.Context, region int, filter model.CustomersAllFilter) (*CustomerAllRows, error) {
	query, args := customersAllQuery(region, filter)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &CustomerAllRows{rows: rows}, nil
}

// Next advances to the next row.
func (r *CustomerAllRows) Next() bool { return r.rows.Next() }

// Err returns any error encountered during iteration.
func (r *CustomerAllRows) Err() error { return r.rows.Err() }

// Close releases the underlying database rows.
func (r *CustomerAllRows) Close() { r.rows.Close() }

// Customer scans the current row into a model.DMACustomerAll.
func (r *CustomerAllRows) Customer() (model.DMACustomerAll, error) {
	var customer model.DMACustomerAll
	var dmaID, dmaName, isCustomer, custstat, meterstat, usetype, custname pgtype.Text
	var custaddr, custcode, meterno, mtrrdroute, mtrseq, metermake, metersize pgtype.Text
	var latitude, longitude, prswtusg pgtype.Float8
	var lst [12]pgtype.Float8

	dest := []any{
		&dmaID, &dmaName, &customer.PwaCode,
		&isCustomer, &custstat, &meterstat, &usetype, &custname,
		&latitude, &longitude,
		&custaddr, &custcode, &meterno, &mtrrdroute, &mtrseq,
		&metermake, &metersize,
		&prswtusg,
	}
	for i := range lst {
		dest = append(dest, &lst[i])
	}

	if err := r.rows.Scan(dest...); err != nil {
		return model.DMACustomerAll{}, err
	}

	customer.DmaID = textPtr(dmaID)
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
	customer.Lstwtusg1 = float8Ptr(lst[0])
	customer.Lstwtusg2 = float8Ptr(lst[1])
	customer.Lstwtusg3 = float8Ptr(lst[2])
	customer.Lstwtusg4 = float8Ptr(lst[3])
	customer.Lstwtusg5 = float8Ptr(lst[4])
	customer.Lstwtusg6 = float8Ptr(lst[5])
	customer.Lstwtusg7 = float8Ptr(lst[6])
	customer.Lstwtusg8 = float8Ptr(lst[7])
	customer.Lstwtusg9 = float8Ptr(lst[8])
	customer.Lstwtusg10 = float8Ptr(lst[9])
	customer.Lstwtusg11 = float8Ptr(lst[10])
	customer.Lstwtusg12 = float8Ptr(lst[11])

	return customer, nil
}
