## 1. Enrichment contract

- [x] 1.1 Add a single `Regions []string` (`json:"regions,omitempty"`) to `Enrichment` in `internal/enrich/enrichment.go`, in the Location/eligibility group, with a comment: reach codes, remote-only, empty = unknown, `global` explicit.
- [x] 1.2 Add `RegionValues = []string{"global","eu","emea","eea","uk","americas","north_america","latam","apac","mena","africa","us","ru"}` to the controlled-vocabulary block. No `remote_scope`.
- [x] 1.3 Extend `Enrichment.Validate`: element-wise `regions` check against `RegionValues` (no scope, no cross-field check).
- [x] 1.4 Unit tests in `enrichment_test.go`: valid `regions` pass (incl. `global`); out-of-vocab rejected with the offending field; empty passes; round-trip with `regions`; `regions=[global]` serializes vs empty (global ≠ unknown).

## 2. LLM extraction prompt

- [x] 2.1 In `internal/enrich/langchain.go`, extend the extraction instructions to emit `regions` from `RegionValues` (use `global` only on an explicit worldwide/anywhere signal), only when `work_mode` is `remote`. Keep the `Remote flag` hint line.
- [x] 2.2 Mark the `Remote flag` prompt line as a source-provided remote hint (enrichment input), not a public/fallback field.

## 3. Search: filter regions directly

- [x] 3.1 In `internal/search/client.go` `indexSettings()`, add `"enrichment.regions"` to `FilterableAttributes` and remove `"remote"`. No derived document field.
- [x] 3.2 Confirm `internal/search/document.go` carries `regions` automatically via the embedded enrichment (no code change beyond removing any derived-field scaffolding).

## 4. Search handler param

- [x] 4.1 In `internal/handler/search.go`, add `"regions": "enrichment.regions"` to `searchStringFacets` and remove the `?remote=true` filter block in `buildSearchFilter`.

## 5. Remove the public raw remote flag

- [x] 5.1 Remove the `Remote bool` field and its `FromRow` assignment from `internal/jobview/jobview.go`; update `jobview_test.go` (assert `remote` absent from the wire).
- [x] 5.2 Confirm `internal/search/search_integration_test.go` literals use `db.Job` (column retained) — no change needed.
- [x] 5.3 Confirm the `jobs.remote` column, `db` model, sqlc queries, `sources/*`/`pipeline` `Remote` field, and `cmd/ingest`/`cmd/enrich` paths are untouched (column + hint retained).

## 6. Frontend

- [x] 6.1 `web/src/lib/types.ts`: add `regions?: string[]` to the `Enrichment` type; remove `remote: boolean` from the job type.
- [x] 6.2 `web/src/lib/facets.ts`: add a `REGION` options list (`global`/`ru`/`eu`/`us` → Global/Russia/Europe/USA) and a `{ param: 'regions', label: 'Region', control: 'pills', options: REGION, excludable: true }` entry placed immediately after `work_mode`.
- [x] 6.3 `web/src/lib/enrichment.ts`: drop the `remote` param and `?? job.remote` fallback from `workArrangement()`/`cardTags()`; add a `REGION` label map and a `remoteReach()` helper that reads `regions` (Global / region(s) / country code(s)); show reach on cards and in `summaryFacets`.
- [x] 6.4 Update `web/src/lib/components/JobView.svelte` (remove the `job.remote && !e.work_mode` branch) and `JobRow.svelte` (drop the `remote` arg).

## 7. Verify

- [x] 7.1 `go build ./... && go vet ./... && go test ./...` pass.
- [x] 7.2 Frontend `npm run check` + build; the "Region" facet renders under "Work format" and filters; a global-remote job shows "Global" reach, a regional shows its region.
- [x] 7.3 `openspec validate add-remote-scope` passes.
