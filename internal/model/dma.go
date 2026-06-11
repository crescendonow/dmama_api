package model

// DMABoundary represents a DMA boundary record.
type DMABoundary struct {
	PwaCode     string  `json:"pwa_code"`
	DmaID       string  `json:"dma_id"`
	DmaNo       *string `json:"dma_no,omitempty"`
	Geometry    *string `json:"geometry,omitempty"`
	WkbGeometry *string `json:"-"` // raw geometry for spatial queries, not exposed in JSON
}

// DMAMapItem represents a DMA boundary for map display.
type DMAMapItem struct {
	ID       string  `json:"id"`
	PwaCode  string  `json:"pwa_code"`
	DmaID    string  `json:"dma_id"`
	Name     *string `json:"name,omitempty"`
	Geometry *string `json:"geometry,omitempty"`
}

// DMAUsage holds aggregated usage statistics within a DMA.
type DMAUsage struct {
	Total         float64 `json:"total"`
	House         float64 `json:"house"`
	Government    float64 `json:"government"`
	BusinessSmall float64 `json:"business_small"`
	BusinessLarge float64 `json:"business_large"`
}

// DMAPopulation holds aggregated population counts within a DMA.
type DMAPopulation struct {
	Total      int `json:"total"`
	House      int `json:"house"`
	Government int `json:"government"`
	Business   int `json:"business"`
}

// DMAPopulationStats holds population counts split by the stats endpoint categories.
type DMAPopulationStats struct {
	Total         int `json:"total"`
	House         int `json:"house"`
	Government    int `json:"government"`
	BusinessSmall int `json:"business_small"`
	BusinessLarge int `json:"business_large"`
}

// DMAStats holds merged usage and population statistics within a DMA.
type DMAStats struct {
	PwaCode    string             `json:"pwa_code"`
	DmaID      string             `json:"dma_id"`
	Column     string             `json:"column"`
	Usage      DMAUsage           `json:"usage"`
	Population DMAPopulationStats `json:"population"`
}

// DMADailyMeterCount holds the number of active meters within a DMA.
type DMADailyMeterCount struct {
	MeterCount int `json:"meter_count"`
}

// DMAPipeLength holds the total pipe length within a DMA.
type DMAPipeLength struct {
	TotalLength float64 `json:"total_length"`
}

// DMAUsageV2 holds usage stats from the v2 spatial join query.
type DMAUsageV2 struct {
	PwaCode  string  `json:"pwa_code"`
	DmaID    string  `json:"dma_id"`
	DmaNo    *string `json:"dma_no,omitempty"`
	Prswtusg float64 `json:"prswtusg"`
}

// DMALeakpointsBySize holds leakpoint counts grouped by pipe size.
type DMALeakpointsBySize struct {
	PSize100 int `json:"psize_100"`
	PSize200 int `json:"psize_200"`
	PSize300 int `json:"psize_300"`
	PSize400 int `json:"psize_400"`
	PSize500 int `json:"psize_500"`
}

// DMAPipeLengthClipped holds the clipped pipe length within a DMA in km.
type DMAPipeLengthClipped struct {
	SumLongKm float64 `json:"sum_long_km"`
}
