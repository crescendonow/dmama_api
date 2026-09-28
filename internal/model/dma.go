package model

// DMABoundary represents a DMA boundary record.
type DMABoundary struct {
	PwaCode     string  `json:"pwa_code"`
	DmaID       string  `json:"dma_id"`
	DmaNo       *string `json:"dma_no,omitempty"`
	DmaName     *string `json:"dma_name"`
	Geometry    *string `json:"geometry,omitempty"`
	WkbGeometry *string `json:"-"` // raw geometry for spatial queries, not exposed in JSON
}

// DMAMapItem represents a DMA boundary for map display.
type DMAMapItem struct {
	ID       string  `json:"id"`
	PwaCode  string  `json:"pwa_code"`
	DmaID    string  `json:"dma_id"`
	DmaNo    *string `json:"dma_no,omitempty"`
	DmaName  *string `json:"dma_name"`
	Geometry *string `json:"geometry,omitempty"`
}

// DMACustomer represents a customer point inside a DMA boundary.
type DMACustomer struct {
	DmaID      string   `json:"dma_id"`
	DmaName    *string  `json:"dma_name"`
	PwaCode    string   `json:"pwa_code"`
	IsCustomer *string  `json:"is_customer"`
	Custstat   *string  `json:"custstat"`
	Meterstat  *string  `json:"meterstat"`
	Usetype    *string  `json:"usetype"`
	Custname   *string  `json:"custname"`
	Latitude   *float64 `json:"latitude"`
	Longitude  *float64 `json:"longitude"`
	Custaddr   *string  `json:"custaddr"`
	Custcode   *string  `json:"custcode"`
	Meterno    *string  `json:"meterno"`
	Mtrrdroute *string  `json:"mtrrdroute"`
	Mtrseq     *string  `json:"mtrseq"`
	Metermake  *string  `json:"metermake"`
	Metersize  *string  `json:"metersize"`
	Prswtusg   *float64 `json:"prswtusg"`
}

// CustomersAllFilter holds parsed, validated request filters for the streamed
// /api/dma/customers-all endpoint (see note/22_plan_for_customers_endpoint.md).
// Regions is always non-empty: the resolved single region, or every region (1..10) when
// neither region nor pwa_code was supplied.
type CustomersAllFilter struct {
	Regions        []int
	PwaCode        string
	DmaIDs         []int
	Usetypes       []string
	PolygonGeoJSON string // raw GeoJSON geometry object (my_polygon), empty when not provided
}

// DMACustomerAll represents one customer row streamed by /api/dma/customers-all.
// Field order matches the public JSON contract exactly (json.Marshal follows struct field order).
// DmaID/DmaName are nil for customers outside every matching DMA (LEFT JOIN).
type DMACustomerAll struct {
	DmaID      *string  `json:"dma_id"`
	DmaName    *string  `json:"dma_name"`
	PwaCode    string   `json:"pwa_code"`
	IsCustomer *string  `json:"is_customer"`
	Custstat   *string  `json:"custstat"`
	Meterstat  *string  `json:"meterstat"`
	Usetype    *string  `json:"usetype"`
	Custname   *string  `json:"custname"`
	Latitude   *float64 `json:"latitude"`
	Longitude  *float64 `json:"longitude"`
	Custaddr   *string  `json:"custaddr"`
	Custcode   *string  `json:"custcode"`
	Meterno    *string  `json:"meterno"`
	Mtrrdroute *string  `json:"mtrrdroute"`
	Mtrseq     *string  `json:"mtrseq"`
	Metermake  *string  `json:"metermake"`
	Metersize  *string  `json:"metersize"`
	Prswtusg   *float64 `json:"prswtusg"`
	Lstwtusg1  *float64 `json:"lstwtusg1"`
	Lstwtusg2  *float64 `json:"lstwtusg2"`
	Lstwtusg3  *float64 `json:"lstwtusg3"`
	Lstwtusg4  *float64 `json:"lstwtusg4"`
	Lstwtusg5  *float64 `json:"lstwtusg5"`
	Lstwtusg6  *float64 `json:"lstwtusg6"`
	Lstwtusg7  *float64 `json:"lstwtusg7"`
	Lstwtusg8  *float64 `json:"lstwtusg8"`
	Lstwtusg9  *float64 `json:"lstwtusg9"`
	Lstwtusg10 *float64 `json:"lstwtusg10"`
	Lstwtusg11 *float64 `json:"lstwtusg11"`
	Lstwtusg12 *float64 `json:"lstwtusg12"`
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
	YearMonth  string             `json:"year_month"`
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
