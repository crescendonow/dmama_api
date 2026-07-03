# Add `/api/dma/stats-region`

## Summary
- Source requirement: `note/13_requirements_crud_20260703.md`.
- Add exactly one authenticated endpoint: `GET /api/dma/stats-region`.
- Keep existing DMA/API endpoint behavior unchanged.

## Public API
- Request: `/api/dma/stats-region?region=9&column=prswtusg`
- `region` is required and must be `1..10`.
- `column` is optional, defaults to `prswtusg`, and accepts only `prswtusg` or `lstwtusg1`.
- Success response uses an array and count:

```json
{
  "success": true,
  "data": [
    {
      "pwa_code": "5511013",
      "dma_id": "6",
      "column": "prswtusg",
      "usage": {
        "total": 37260,
        "house": 27713,
        "government": 1247,
        "business_small": 6099,
        "business_large": 1256
      },
      "population": {
        "total": 2484,
        "house": 2127,
        "government": 11,
        "business_small": 298,
        "business_large": 48
      }
    }
  ],
  "count": 1
}
```

## Implementation
- Register only `api.Get("/dma/stats-region", dmaH.GetStatsRegion)` beside the existing DMA stats route.
- Add separate handler/service/repository flow for stats-region; do not route through or modify `/api/dma/stats` behavior.
- Reuse `model.DMAStats` for each response item.
- Resolve stats-region column separately from normal stats so only `prswtusg` and `lstwtusg1` are allowed.
- Query once per request with grouped SQL:
  - Select from `pwa_dma.dma_boundary`.
  - Filter DMA rows by `repository.ZonePrefix(region)` using `dma.pwa_code LIKE $1`.
  - LEFT JOIN `giswebm_stamp.r{region}_bl_customer`.
  - Group by `dma.pwa_code, dma.dma_id`.
  - Use the same usage/population category formulas as `GetStatsInDMA`.
  - Return zero aggregates for DMA boundaries with no matching customers.

## Test Plan
- Handler tests cover missing `region`, invalid `region`, and invalid `column`.
- Service tests cover default `prswtusg`, accepted `prswtusg`/`lstwtusg1`, and rejected `ltwtusg1`, `lstwtusg2`, SQL-like input.
- Repository query tests cover region table selection, region prefix filter, LEFT JOIN, spatial intersection, grouping, and ordering.
- Run `go test ./...`.

## Assumptions
- Response `data` is an array.
- Requirement typo `ltwtusg1` is treated as `lstwtusg1`; no alias is accepted.
- `column` defaults to `prswtusg`.
- No database migration is required.