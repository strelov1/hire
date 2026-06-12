## 1. Personio adapter (single-request, XML)

- [x] 1.1 Add a `GetXML` method to `HTTPClient` (mirror `GetJSON`, stdlib `encoding/xml`) with a test
- [x] 1.2 Capture the real `…jobs.personio.com/xml` `<position>` schema (used to shape the test)
- [x] 1.3 Write a failing `personio_test.go`: maps `<position>` → `Job`, description from `jobDescriptions` sanitized, empty feed → zero jobs no error
- [x] 1.4 Implement `internal/sources/personio.go` (provider `personio`) until tests pass; register in `sources.All`

## 2. Breezy adapter — DEFERRED (reclassified)

Live probe found Breezy's `/json` list carries **no description**; the body lives in a
`JobPosting` **JSON-LD** block on each posting's HTML page — a different transport class
(raw HTML + ld+json extraction) than this change's clean JSON/XML adapters, shared with
the deferred gem/jazzhr/recruiterbox. Moved to a future "open-web JSON-LD source" change.

- [ ] 2.1 (deferred) Build Breezy in the JSON-LD source change, not here

## 3. Pinpoint adapter (single-request, JSON)

- [x] 3.1 Capture the real `…pinpointhq.com/postings.json` shape (location/workplace_type/body sections)
- [x] 3.2 Write a failing `pinpoint_test.go`: maps `data[]` → `Job`, inline body sanitized, remote from `workplace_type`
- [x] 3.3 Implement `internal/sources/pinpoint.go` (provider `pinpoint`) until tests pass; register in `sources.All`

## 4. Rippling adapter (list + per-posting detail)

- [x] 4.1 Capture the real board list and one posting detail (description is `{company,role}`; role only)
- [x] 4.2 Write a failing `rippling_test.go`: list mapped, per-`uuid` detail for description, company boilerplate excluded, bounded fan-out, failed detail skipped
- [x] 4.3 Implement `internal/sources/rippling.go` (provider `rippling`) until tests pass; register in `sources.All`

## 5. BambooHR adapter (list + per-posting detail)

- [x] 5.1 Capture the real `…/careers/list` (carries `isRemote`) and `…/careers/{id}/detail` shapes
- [x] 5.2 Write a failing `bamboohr_test.go`: list mapped, per-`id` detail for description, location from `joinNonEmpty(city,state,country)`, bounded fan-out, failed detail skipped
- [x] 5.3 Implement `internal/sources/bamboohr.go` (provider `bamboohr`) until tests pass; register in `sources.All`

## 6. Shared detail fan-out refactor (emerged under green)

- [x] 6.1 Extract the bounded detail fan-out into a generic `fetchDetails[P]` in `source.go`
- [x] 6.2 Converge smartrecruiters, rippling, and bamboohr onto it; tests stay green

## 7. Join.com adapter — DEFERRED to its own change

Decision: join.com (GraphQL `candidate-api` / `__NEXT_DATA__`, ~23k companies, ToS check)
ships as a separate change `add-joincom-source`, not here.

- [ ] 7.1 (deferred) Plan and build `add-joincom-source` separately

## 8. Seed boards and verification

- [x] 8.1 Add live-validated boards per new provider to `sources.yml` (3 each; each confirmed ≥1 posting)
- [x] 8.2 `go build ./... && go vet ./... && go test ./...` all green
- [x] 8.3 Ran `cmd/ingest` against a throwaway worktree DB (:5433): 496 jobs ingested, 0 failed, descriptions sanitized HTML (no script/onclick/img), external_id namespaced
