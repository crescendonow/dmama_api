package service

import (
	"context"
	"fmt"

	"dmama_api/internal/model"
	"dmama_api/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AssetService struct {
	pool *pgxpool.Pool
}

func NewAssetService(pool *pgxpool.Pool) *AssetService {
	return &AssetService{pool: pool}
}

// FindNearest finds the nearest asset (pipe/meter/valve/leakpoint) within radius. (Endpoint 10)
func (s *AssetService) FindNearest(ctx context.Context, region int, lng, lat, radius float64, assetType string, format model.GeomFormat) (*model.NearestAsset, error) {
	tbl := repository.TableName("oracle", region, assetType)
	geomExpr := model.SQLGeomExpr(fmt.Sprintf("%s.wkb_geometry", assetType), format)

	query := fmt.Sprintf(`
		SELECT %s.ogc_fid, %s.pwa_code, %s,
			ST_Distance(ST_Point($1, $2)::geography, %s.wkb_geometry::geography) AS distance
		FROM %s AS %s
		JOIN pwa_office.pwa_office234 AS office ON office.pwa_code = %s.pwa_code
		WHERE ST_DWithin(ST_Point($1, $2)::geography, %s.wkb_geometry::geography, $3)
		ORDER BY distance ASC
		LIMIT 1`,
		assetType, assetType, geomExpr,
		assetType, tbl, assetType, assetType, assetType)

	var result model.NearestAsset
	result.Type = assetType
	err := s.pool.QueryRow(ctx, query, lng, lat, radius).Scan(
		&result.OgcFid, &result.PwaCode, &result.Geometry, &result.Distance,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

// CountAssetsInArea counts assets within a drawn polygon or buffered linestring. (Endpoint 13)
func (s *AssetService) CountAssetsInArea(ctx context.Context, regions []int, geometryGeoJSON string, buffer float64, geomType string) (*model.AssetCount, error) {
	var result model.AssetCount

	for _, assetType := range []string{"meter", "valve", "leakpoint", "pipe"} {
		var spatialClause string
		if geomType == "LineString" {
			spatialClause = fmt.Sprintf(
				"ST_DWithin(ST_GeomFromGeoJSON($1)::geography, %s.wkb_geometry::geography, $2)",
				assetType)
		} else {
			spatialClause = fmt.Sprintf(
				"ST_Intersects(ST_GeomFromGeoJSON($1)::geometry, %s.wkb_geometry)",
				assetType)
		}

		query := repository.UnionAll(regions, "oracle", assetType, func(tbl string, region int) string {
			return fmt.Sprintf(
				`SELECT COUNT(*) AS total FROM %s AS %s WHERE %s`,
				tbl, assetType, spatialClause)
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
