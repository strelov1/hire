## 1. Workable adapter

- [x] 1.1 Write a failing `fakeHTTP` test: maps a canned widget response to a Job — title, url (from shortcode), location (city/state/country), remote (telecommuting), sanitized HTML description, external_id = shortcode, posted_at
- [x] 1.2 Implement `workable.go` (provider key `workable`, one `GetJSON` to the widget endpoint, `sanitizeHTML(description)`); make tests green

## 2. Recruitee adapter

- [x] 2.1 Write a failing test: maps a canned offers response; description combines `description` + `requirements` (sanitized), remote from the `remote` bool, external_id = id, including an offer with empty `requirements`
- [x] 2.2 Implement `recruitee.go` (provider key `recruitee`, one `GetJSON`, assemble description+requirements via `sanitizeHTML`); make tests green

## 3. SmartRecruiters adapter

- [x] 3.1 Write a failing test for pagination: a two-page postings fixture is fully collected (offset loop), and a per-posting detail fixture yields a sanitized HTML description from `jobAd.sections`
- [x] 3.2 Write a failing test: a posting whose detail fetch fails is skipped, the rest of the board still returns
- [x] 3.3 Implement `smartrecruiters.go` (provider key `smartrecruiters`): paginate postings, bounded-concurrency detail fetch, assemble sections via `sanitizeHTML`, external_id = posting id; make tests green

## 4. Registration

- [x] 4.1 Register the three adapters in `sources.All`; confirm config validation accepts the new provider keys

## 5. Verify

- [x] 5.1 `go build ./... && go vet ./... && go test ./...` all green

## 6. Data follow-up (boards + ingest)

- [x] 6.1 Re-harvest OpenJobs (outscal/OpenJobs companies_v2.json) for workable/recruitee/smartrecruiters slugs, validate each live (>0 postings), append working boards to `sources.yml`
- [x] 6.2 Re-ingest (`go run ./cmd/ingest`); confirm the new providers land jobs and spot-check a Workable (Hugging Face), Recruitee, and SmartRecruiters job hold sanitized HTML
