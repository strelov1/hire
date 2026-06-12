Each group is one audit finding, implemented one at a time. For Go tasks the
discipline is: confirm the relevant tests are green, make the change, then
`go build ./... && go vet ./... && go test ./...` stays green. For tasks that
genuinely add/restore behavior (5.x slug query, 9.x vocab guard) write the test
first (RED). Web tasks verify via `svelte-check` + lint (no unit runner).

## 1. Dead code: remove unused `search.Client.DeleteJob`

- [x] 1.1 Confirm zero callers (`grep -rn DeleteJob --include='*.go'`), then delete the `DeleteJob` method in `internal/search/client.go` and drop the now-orphaned `strconv` import from that file (leave `filter.go`'s own import).
- [x] 1.2 `go build ./... && go vet ./... && go test ./...` green.

## 2. Dead code: remove `enrich` `Claimed.Attempts` field

- [x] 2.1 Drop `Attempts int` from the `Claimed` struct (`internal/enrich/runner.go`), drop the `Attempts: int(r.Attempts)` mapping (`cmd/enrich/store.go`), and remove `o.attempts` from the `ClaimEnrichmentBatch` RETURNING clause (`internal/db/queries/enrichment.sql`) keeping `id, job_id, target_version`. Do NOT touch `RecordEnrichmentFailure`'s `RETURNING attempts` (it is read).
- [x] 2.2 Run `make sqlc` and commit the regenerated `internal/db/enrichment.sql.go`.
- [x] 2.3 `go build ./... && go vet ./... && go test ./...` green (incl. `runner_test.go`); enrichment queue integration test green (`-tags=integration`).

## 3. Dead code: remove unused `pipeline.Runner.Concurrency` knob

- [x] 3.1 Delete the `Concurrency int` field and the `limit := r.Concurrency; if limit <= 0 { ... }` fallback in `internal/pipeline/pipeline.go`; build the semaphore directly as `make(chan struct{}, defaultConcurrency)`. Update the two doc comments referencing the field.
- [x] 3.2 `go build ./... && go vet ./... && go test ./...` green.

## 4. Dedup: one `listResponse` helper for the list envelope

- [ ] 4.1 Add free func `listResponse(c *fiber.Ctx, data any, total int64, limit, offset int) error` next to `pageParams` in `internal/handler/handler.go`.
- [ ] 4.2 Replace the verbatim `{data, meta}` envelopes in `ListJobs` (jobs.go), `ListCompanies` (companies.go), `SearchJobs` (search.go) with `return listResponse(...)`. Single-item `{data}` handlers untouched.
- [ ] 4.3 `go build ./... && go vet ./... && go test ./...` green (handler tests assert the same shape).

## 5. Efficiency: slim `GetJobIDBySlug` for the view/apply path

- [ ] 5.1 RED — add/extend a test asserting the view/apply interaction path resolves a job by slug to its id (and 404s on an unknown slug) without depending on the heavy columns.
- [ ] 5.2 Add `-- name: GetJobIDBySlug :one` / `SELECT id FROM jobs WHERE public_slug = $1;` to `internal/db/queries/jobs.sql`; run `make sqlc`; commit regenerated `jobs.sql.go`.
- [ ] 5.3 Change `interactionParams` (`internal/handler/user_jobs.go`) to call `GetJobIDBySlug`, using the returned int64 id directly (drop the `job.ID` deref); update the doc comment that names `GetJobBySlug`. Leave `GetJobBySlug` (`SELECT *`) for the public detail handler.
- [ ] 5.4 `go build ./... && go vet ./... && go test ./...` green.

## 6. Simplify: collapse smartRecruiters parallel slices

- [ ] 6.1 In `internal/sources/smartrecruiters.go` replace `jobs []Job` + `found []bool` with a single `jobs := make([]*Job, len(postings))` (nil = skipped); write `jobs[i] = &j` on success; compact non-nil into `[]Job`. Add a one-line comment ("nil = detail fetch failed, skipped"). Do NOT use the `ExternalID != ""` sentinel.
- [ ] 6.2 `go test ./internal/sources/...` green, incl. `TestSmartRecruitersFetchSkipsFailedDetail`; then full `go build && go vet && go test ./...`.

## 7. Frontend: inline the `get<T>()` pass-through

- [ ] 7.1 Delete `get<T>(path)` in `web/src/lib/api.ts`; replace its 5 call sites (`listJobs`, `getJob`, `searchJobs`, `listCompanies`, `getCompany`) with `request<T>(...)`. Keep `call` and `request`.
- [ ] 7.2 `npm run check` (svelte-check) + lint clean.

## 8. Frontend: extract `ui/input.svelte` primitive

- [ ] 8.1 Add `web/src/lib/ui/input.svelte` mirroring `button.svelte`/`badge.svelte`: shared class constant merged via `cn`, `...rest` spread, `value = $bindable()` for `bind:value`. Export from `web/src/lib/ui/index.ts`.
- [ ] 8.2 Replace the three duplicated inputs (`JobsView.svelte`, `CompaniesView.svelte` via `value`+`oninput`; `facets/SearchSelect.svelte` via `bind:value`) with `<Input .../>`, passing the width override via `class`. Keep debounce/URL logic in the call sites. Leave `TokenInput` alone.
- [ ] 8.3 `npm run check` + lint clean; visual glance at jobs/companies search + facet filter.

## 9. Frontend: `Badge variant="secondary"` for JobRow chips

- [ ] 9.1 In `web/src/lib/components/JobRow.svelte` replace both inline secondary-chip spans (tag + skill) with `<Badge variant="secondary">…</Badge>` (import from `$lib/ui`), matching `JobView.svelte`. Render both as plain Badge (no `font-normal` override).
- [ ] 9.2 `npm run check` + lint clean; visual glance at the jobs/company list rows.

## 10. Frontend: finish the generic And/Or facet toggle

- [ ] 10.1 Add `matchAll: boolean` to `FacetState` (`web/src/lib/filters.svelte.ts`), mirroring `exclude`; replace the top-level `skillsAnd` field + `setSkillsAnd` method with `setMatchAll(param, on)`.
- [ ] 10.2 Update serialization: `filtersToParams` emits `${param}_mode=and` when `matchAll && !exclude && values.length > 1`; `filtersFromParams` sets `matchAll = p.get(`${param}_mode`) === 'and'` per facet.
- [ ] 10.3 In `facets/FacetSection.svelte` read/write `store.facet(def.param).matchAll` via `setMatchAll(def.param, ...)`; keep `hasAndOr` as the honest gate.
- [ ] 10.4 `npm run check` + lint clean; confirm the backend `skills_mode=and` test still passes (`go test ./internal/handler/...`) and the skills facet behaves identically end-to-end.

## 11. Guard: enum vocab drift (Go ↔ web `facets.ts`)

- [ ] 11.1 RED — add a Go test in `internal/enrich` asserting each faceted closed `*Values` list matches a checked-in expected set (fails on drift). Drop the test only if it reads as ceremony once written.
- [ ] 11.2 Add a one-line source-of-truth comment in `web/src/lib/facets.ts` pointing at `internal/enrich/enrichment.go`. Do NOT build a codegen pipeline.
- [ ] 11.3 `go test ./internal/enrich/...` green; `npm run check` clean.

## 12. Final verification

- [ ] 12.1 Full backend: `go build ./... && go vet ./... && go test ./...` (+ integration tags where touched).
- [ ] 12.2 Full web: `npm run check` + lint + `npm run build`.
- [ ] 12.3 Confirm no wire-shape change: list/search/view/apply/facet responses byte-identical to pre-change.
