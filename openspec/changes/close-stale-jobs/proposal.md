## Why

Ingest only ever adds or updates jobs. A posting removed from its ATS board stays in
the catalogue forever and keeps surfacing in lists and search, linking to a dead
"position closed" page at the employer. Every full crawl already sees the complete
set of currently-open postings per board, so liveness is free information we are
throwing away — we just need to record it and act on it.

## What Changes

- Track liveness: `jobs.last_seen_at` is touched on every upsert (the crawl saw the
  posting); a job seen again after being closed reopens.
- Close stale jobs: after a successful ingest run, a sweep stamps `closed_at` on
  open jobs not seen within a 48h grace window (~8 missed runs — tolerant of
  transient board failures). The sweep is skipped when the run ingested nothing,
  so a total crawl outage can never mass-close the catalogue.
- Read paths show only open jobs: the jobs list, company detail jobs, company
  `job_count`, and the search index exclude closed jobs. The job detail page still
  serves a closed job (stable public identity, user application history, SEO) and
  exposes `closed_at` so the SPA renders a "no longer accepting applications"
  state instead of an Apply button.

## Capabilities

### New Capabilities

- `job-lifecycle`: liveness tracking, the closing sweep, reopening, and
  closed-job visibility semantics.

### Modified Capabilities

- `job-search`: the index contains only open jobs; reindex removes closed ones.
- `companies`: company detail jobs and `job_count` count only open jobs.
- `web-frontend`: the job page renders a closed state instead of the Apply action.

## Impact

- **Schema**: `ALTER TABLE jobs ADD last_seen_at timestamptz NOT NULL DEFAULT now(),
  ADD closed_at timestamptz` (new migration file; prod gets a manual apply — the
  versioned-migration-runner seam from AGENT.md remains open).
- **Code**: `UpsertJob` delta + `CloseUnseenJobs` sweep query + `closed_at IS NULL`
  filters in list/company/count queries (`internal/db/queries`, regenerated sqlc);
  sweep call in `cmd/ingest`; index filtering in `internal/search` + `cmd/reindex`;
  `closed_at` in `internal/jobview`; closed state in `web/`.
- **Out of scope**: a versioned migration runner; per-board sweep precision
  (the time threshold covers it); closing via ATS "job closed" detail probes.
