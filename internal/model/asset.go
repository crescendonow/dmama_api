package model

// Pipe represents a pipe record.
type Pipe struct {
	PipeID      *string `json:"pipe_id,omitempty"`
	PwaCode     *string `json:"pwa_code,omitempty"`
	PipeSize    *int    `json:"pipe_size,omitempty"`
	PipeType    *string `json:"pipe_type,omitempty"`
	PipeFunc    *int    `json:"pipe_func,omitempty"`
	YearInstall *int    `json:"yearinstall,omitempty"`
	Long        *float64 `json:"long,omitempty"`
	Geometry    *string `json:"geometry,omitempty"`
}

// NearestPipe is the result of finding the nearest pipe to a point.
type NearestPipe struct {
	PipeID   string  `json:"pipe_id"`
	PwaCode  *string `json:"pwa_code,omitempty"`
	Distance float64 `json:"distance,omitempty"`
}

// Meter represents a meter/water meter record.
type Meter struct {
	OgcFid   int     `json:"ogc_fid"`
	CustCode *string `json:"custcode,omitempty"`
	PwaCode  *string `json:"pwa_code,omitempty"`
	Geometry *string `json:"geometry,omitempty"`
}

// Valve represents a valve record.
type Valve struct {
	OgcFid   int     `json:"ogc_fid"`
	PwaCode  *string `json:"pwa_code,omitempty"`
	Geometry *string `json:"geometry,omitempty"`
}

// Leakpoint represents a leak point record.
type Leakpoint struct {
	OgcFid   int     `json:"ogc_fid"`
	PwaCode  *string `json:"pwa_code,omitempty"`
	PipeSize *int    `json:"pipe_size,omitempty"`
	Geometry *string `json:"geometry,omitempty"`
}

// NearestAsset is the result of finding the nearest asset to a point.
type NearestAsset struct {
	Type     string  `json:"type"`
	OgcFid   int     `json:"ogc_fid"`
	PwaCode  *string `json:"pwa_code,omitempty"`
	Distance float64 `json:"distance"`
	Geometry *string `json:"geometry,omitempty"`
}

// AssetCount holds counts for assets in area queries.
type AssetCount struct {
	MeterTotal     int `json:"meter_total"`
	ValveTotal     int `json:"valve_total"`
	LeakpointTotal int `json:"leakpoint_total"`
	PipeTotal      int `json:"pipe_total"`
}

// PipeInfo holds pipe detail from leakpoint intersection.
type PipeInfo struct {
	PipeSize    *int    `json:"pipe_size,omitempty"`
	PipeType    *string `json:"pipe_type,omitempty"`
	YearInstall *int    `json:"yearinstall,omitempty"`
}
