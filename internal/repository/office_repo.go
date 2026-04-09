package repository

import (
	"context"
	"fmt"

	"dmama_api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OfficeRepo struct {
	pool *pgxpool.Pool
}

func NewOfficeRepo(pool *pgxpool.Pool) *OfficeRepo {
	return &OfficeRepo{pool: pool}
}

// GetAll fetches all PWA office locations. (Endpoint 8)
func (r *OfficeRepo) GetAll(ctx context.Context, format model.GeomFormat) ([]model.PWAOffice, error) {
	geomExpr := model.SQLGeomExpr("wkb_geometry", format)
	query := fmt.Sprintf(`
		SELECT pwa_code, region, zone, name, %s
		FROM pwa_office.pwa_office234`, geomExpr)

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var offices []model.PWAOffice
	for rows.Next() {
		var o model.PWAOffice
		if err := rows.Scan(&o.PwaCode, &o.Region, &o.Zone, &o.Name, &o.Geometry); err != nil {
			return nil, err
		}
		o.Region = decodeDBText(o.Region)
		o.Zone = decodeDBTextPtr(o.Zone)
		o.Name = decodeDBText(o.Name)
		offices = append(offices, o)
	}
	return offices, nil
}
