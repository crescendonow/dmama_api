package model

import "fmt"

type GeomFormat string

const (
	FormatGeoJSON GeomFormat = "geojson"
	FormatWKT     GeomFormat = "wkt"
)

func ParseGeomFormat(raw string) GeomFormat {
	if raw == "wkt" {
		return FormatWKT
	}
	return FormatGeoJSON
}

func SQLGeomExpr(column string, format GeomFormat) string {
	if format == FormatWKT {
		return fmt.Sprintf("ST_AsText(%s) AS geometry", column)
	}
	return fmt.Sprintf("ST_AsGeoJSON(%s) AS geometry", column)
}
