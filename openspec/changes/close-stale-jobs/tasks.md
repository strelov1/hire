## 1. Schema + queries (liveness, sweep, filters)

- [ ] 1.1 New migration: `jobs.last_seen_at timestamptz NOT NULL DEFAULT now()`, `jobs.closed_at timestamptz`
- [ ] 1.2 `UpsertJob`: `DO UPDATE` also sets `last_seen_at = now(), closed_at = NULL`
- [ ] 1.3 New `CloseUnseenJobs` query (cutoff param, returns rows affected)
- [ ] 1.4 `closed_at IS NULL` filters: jobs list, company detail jobs, company `job_count`, reindex feed
- [ ] 1.5 `make sqlc`; integration tests (tagged) for upsert-refreshes/reopens, sweep closes/spares, list filters

## 2. Ingest sweep

- [ ] 2.1 `cmd/ingest`: after a run with `Ingested > 0`, call the sweep with the 48h cutoff; log closed count; unit test the guard logic

## 3. Search index excludes closed jobs

- [ ] 3.1 Indexer skips closed jobs; reindex removes documents for closed jobs; tests

## 4. API: closed_at in the job view

- [ ] 4.1 `jobview` carries nullable `closed_at`; detail endpoint serves closed jobs; handler test

## 5. Web: closed state on the job page

- [ ] 5.1 Job page renders the closed notice and hides Apply when `closed_at` is set; `svelte-check` clean

## 6. Rollout

- [ ] 6.1 Apply the ALTER manually on prod (migration-runner seam stays open), deploy, verify a synthetic closed job disappears from list/search but resolves on detail
