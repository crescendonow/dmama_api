package model

// PWAOffice represents a PWA office location.
type PWAOffice struct {
	PwaCode  string  `json:"pwa_code"`
	Region   int     `json:"region"`
	Zone     *string `json:"zone,omitempty"`
	Name     string  `json:"name"`
	Geometry *string `json:"geometry,omitempty"`
}

// Waterworks represents a waterworks station marker.
type Waterworks struct {
	PwaCode  string  `json:"pwa_code"`
	Name     string  `json:"name"`
	Geometry *string `json:"geometry,omitempty"`
}
