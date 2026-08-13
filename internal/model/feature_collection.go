package model

import "encoding/json"

// FeatureCollectionRequest is the GeoJSON body accepted by collection create requests.
type FeatureCollectionRequest struct {
	Type     string            `json:"type"`
	Features []json.RawMessage `json:"features"`
}

// FeatureCollectionMember carries the optional client id. step_test maps it to stepName;
// all shapes still generate their stored MongoDB ID on create.
type FeatureCollectionMember struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	FeatureRequest
}

// FeatureCollection is the GeoJSON collection returned after a successful bulk create.
type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}
