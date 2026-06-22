package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Shape types. These match both the Vallaris alias suffix (b<pwa_code>_<shape>) and the
// dmama_layer mirror table names in PostgreSQL 16.
const (
	ShapeDmaBoundary = "dma_boundary"
	ShapeFlowMeter   = "flow_meter"
	ShapeStepTest    = "step_test"
)

// AllowedShapes is the set of shapes the feature API accepts (note/10 shape_features).
var AllowedShapes = map[string]bool{
	ShapeDmaBoundary: true,
	ShapeFlowMeter:   true,
	ShapeStepTest:    true,
}

// GeometryTypeForShape maps a shape to its expected GeoJSON geometry type.
var GeometryTypeForShape = map[string]string{
	ShapeDmaBoundary: "Polygon",
	ShapeFlowMeter:   "Point",
	ShapeStepTest:    "Polygon",
}

// Feature is a Vallaris GeoJSON Feature stored in a per-branch features_<catalogId> collection.
// Geometry stays a GeoJSON geometry object so the collection's 2dsphere index applies.
type Feature struct {
	ID         primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	Type       string                 `json:"type" bson:"type"`
	Geometry   map[string]interface{} `json:"geometry" bson:"geometry"`
	Properties map[string]interface{} `json:"properties" bson:"properties"`
}

// FeatureRequest is the payload accepted from the frontend for create/update/validate.
// Geometry is the bare GeoJSON geometry object; Properties carries free-form metadata
// (camelCase, e.g. dmaNo/dmaName) that is stored verbatim alongside server-managed fields.
type FeatureRequest struct {
	Geometry   map[string]interface{} `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
	// DmaID, when > 0, requests a specific running number for a dma_boundary. Zero means
	// "let the backend assign max+1". A duplicate within the branch is rejected.
	DmaID int `json:"dma_id"`
}

// ValidationResult reports whether a geometry passed all enabled topology rules.
// Violations are blocking (create/update is rejected); Warnings are informational
// (e.g. a branch that has no dma_boundary yet, so the within-coverage rule is skipped).
type ValidationResult struct {
	Valid      bool     `json:"valid"`
	Violations []string `json:"violations,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
}

// AsInt coerces a BSON/JSON numeric value (int32/int64/float64/…) to int, returning 0 for
// nil or non-numeric input. Used to read properties.dmaId regardless of how it was stored.
func AsInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float32:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}
