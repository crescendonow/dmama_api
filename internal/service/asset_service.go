package service

import (
	"context"
	"fmt"

	"dmama_api/internal/model"
	"dmama_api/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AssetService struct {
	pool *pgxpool.Pool
}

func NewAssetService(pool *pgxpool.Pool) *AssetService {
	return &AssetService{pool: pool}
}

// FindNearest finds the nearest asset (pipe/meter/valve/firehydrant/leakpoint) within radius. (Endpoint 10)
func (s *AssetService) FindNearest(ctx context.Context, region int, lng, lat, radius float64, assetType string, format model.GeomFormat) (map[string]interface{}, error) {
	tbl := repository.TableName("oracle", region, assetType)
	geomExpr := model.SQLGeomExpr(fmt.Sprintf("%s.wkb_geometry", assetType), format)
	idColumn, idKey, err := nearestAssetIdentifier(assetType)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT %s.%s, %s.pwa_code, %s,
			ST_Distance(ST_Point($1, $2)::geography, %s.wkb_geometry::geography) AS distance
		FROM %s AS %s
		JOIN pwa_office.pwa_office234 AS office ON office.pwa_code = %s.pwa_code
		WHERE ST_DWithin(ST_Point($1, $2)::geography, %s.wkb_geometry::geography, $3)
		ORDER BY distance ASC
		LIMIT 1`,
		assetType, idColumn, assetType, geomExpr,
		assetType, tbl, assetType, assetType, assetType)

	rows, err := s.pool.Query(ctx, query, lng, lat, radius)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	values, err := rows.Values()
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"type":     assetType,
		idKey:      values[0],
		"pwa_code": values[1],
		"geometry": values[2],
		"distance": values[3],
	}
	return result, rows.Err()
}

func nearestAssetIdentifier(assetType string) (column string, jsonKey string, err error) {
	switch assetType {
	case "pipe":
		return "pipe_id", "pipe_id", nil
	case "meter":
		return "ogc_fid", "ogc_fid", nil
	case "valve":
		return "valve_id", "valve_id", nil
	case "firehydrant":
		return "fire_id", "fire_id", nil
	case "leakpoint":
		return "leak_no", "leak_no", nil
	default:
		return "", "", fmt.Errorf("invalid asset type: %s", assetType)
	}
}

// CountAssetsInArea counts assets within a drawn polygon or buffered linestring. (Endpoint 13)
func (s *AssetService) CountAssetsInArea(ctx context.Context, regions []int, geometryGeoJSON string, buffer float64, geomType string) (*model.AssetCount, error) {
	var result model.AssetCount

	for _, assetType := range []string{"meter", "valve", "leakpoint", "pipe"} {
		query := repository.UnionAll(regions, "oracle", assetType, func(tbl string, region int) string {
			if geomType == "LineString" {
				return fmt.Sprintf(`
					WITH srid_info AS (
						SELECT COALESCE(NULLIF(ST_SRID(wkb_geometry), 0), 4326) AS srid
						FROM %s
						WHERE wkb_geometry IS NOT NULL
						LIMIT 1
					),
					search AS (
						SELECT CASE
							WHEN srid_info.srid = 4326 THEN ST_SetSRID(ST_GeomFromGeoJSON($1), 4326)
							ELSE ST_Transform(ST_SetSRID(ST_GeomFromGeoJSON($1), 4326), srid_info.srid)
						END AS geom
						FROM srid_info
					)
					SELECT COUNT(*) AS total
					FROM %s AS %s
					CROSS JOIN search
					WHERE ST_DWithin(
						search.geom::geography,
						CASE
							WHEN ST_SRID(%s.wkb_geometry) = 0 THEN ST_SetSRID(%s.wkb_geometry, ST_SRID(search.geom))
							ELSE %s.wkb_geometry
						END::geography,
						$2
					)`,
					tbl, tbl, assetType, assetType, assetType, assetType)
			}

			return fmt.Sprintf(`
				WITH srid_info AS (
					SELECT COALESCE(NULLIF(ST_SRID(wkb_geometry), 0), 4326) AS srid
					FROM %s
					WHERE wkb_geometry IS NOT NULL
					LIMIT 1
				),
				search AS (
					SELECT CASE
						WHEN srid_info.srid = 4326 THEN ST_SetSRID(ST_GeomFromGeoJSON($1), 4326)
						ELSE ST_Transform(ST_SetSRID(ST_GeomFromGeoJSON($1), 4326), srid_info.srid)
					END AS geom
					FROM srid_info
				)
				SELECT COUNT(*) AS total
				FROM %s AS %s
				CROSS JOIN search
				WHERE %s.wkb_geometry && search.geom
					AND ST_Intersects(
						search.geom,
						CASE
							WHEN ST_SRID(%s.wkb_geometry) = 0 THEN ST_SetSRID(%s.wkb_geometry, ST_SRID(search.geom))
							ELSE %s.wkb_geometry
						END
					)`,
				tbl, tbl, assetType, assetType, assetType, assetType, assetType)
		})
		query = fmt.Sprintf("SELECT COALESCE(SUM(total), 0) FROM (%s) sub", query)

		var count int
		var err error
		if geomType == "LineString" {
			err = s.pool.QueryRow(ctx, query, geometryGeoJSON, buffer).Scan(&count)
		} else {
			err = s.pool.QueryRow(ctx, query, geometryGeoJSON).Scan(&count)
		}
		if err != nil {
			return nil, fmt.Errorf("counting %s: %w", assetType, err)
		}

		switch assetType {
		case "meter":
			result.MeterTotal = count
		case "valve":
			result.ValveTotal = count
		case "leakpoint":
			result.LeakpointTotal = count
		case "pipe":
			result.PipeTotal = count
		}
	}
	return &result, nil
}
