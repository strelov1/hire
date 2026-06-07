## 1. Schema

- [x] 1.1 Add `migrations/0002_companies.sql`: create `companies` (`slug TEXT PRIMARY KEY`, `name TEXT NOT NULL`, `created_at`/`updated_at TIMESTAMPTZ DEFAULT now()`)
- [x] 1.2 In the same migration, add `jobs.company_slug TEXT NOT NULL DEFAULT ''` and `CREATE INDEX jobs_company_slug_idx ON jobs (company_slug)`
- [x] 1.3 Recreate the dev volume (`docker compose down -v && make up`) and verify the new table/column exist via `make psql`

## 2. DB layer (sqlc)

- [x] 2.1 Add `internal/db/queries/companies.sql`: `UpsertCompany` (`ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, updated_at = now()`), `ListCompanies`, `GetCompany`
- [x] 2.2 Update `internal/db/queries/jobs.sql`: add `company_slug` to `UpsertJob` insert/update; add `ListJobsByCompany` filtering on `company_slug`; add company job-count to the companies list query (computed on the fly)
- [x] 2.3 Run `make sqlc` and commit the generated `internal/db/*.go`

## 3. Normalization + write path

- [x] 3.1 Add a Go `slugify(name)` helper (lowercase, trim, collapse whitespace, strip punctuation; empty name → empty slug) with unit-style coverage of the documented cases
- [x] 3.2 Update the `UpsertJob` write path to compute `company_slug`, upsert the company when the slug is non-empty, and write both `company` and `company_slug` on the job — as one atomic unit (CTE, else a transaction)

## 4. API

- [x] 4.1 Add handlers for `GET /api/v1/companies` (list, with on-the-fly counts) and `GET /api/v1/companies/:slug` (company + its jobs via `ListJobsByCompany`; 404 when the company is unknown), using the existing `{data, meta}` / `{data}` response shapes
- [x] 4.2 Wire the new routes in `handlers.Register`

## 5. Verify

- [x] 5.1 `go build ./... && go vet ./...`
- [x] 5.2 Manually exercise the endpoints: a company appears in `/api/v1/companies` and its jobs show under `/api/v1/companies/:slug`; no `companies`↔`jobs` join in the read queries
