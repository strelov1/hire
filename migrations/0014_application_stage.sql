-- Application stage + notes on the per-(user, job) interaction. `stage` tracks
-- where an application stands — a controlled vocabulary validated in Go
-- (applied/screening/responded/interview/offer + accepted/rejected/withdrawn);
-- NULL = not in the pipeline. `notes` is free text; NULL = none. Both live on the
-- same one-row-per-(user, job) interaction as viewed_at/saved_at/applied_at.
-- Applied automatically by Postgres on first volume init (same as 0001) and also
-- serves as schema source for sqlc. Existing volumes/prod need a manual apply
-- (the versioned-migration-runner seam from AGENT.md remains open).

ALTER TABLE user_jobs ADD COLUMN IF NOT EXISTS stage TEXT;
ALTER TABLE user_jobs ADD COLUMN IF NOT EXISTS notes TEXT;
