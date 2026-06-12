## 1. Enrichment contract

- [ ] 1.1 Add `RemoteScope string` (`json:"remote_scope,omitempty"`) and `Regions []string` (`json:"regions,omitempty"`) to `Enrichment` in `internal/enrich/enrichment.go`, in the Location/eligibility group, with comments noting they are remote-only and that empty scope means unknown (not global).
- [ ] 1.2 Add `RemoteScopeValues = []string{"global","regional","national"}` and `RegionValues = []string{"EU","EMEA","EEA","UK","AMERICAS","NORTH_AMERICA","LATAM","APAC","MENA","AFRICA"}` to the controlled-vocabulary block.
- [ ] 1.3 Extend `Enrichment.Validate`: add `remote_scope` to the scalar enum table and an element-wise `regions` check against `RegionValues`; do NOT add any cross-field consistency check.
- [ ] 1.4 Unit tests in `enrichment_test.go`: valid `remote_scope`/`regions` pass; out-of-vocab values are rejected with the offending field; empty passes; `regional` with empty `regions` is NOT rejected.

## 2. LLM extraction prompt

- [ ] 2.1 In `internal/enrich/langchain.go`, extend the extraction instructions to emit `remote_scope` (`global` only on an explicit worldwide/anywhere signal; `regional`→fill `regions`; `national`→fill `countries`; else omit) and `regions` from `RegionValues`, only when `work_mode` is `remote`. Keep the existing `Remote flag` hint line.
- [ ] 2.2 Reframe the stale "fallback" wording around the `remote` input to "source-provided remote hint (enrichment input)" where it appears (`langchain.go`, `enrich.Provider`).

## 3. Search: derived remote_type

- [ ] 3.1 Add a sibling `RemoteType []string` (`json:"remote_type,omitempty"`) to `JobDocument` in `internal/search/document.go` (outside the embedded enrichment), and a `remoteType(enrich.Enrichment) []string` helper (gate on `work_mode==remote`; global→`["global"]`, regional→lowercased regions, national→lowercased countries, else nil). Populate it in `FromJob`.
- [ ] 3.2 In `internal/search/client.go` `indexSettings()`, add `"remote_type"` to `FilterableAttributes` and remove `"remote"`.
- [ ] 3.3 Unit tests for `remoteType`: each scope branch, the non-remote gate, the unknown (empty) case.

## 4. Search handler param

- [ ] 4.1 In `internal/handler/search.go`, add `"remote_type": "remote_type"` to `searchStringFacets` and remove the `?remote=true` filter block in `buildSearchFilter`.

## 5. Remove the public raw remote flag

- [ ] 5.1 Remove the `Remote bool` field and its `FromRow` assignment from `internal/jobview/jobview.go`; update `jobview_test.go`.
- [ ] 5.2 Update `internal/search/search_integration_test.go` literals that set `Remote` on the job/document.
- [ ] 5.3 Confirm the `jobs.remote` column, `db` model, sqlc queries, `sources/*`/`pipeline` `Remote` field, and `cmd/ingest`/`cmd/enrich` paths are untouched (column + hint retained).

## 6. Frontend

- [ ] 6.1 `web/src/lib/types.ts`: add `remote_scope?: string` and `regions?: string[]` to the `Enrichment` type; remove `remote: boolean` from the job type.
- [ ] 6.2 `web/src/lib/facets.ts`: add a `REMOTE_TYPE` options list (`global`/`ru`/`eu`/`us` → Global/Russia/Europe/USA) and a `{ param: 'remote_type', label: 'Remote type', control: 'pills', options: REMOTE_TYPE, excludable: true }` entry placed immediately after `work_mode`.
- [ ] 6.3 `web/src/lib/enrichment.ts`: drop the `remote` param and `?? job.remote` fallback from `workArrangement()`/`cardTags()`; add `REMOTE_SCOPE`/`REGION` label maps and show a remote job's reach (Global / region(s) / country(ies)).
- [ ] 6.4 Update `web/src/lib/components/JobView.svelte` (remove the `job.remote && !e.work_mode` branch) and `JobRow.svelte` (drop the `remote` arg) to render reach from work_mode/remote_scope.

## 7. Verify

- [ ] 7.1 `go build ./... && go vet ./... && go test ./...` pass.
- [ ] 7.2 Frontend builds (`web`); the "Remote type" facet renders under "Work format" and filters; a global-remote job shows "Global" reach, a regional shows its region, a national shows its country.
- [ ] 7.3 `openspec validate add-remote-scope` passes.
