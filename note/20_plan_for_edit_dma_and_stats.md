# Plan: Filter DMA region stats and bulk-create feature collections

Source: `note/19_requirement_edit_dma_insert_stats_region_20260813.md`

## Scope and public contract

### `GET /api/dma/stats-region`

- Keep `region` required (`1..10`) and keep the existing `column` contract (`prswtusg` or `lstwtusg1`, default `prswtusg`).
- Add optional query parameter `pwa_code`.
- Without `pwa_code`, preserve the current result: every DMA whose `pwa_code` belongs to the requested region.
- With `pwa_code`, query the same region customer table but return only DMA rows whose `dma.pwa_code` exactly equals the supplied value (for example `5531011`).
- Preserve the response envelope `{success, data, count}` and the existing usage/population formulas.

### `POST /api/features/:shape/:pwaCode`

- Keep the current `step_test` FeatureCollection contract.
- Allow `dma_boundary` and `flow_meter` create requests to use a GeoJSON `FeatureCollection` as well.
- Preserve backward compatibility: `dma_boundary` and `flow_meter` continue to accept the existing single-feature request body.
- A collection must have `type: "FeatureCollection"`, at least one member, and every member must have `type: "Feature"` plus the existing `geometry`, `properties`, and optional `dma_id` fields.
- Keep the client `id` handling shape-specific: `step_test` still requires a non-empty `id` and maps it to `properties.stepName`; `dma_boundary` and `flow_meter` do not use the client `id` as the stored MongoDB identity.
- Validate the entire collection before the first insert using each shape's existing rules. Check overlap between members for polygon shapes (`dma_boundary` and `step_test`); `flow_meter` retains its point/within-DMA rules.
- On success, insert members in request order and return HTTP 201 with a GeoJSON `FeatureCollection` and `count`.
- If parsing or preflight validation fails, insert nothing. If persistence fails after earlier members succeeded, stop at the failing index and keep the existing non-transactional partial-write behavior.

## Implementation seams

1. Thread optional `pwa_code` from the DMA handler through `DMAService.GetStatsRegion` to the customer repository.
2. Extend the stats-region query builder to select either the region-prefix predicate or an exact `pwa_code` predicate while retaining parameter binding and the existing column allowlist.
3. Generalize the FeatureCollection parser/orchestrator and collection validator around a `shape` argument instead of duplicating the `step_test` path. Keep the public HTTP route unchanged.
4. Update API documentation/examples that describe these request parameters and bodies.

## TDD sequence

1. RED -> GREEN: repository/query test proves no `pwa_code` keeps the region-prefix filter.
2. RED -> GREEN: repository/query test proves supplied `pwa_code` uses an exact bound predicate; handler test covers forwarding and input acceptance through the HTTP interface where practical.
3. RED -> GREEN: HTTP create test for a two-member `dma_boundary` FeatureCollection, including ordered calls and FeatureCollection/count response.
4. RED -> GREEN: HTTP create test for a two-member `flow_meter` FeatureCollection.
5. RED -> GREEN: malformed/empty collection and collection preflight failures perform zero inserts; failing member errors include its zero-based index.
6. Regression: `step_test` FeatureCollection mapping and legacy single-feature creates for `dma_boundary`/`flow_meter` remain unchanged.
7. Run `gofmt`, focused package tests after each slice, then `go vet ./...`, `go test ./...`, and `go build ./...`.

## Review and delivery

- Compare the completed work with the pre-work fixed point `3d6f786929cc6e69608bb0f276d466a2081b3f09` on separate Standards and Spec axes.
- Preserve all pre-existing uncommitted work; stage and commit only files/hunks belonging to this requirement.
