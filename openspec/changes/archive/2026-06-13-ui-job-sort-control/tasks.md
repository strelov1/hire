## 1. Backend: flip the empty-query default

- [x] 1.1 Update `TestSearchJobs_DefaultSortIsCreatedAtForEmptyQuery` in
  `internal/handler/search_test.go` to expect `posted_at:desc` for an empty `q`
  with no sort (rename the test accordingly); keep the relevance-for-text-query
  and explicit-sort-wins cases (RED).
- [x] 1.2 Change `searchSort` in `internal/handler/search.go` to default to
  `posted_at:desc` (not `created_at:desc`) for an empty query; `created_at`
  stays in the `searchSortable` allowlist (GREEN).
- [x] 1.3 `go build ./... && go vet ./... && go test ./internal/handler/` green.

## 2. Frontend: sort in the filter model + URL

- [x] 2.1 Add `sort: 'posted_at' | 'created_at'` to `JobFilters` with
  `emptyFilters()` defaulting to `'posted_at'`; serialize in `filtersToParams`
  (omit `sort` when it equals the default) and parse in `filtersFromParams`;
  add a `setSort` method to `FilterStore`.
- [x] 2.2 `npm run check` (svelte-check) and lint pass in `web/`.
  (svelte-check 0 errors; the changed file is lint-clean — the only `npm run
  lint` failure is a pre-existing oxlint error in `JobsView.svelte:35`,
  unrelated to this change and left untouched per surgical-changes.)

## 3. Frontend: the sort control

- [x] 3.1 Add a labelled sort `<select>` to the `JobsView` toolbar (next to the
  search input) with "Date posted" (`posted_at`) and "Recently added"
  (`created_at`), bound to `filters.value.sort` via `filters.setSort(...)`.
- [x] 3.2 `npm run check` and lint pass; manually confirm the three spec
  scenarios (default, switch, restore-from-URL) against a running stack.
  (Verified live via vite dev + agent-browser: default control = "Date posted"
  with no `?sort=`; selecting "Recently added" → URL `?sort=created_at`;
  `?sort=created_at` restores "Recently added"; `?sort=garbage` falls back to
  "Date posted". The list-reorder half needs backend+Meili data and was not run.)
