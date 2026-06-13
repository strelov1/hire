## 1. Location parser (`internal/location`)

- [ ] 1.1 Write table tests for `location.Parse` covering real prod strings:
  `Remote - Germany`, `Remote - Europe`, `Remote - UK or Europe`, `Remote`,
  `Remote - Anywhere`, `Burlington, Massachusetts, United States; Remote`,
  `United States`, empty, and an unresolvable token (RED).
- [ ] 1.2 Implement `Parse(location string) (countries, regions []string)` with the
  three dictionaries (name→country, name→region, country→region drawn from
  `enrich.RegionValues`), tokenization on `, ; / | " - " " or "`, dedup, and the
  "never guess / global only on explicit anywhere" rules (GREEN).
- [ ] 1.3 Seed the dictionaries from the high-frequency prod location strings; add a
  test asserting every emitted region ∈ `RegionValues` and every country is ISO
  alpha-2.

## 2. Schema + DB access

- [ ] 2.1 Add migration: `jobs.countries text[] NOT NULL DEFAULT '{}'`,
  `jobs.regions text[] NOT NULL DEFAULT '{}'` (follow existing `migrations/`
  numbering).
- [ ] 2.2 Update `UpsertJob` in `internal/db/queries/jobs.sql` to write `countries`
  and `regions` in INSERT and in `ON CONFLICT DO UPDATE SET`.
- [ ] 2.3 Add `SetJobLocation` query (set `countries`/`regions` by id) for the
  backfill path.
- [ ] 2.4 Run `make sqlc` and commit the regenerated `internal/db`.

## 3. Ingest write path

- [ ] 3.1 Add `Countries`/`Regions` to `pipeline.Job`; have `normalizeJob` call
  `location.Parse(j.Location)` (RED test on `normalizeJob` first).
- [ ] 3.2 Thread the new fields through the `Store.Save` implementation into
  `UpsertJob`; verify a re-ingest refreshes geography.

## 4. Read-time union (`internal/jobview`)

- [ ] 4.1 Add top-level `Regions []string`, `Countries []string` to `jobview.Job`;
  `FromRow` unions `jobs.regions ∪ enrichment.regions` (and countries),
  deduped, then blanks `enrichment.regions`/`enrichment.countries` in the served
  copy (RED tests for union, dedup, and blanking first).

## 5. Search

- [ ] 5.1 Update the index settings: filterable attributes use top-level
  `regions`/`countries`, not `enrichment.regions`; remove the enrichment geography
  dot-path facet.
- [ ] 5.2 Update the search handler's region/country filter to build against the
  top-level path; adjust `internal/search` document/filter tests.

## 6. Enrichment prompt

- [ ] 6.1 Update the `regions` block in `internal/enrich/langchain.go`
  `buildSystemPrompt` to describe `regions` as geographic area for any work mode
  (drop "only when remote"); adjust the prompt test if it asserts that wording.

## 7. Web SPA

- [ ] 7.1 Map the SPA region filter query-param to the top-level `regions` field;
  verify via `svelte-check` + lint (no SPA test runner).

## 8. Backfill

- [ ] 8.1 Add run-once `cmd/backfill-geo`: keyset scan via `ListJobsByIDAfter`,
  `location.Parse`, write via `SetJobLocation`; idempotent and safe to re-run.

## 9. Verification

- [ ] 9.1 `go build ./... && go vet ./... && go test ./...` green.
- [ ] 9.2 Manually confirm the migration → deploy → backfill → reindex order is
  documented (design Migration Plan) and the reindex step is noted for ops.
