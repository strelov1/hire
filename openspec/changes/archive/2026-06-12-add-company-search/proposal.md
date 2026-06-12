## Why

The `/companies` page lists every company with no way to find one by name. As
the catalog grows past a screenful, users have to page through "Load More" to
locate a specific company. The jobs list already has search; companies should
offer the same affordance.

## What Changes

- The `GET /api/v1/companies` endpoint gains an optional `?q=` query parameter
  that filters companies by a case-insensitive name match. An absent or empty
  `q` preserves today's behavior (full list).
- `meta.total` reflects the count of companies matching `q`, so pagination over
  search results is correct.
- The `/companies` page in the web SPA gains a "Search companies…" input. Typing
  filters the list (debounced) and mirrors the query into the URL (`?q=`) so a
  search is shareable and survives reload and back/forward.
- The companies list gains a result count line and a "No matching companies."
  empty state.

Non-goals: no Meilisearch index for companies (a single name field does not
warrant it); search is a SQL `ILIKE` over the existing list query.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `companies`: the company-list endpoint requirement gains an optional
  name-search filter (`?q=`) backed by a SQL `ILIKE`, with `meta.total`
  reporting the filtered count.
- `web-frontend`: the companies-list requirement gains a name-search input that
  filters the list and is mirrored to the URL query.

## Impact

- Backend: `internal/db/queries/companies.sql` (`ListCompanies`, `CountCompanies`
  gain a name filter), regenerated `internal/db/companies.sql.go` via `make sqlc`,
  and `internal/handler/companies.go` (`ListCompanies` reads `q`).
- Frontend: `web/src/lib/api.ts` (`listCompanies` gains a `q` argument) and
  `web/src/lib/components/CompaniesView.svelte` (search input + URL sync).
- No schema migration, no new dependency, no Meilisearch change.
