## 1. Test scaffolding (body-aware fake)

- [x] 1.1 Add a body-aware test HTTP fake (e.g. `gqlHTTP` in `gem_test.go`, or extend the shared test fakes): Gem sends both list and detail as `POST /api/public/graphql`, distinguished only by the request body, so the fake MUST route the canned response on the request body (`operationName` / the `extId` in `variables`), not the URL
- [x] 1.2 Add canned GraphQL JSON fixtures inline: a `JobBoardList` response with 2+ postings (one with `isRemote: true`, one `IN_OFFICE`, one missing `firstPublishedTsSec`) and matching `ExternalJobPostingQuery` detail responses carrying `descriptionHtml`

## 2. parseEpochSeconds helper (TDD)

- [x] 2.1 Test `parseEpochSeconds`: a positive Unix-seconds value → the corresponding UTC `*time.Time`; `0`/absent → nil (mirrors `parseEpochMillis`)
- [x] 2.2 Implement `parseEpochSeconds` in `internal/sources/source.go` beside `parseEpochMillis`

## 3. Gem adapter — Provider + list→detail Fetch (TDD)

- [x] 3.1 Test `Provider()` returns `"gem"`
- [x] 3.2 Test `Fetch` issues `JobBoardList` with `boardId = e.Board` (vanity path), then one `ExternalJobPostingQuery` per posting, and yields the normalized jobs; assert the request bodies carry the right `operationName` and `boardId`/`extId`
- [x] 3.3 Implement `gem.go`: `gem` struct over `HTTPClient`, `NewGem`, embedded query string constants, `Fetch` = `JobBoardList` once then `fetchDetails(postings, gemDetailWorkers, detail)` using the shared bounded-pool helper

## 4. Field mapping (TDD)

- [x] 4.1 Test mapping: `ExternalID = extId`; `URL = https://jobs.gem.com/<board>/<extId>`; `Title`; `Company = e.Company`; `Location = joinNonEmpty(city, isoCountry)` of the first location; `Description = sanitizeHTML(descriptionHtml)` (assert active content stripped, structure kept)
- [x] 4.2 Test `Remote`: true when the first location's `isRemote` is true OR `job.locationType == "REMOTE"`; false otherwise
- [x] 4.3 Test `PostedAt`: derived from `firstPublishedTsSec`; nil when the timestamp is absent/zero

## 5. Isolation and empty-board behavior (TDD)

- [x] 5.1 Test a failed `ExternalJobPostingQuery` for one posting drops only that posting and still yields the rest (no board abort)
- [x] 5.2 Test an empty `jobPostings` list yields zero jobs and no error

## 6. Registration and configuration

- [x] 6.1 Register `NewGem(c)` in `sources.All` (`internal/sources/source.go`); confirm `reg`'s duplicate-provider guard still passes
- [x] 6.2 Add at least one verified `gem` entry to `sources.yml` (`provider: gem`, `board: <vanity-path>`)

## 7. Verification

- [x] 7.1 `go build ./... && go vet ./... && go test ./internal/sources/...` all green
- [x] 7.2 Smoke-run `go run ./cmd/ingest` against the configured `gem` board (or a focused live check) to confirm real postings normalize and upsert; confirm the validated-registry fail-fast still accepts `gem`
