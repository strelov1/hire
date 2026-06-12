-- name: ListJobs :many
-- Newest-added first: created_at is when the job entered the catalogue (stable
-- across re-ingests), so fresh ingests surface on top regardless of how old the
-- platform's posted_at is. id breaks ties within one ingest batch.
SELECT *
FROM jobs
WHERE closed_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: ListJobsByIDAfter :many
-- Keyset scan for the reindex command: pages by the immutable primary key, so
-- concurrent inserts/updates (which shift posted_at ordering) cannot make the
-- scan skip or repeat rows the way OFFSET pagination would.
SELECT *
FROM jobs
WHERE id > sqlc.arg(after_id)
ORDER BY id
LIMIT sqlc.arg(batch_size);

-- name: GetJob :one
SELECT *
FROM jobs
WHERE id = $1;

-- name: GetJobBySlug :one
SELECT *
FROM jobs
WHERE public_slug = $1;

-- name: GetJobIDBySlug :one
-- Slim slug->id lookup for the view/apply interaction path, which needs only the
-- internal id (the user_jobs FK) and must not drag the wide description/enrichment
-- columns over the wire on every silent view. GetJobBySlug (SELECT *) stays for the
-- public detail handler that renders the whole row.
SELECT id
FROM jobs
WHERE public_slug = $1;

-- name: CountJobs :one
SELECT count(*)
FROM jobs
WHERE closed_at IS NULL;

-- name: ListJobsByCompany :many
SELECT *
FROM jobs
WHERE company_slug = $1 AND closed_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;

-- name: UpsertJob :one
-- Single atomic write: upsert the company (only when the slug is non-empty,
-- via the WHERE on the SELECT) and the job together, keeping the "one write =
-- one job" property of the pipeline's write path.
-- The enrichment columns are deliberately NOT written here: ingest carries no
-- enrichment, so a new row takes the table defaults ('{}' / NULL / 0) and a
-- re-ingest leaves any existing enrichment untouched. SetJobEnrichment (the
-- enrichment worker) is the sole writer of those columns.
WITH company_upsert AS (
    INSERT INTO companies (slug, name)
    SELECT sqlc.arg(company_slug), sqlc.arg(company)
    WHERE sqlc.arg(company_slug) <> ''
    ON CONFLICT (slug) DO UPDATE SET
        name       = EXCLUDED.name,
        updated_at = now()
)
INSERT INTO jobs (
    source, external_id, url, title, company, company_slug, location, remote, description, posted_at,
    public_slug
) VALUES (
    sqlc.arg(source), sqlc.arg(external_id), sqlc.arg(url), sqlc.arg(title),
    sqlc.arg(company), sqlc.arg(company_slug), sqlc.arg(location), sqlc.arg(remote),
    sqlc.arg(description), sqlc.arg(posted_at),
    sqlc.arg(public_slug)
)
-- public_slug is deliberately NOT in the DO UPDATE SET: the slug is minted once
-- at insert and is the row's stable public identity. Re-ingest of the same
-- (source, external_id) must not rewrite it, so external links stay valid even
-- if the slug builder changes later (that would be a deliberate migration).
ON CONFLICT (source, external_id) DO UPDATE SET
    url          = EXCLUDED.url,
    title        = EXCLUDED.title,
    company      = EXCLUDED.company,
    company_slug = EXCLUDED.company_slug,
    location     = EXCLUDED.location,
    remote       = EXCLUDED.remote,
    description  = EXCLUDED.description,
    posted_at    = EXCLUDED.posted_at,
    -- The crawl saw the posting: refresh liveness and reopen if it was closed.
    last_seen_at = now(),
    closed_at    = NULL,
    updated_at   = now()
RETURNING *;

-- name: CloseUnseenJobs :execrows
-- Post-ingest sweep (see job-lifecycle spec): close every open job not seen since
-- the cutoff. The caller owns the grace window (cutoff = now() - window) and the
-- "run ingested something" guard, so a failed crawl never mass-closes the catalogue.
UPDATE jobs
SET closed_at  = now(),
    updated_at = now()
WHERE closed_at IS NULL
  AND last_seen_at < sqlc.arg(cutoff);

-- name: UpdateJobSlugs :exec
-- One-off backfill for a deliberate slug-builder change (see the UpsertJob note on
-- why slugs are otherwise immutable). public_slug/company_slug are deterministic
-- from the row's immutable fields, so recomputing and rewriting them is idempotent.
UPDATE jobs
SET public_slug  = sqlc.arg(public_slug),
    company_slug = sqlc.arg(company_slug)
WHERE id = sqlc.arg(id);

-- name: EnqueueJobEnrichment :execrows
-- Transactional-outbox enqueue for the ingest write path: queue this one job for
-- enrichment, gated on the same condition the backfill uses (unenriched or below the
-- target schema version), so an already-enriched job is not re-queued. Idempotent via
-- the outbox's UNIQUE (job_id, target_version). Run in the same transaction as the
-- job's UpsertJob so a newly ingested job is queued atomically with its write.
INSERT INTO enrichment_outbox (job_id, target_version)
SELECT id, sqlc.arg(target_version)::int
FROM jobs
WHERE id = sqlc.arg(job_id)::bigint
  AND (enriched_at IS NULL OR enrichment_version < sqlc.arg(target_version)::int)
ON CONFLICT (job_id, target_version) DO NOTHING;

-- name: SetJobEnrichment :exec
-- Targeted enrichment write used by the enrichment command: set only the payload
-- and the provenance stamp, touching no raw source field. Kept separate from
-- UpsertJob (the ingest full-upsert path) so ingest and enrichment stay decoupled.
UPDATE jobs
SET enrichment         = sqlc.arg(enrichment),
    enriched_at        = sqlc.arg(enriched_at),
    enrichment_version = sqlc.arg(enrichment_version),
    updated_at         = now()
WHERE id = sqlc.arg(id);
