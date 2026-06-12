# Design — close-stale-jobs

## Context

Every ingest run fetches the **complete** current posting list of every configured
board (ATS list endpoints have no incremental API). Absence from a run therefore
means "no longer open on that board" — modulo transient board failures, which the
run already isolates per board (`stats.Failed`).

## Decisions

### Liveness column, not a deletions diff

`jobs.last_seen_at timestamptz NOT NULL DEFAULT now()`, set to `now()` in
`UpsertJob`'s `DO UPDATE` (and by default on insert). No per-run diffing, no run
table: the column is the entire liveness mechanism, written by the same single
atomic write path the pipeline already has.

### Time-threshold sweep, not per-board reconciliation

After a run, one statement closes open jobs not seen for 48h:

```sql
UPDATE jobs SET closed_at = now()
WHERE closed_at IS NULL AND last_seen_at < now() - interval '48 hours';
```

48h ≈ 8 crawl cycles at the 6h cadence, so a board that fails a few runs in a row
does not close its jobs. Per-board reconciliation ("close exactly what this run's
successful boards no longer list") would be more precise but needs a job→board
mapping and per-board run bookkeeping — complexity the threshold makes unnecessary.
The threshold is a constant in `cmd/ingest`; no config until a need appears.

Guard: the sweep runs only when the run ingested at least one job. A total outage
(DNS down, DB unreachable mid-run) then cannot mass-close the catalogue; the 48h
window absorbs the skipped sweep.

### Closed ≠ deleted; reopen is free

`closed_at timestamptz NULL` is a soft state, not a delete: the row keeps its
`public_slug` (stable public identity), its enrichment, and its `user_jobs`
references. If a posting reappears (board republished it), `UpsertJob` clears
`closed_at` — the upsert path is also the reopen path.

### Visibility split: lists hide, detail tells

List/search/company surfaces exclude closed jobs (`closed_at IS NULL` filters; the
search indexer skips closed jobs and reindex deletes their documents). The detail
endpoint (`GET /jobs/:slug`) still returns a closed job with `closed_at` in the
`jobview` shape: external links and a user's application history must not 404, and
the SPA needs the timestamp to render the closed state.

## Risks

- **Search index drift**: a job closed between reindex runs stays searchable until
  the next reindex (≤6h). Accepted — same staleness window the index already has
  for new jobs.
- **Existing rows**: the migration backfills `last_seen_at = now()` via the column
  default, so nothing closes until 48h after deploy — by then two days of real
  crawls have refreshed every live posting.
