package model

import "encoding/json"

// FeatureCollectionRequest is the GeoJSON body accepted by step_test create and validate.
type FeatureCollectionRequest struct {
	Type     string            `json:"type"`
	Features []json.RawMessage `json:"features"`
}

// FeatureCollectionMember omits client id because Create generates the stored ID.
type FeatureCollectionMember struct {
	Type string `json:"type"`
	FeatureRequest
}

// FeatureCollection is the GeoJSON collection returned after a successful step_test bulk create.
type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}
