## 1. Backend: query layer

- [x] 1.1 Add an integration test (`-tags=integration`, testcontainers) asserting
  `ListCompanies` with a non-empty `search` returns only name-matching rows
  (case-insensitive) and with empty `search` returns all rows; and that
  `CountCompanies` with `search` returns the filtered count.
- [x] 1.2 Add the optional name filter to `ListCompanies` and `CountCompanies` in
  `internal/db/queries/companies.sql`
  (`WHERE sqlc.arg('search')::text = '' OR <name> ILIKE '%' || sqlc.arg('search') || '%'`),
  switching `ListCompanies` to named sqlc args (`search`/`limit`/`offset`).
- [x] 1.3 Run `make sqlc` and commit the regenerated `internal/db/companies.sql.go`;
  confirm the integration test passes.

## 2. Backend: handler

- [x] 2.1 Add a test covering `GET /api/v1/companies?q=` filtering and
  `meta.total` reflecting the filtered count (empty `q` = full list unchanged).
- [x] 2.2 Update `Handler.ListCompanies` in `internal/handler/companies.go` to
  read `c.Query("q")` and pass it as `Search` to both `ListCompanies` and
  `CountCompanies`.

## 3. Frontend: API client

- [x] 3.1 Change `listCompanies` in `web/src/lib/api.ts` to
  `listCompanies(q, limit, offset)`, building `?q=&limit=&offset=` and omitting
  `q` when empty.

## 4. Frontend: companies view

- [x] 4.1 Add a "Search companies…" input to
  `web/src/lib/components/CompaniesView.svelte` bound to a local `q` state
  initialized from `router.query.get('q')`.
- [x] 4.2 Add the `$effect` that pulls `q` from `router.search` on back/forward
  (guarded by `if (urlQ !== q)`), and the debounced (300ms) `$effect` that writes
  `?q=` via `router.setQuery` and recreates the `Paginator` with the current `q`;
  clear the debounce timer on unmount.
- [x] 4.3 Add the result count line ("N companies") and the
  "No matching companies." empty state.

## 5. Verify

- [x] 5.1 Run `go build ./... && go vet ./...`, `go test ./...`, and the DB
  integration test; build the web app (`npm run build` / type-check) and verify
  the search works end-to-end on `/companies` (typing filters, URL updates,
  back/forward restores).
