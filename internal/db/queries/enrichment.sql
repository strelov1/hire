-- name: EnqueuePendingJobs :execrows
-- Idempotent backfill: enqueue every job that is unenriched or below the target
-- schema version. ON CONFLICT keeps exactly one entry per (job_id, target_version),
-- so running this every command invocation never duplicates work.
INSERT INTO enrichment_outbox (job_id, target_version)
SELECT id, sqlc.arg(target_version)::int
FROM jobs
WHERE enriched_at IS NULL OR enrichment_version < sqlc.arg(target_version)::int
ON CONFLICT (job_id, target_version) DO NOTHING;

-- name: ClaimEnrichmentBatch :many
-- Claim a batch of live, unleased entries by stamping claimed_at. SKIP LOCKED lets
-- concurrent workers take disjoint rows; the lease predicate reclaims entries whose
-- worker died (stale claimed_at), so no separate reaper process is needed.
WITH claimable AS (
    SELECT id
    FROM enrichment_outbox
    WHERE failed_at IS NULL
      AND (claimed_at IS NULL
           OR claimed_at < now() - make_interval(secs => sqlc.arg(lease_seconds)::int))
    ORDER BY id
    FOR UPDATE SKIP LOCKED
    LIMIT sqlc.arg(batch_size)
)
UPDATE enrichment_outbox o
SET claimed_at = now()
FROM claimable c
WHERE o.id = c.id
RETURNING o.id, o.job_id, o.target_version;

-- name: DeleteEnrichmentEntry :exec
DELETE FROM enrichment_outbox
WHERE id = $1;

-- name: RecordEnrichmentFailure :one
-- Count a failed attempt: bump attempts, record the error, and dead-letter (set
-- failed_at) once attempts reach the max. The lease (claimed_at) is intentionally
-- left in place — its expiry gates the retry to a later run and doubles as the
-- crash reaper, so a failed entry is never reprocessed within the same run.
UPDATE enrichment_outbox
SET attempts   = attempts + 1,
    last_error = sqlc.arg(last_error),
    failed_at  = CASE
                     WHEN attempts + 1 >= sqlc.arg(max_attempts)::int THEN now()
                     ELSE NULL
                 END
WHERE id = sqlc.arg(id)
RETURNING attempts, failed_at;
