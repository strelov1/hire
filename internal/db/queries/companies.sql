-- name: ListCompanies :many
-- Catalog page: companies with their job counts. The job count is computed on
-- the fly (no denormalized counter yet). This is the one acknowledged place a
-- join to jobs is acceptable; LEFT JOIN keeps companies with zero jobs visible.
-- An empty `search` short-circuits the ILIKE, so the same prepared statement
-- serves both the full list and a name search (`search` is a case-insensitive
-- substring of the name).
SELECT c.slug, c.name, count(j.company_slug) AS job_count
FROM companies c
LEFT JOIN jobs j ON j.company_slug = c.slug AND j.closed_at IS NULL
WHERE sqlc.arg('search')::text = '' OR c.name ILIKE '%' || sqlc.arg('search') || '%'
GROUP BY c.slug, c.name
ORDER BY c.name
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountCompanies :one
-- Total companies matching the same optional name filter as ListCompanies, so
-- search pagination reports the filtered total.
SELECT count(*)
FROM companies
WHERE sqlc.arg('search')::text = '' OR name ILIKE '%' || sqlc.arg('search') || '%';

-- name: GetCompany :one
SELECT slug, name, created_at, updated_at
FROM companies
WHERE slug = $1;

-- name: SyncCompaniesFromJobs :exec
-- Rebuild the companies catalogue from jobs. The companies table is derivable
-- from jobs (slug = company_slug, name = company), so after a slug-builder change
-- re-keys jobs, this re-keys companies to match. DISTINCT ON collapses a slug's
-- name variants; ON CONFLICT folds collisions and refreshes existing rows.
INSERT INTO companies (slug, name)
SELECT DISTINCT ON (company_slug) company_slug, company
FROM jobs
WHERE company_slug <> ''
ORDER BY company_slug
ON CONFLICT (slug) DO UPDATE SET
    name       = EXCLUDED.name,
    updated_at = now();

-- name: DeleteOrphanCompanies :execrows
-- Drop companies no longer referenced by any job — the stale rows left behind
-- when a slug-builder change re-keys jobs onto new slugs.
DELETE FROM companies c
WHERE NOT EXISTS (SELECT 1 FROM jobs j WHERE j.company_slug = c.slug);
