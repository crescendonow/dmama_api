# Plan: add `year_month` to DMA stats responses

## Source

- Requirement: `note/19_requirement_edit_dma_insert_stats_region_20260825.md`
- Confirmed on 2026-08-25:
  - Change the billing cutoff from day 20 to day 15.
  - On days 1-15, `prswtusg` represents the previous calendar month.
  - On days 16 through month end, `prswtusg` represents the current calendar month.
  - Expand `/api/dma/stats-region` to accept `lstwtusg1` through `lstwtusg12`.
  - Keep legacy `/api/dma/stats` columns (`use_water`, `use_jan` through `use_dec`) for backward compatibility and return `year_month: ""` for them.

## Public behavior

Add `year_month` to every item returned by:

- `GET /api/dma/stats`
- `GET /api/dma/stats-region`

`year_month` is a six-character Buddhist Era value in `YYYYMM` format. Month values are zero-padded.

For a request evaluated on 2026-08-25:

| Column | `year_month` |
|---|---|
| `prswtusg` | `256908` |
| `lstwtusg1` | `256907` |
| `lstwtusg2` | `256906` |
| `lstwtusg12` | `256808` |

For a request evaluated on 2026-08-15, `prswtusg` is `256907`; on 2026-08-16 it is `256908`.

When `/api/dma/stats` receives `year` and `month`, keep the existing Gregorian input contract and column resolution behavior. The returned `year_month` describes the resolved database column and is formatted in Buddhist Era.

## Implementation

1. Add `YearMonth string \`json:"year_month"\`` to `model.DMAStats`.
2. Put billing-period calculation behind a service function that accepts the resolved column and a supplied `time.Time`:
   - determine the base billing month using cutoff day 15;
   - subtract `N` months for `lstwtusgN`;
   - add 543 to the Gregorian year;
   - return an empty string for the confirmed legacy columns.
3. Change the existing `ResolveStatsColumn` cutoff from day 20 to day 15 so explicit `year`/`month` requests select the matching column.
4. Attach the calculated `year_month` to `/api/dma/stats` without putting it in the cache key or database query.
5. Attach the same calculated value to every `/api/dma/stats-region` result item.
6. Expand stats-region column validation to `prswtusg` plus `lstwtusg1..12`; continue rejecting other legacy and unsafe column values.
7. Update `template/index.html` for both endpoints: response examples, cutoff rule, output format, and the expanded stats-region allowlist.

## Test-first slices

1. Service tests for cutoff boundaries (day 15/day 16), zero-padded months, Buddhist year, year rollover, and offsets 1/2/12.
2. Service tests for legacy columns returning an empty `year_month` and invalid columns returning an error.
3. Existing explicit `year`/`month` resolution tests updated to cutoff day 15.
4. Repository/service validation tests proving stats-region accepts `lstwtusg1..12` and rejects unsupported/unsafe columns.
5. Response-preparation tests proving `year_month` is present and cached numeric data remains unmodified.
6. Documentation assertions or focused text checks for both endpoint examples and cutoff wording.

## Verification

- Run `gofmt` on changed Go files.
- Run focused package tests after each RED/GREEN slice.
- Run `go test ./...` once all changes are complete.
- Review the final diff against this plan and the source requirement.
- Commit only files belonging to this requirement; preserve unrelated worktree changes.
