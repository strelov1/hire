## Why

Phase 1 defined the `job-enrichment` schema — a typed `Enrichment` contract and a
`jobs.enrichment` JSONB column with provenance columns (`enriched_at`,
`enrichment_version`) — but nothing populates it. Every job's enrichment is still
`{}`. The filterable facets a job seeker needs (seniority, work mode, skills,
salary, location) sit locked inside `description` prose. This change adds the AI
layer that reads that prose and fills the contract, driven by a durable enrichment
queue so the work can later be offloaded to an external queue without reshaping the
data model.

## What Changes

- Add a durable work queue, `enrichment_outbox` (new migration), holding one row per
  `(job_id, target_version)` that needs enriching. It is a **reference** queue
  (job id + version + bookkeeping), not a copy of the job — `jobs` stays canonical.
- Add the queue's producer paths: an idempotent backfill (`EnqueuePendingJobs`) that
  enqueues `jobs` rows below the current schema version now, and the seam for a future
  ingest path to insert into the outbox in the **same transaction** as the job upsert
  (transactional outbox).
- Add a safe claim path: workers claim a batch with `FOR UPDATE SKIP LOCKED`, stamping
  `claimed_at`; a lease makes a crashed/stalled claim reclaimable (built-in reaper),
  so concurrent workers never process the same row twice.
- Add `internal/enrich` extraction: a `Provider` interface plus a LangChainGo-backed
  implementation (OpenAI-compatible client with a configurable base URL) that, given a
  job, asks an LLM to populate an `Enrichment` against the controlled vocabularies.
- Add validated write-back: each result is checked with `Enrichment.Validate`; on
  success the payload + provenance stamp is written to `jobs` and the outbox row is
  **deleted**; on failure `attempts` is incremented and, after a max, the row is
  **dead-lettered** (`failed_at` set) so it is not retried forever.
- Add `cmd/enrich`: a standalone command (externally scheduled) that enqueues pending
  jobs, drains a claimed batch, and reports enriched / failed / dead-lettered counts.
- Add provider-agnostic LLM config: `LLM_BASE_URL`, `LLM_API_KEY`, `LLM_MODEL` — any
  OpenAI-compatible endpoint (a LiteLLM gateway, Chinese model providers, etc.).

## Capabilities

### New Capabilities

- `ai-enrichment`: populating a job's enrichment from its description via an LLM,
  organized around a durable outbox queue — enqueue (backfill now, ingest later),
  lease-based claim, validated write-back into `jobs`, dead-lettering of repeated
  failures, and the batch command that runs it.

### Modified Capabilities

<!-- None. Phase-1 `job-enrichment` storage and contract requirements are unchanged;
     this change adds the queue and the layer that populates the payload. -->

## Impact

- **New code**: `internal/enrich` (provider + extraction; the existing contract file
  stays), `cmd/enrich` (batch command), `internal/db/queries/enrichment.sql` + a
  `SetJobEnrichment` query in `jobs.sql` (regenerated via `make sqlc`).
- **DB**: new migration `migrations/0004_enrichment_outbox.sql`. Applies via Postgres
  initdb only (no runner yet) — dev needs `docker compose down -v && make up`.
- **Config**: `internal/config` gains `LLM_BASE_URL`, `LLM_API_KEY`, `LLM_MODEL`.
- **Dependencies**: adds `github.com/tmc/langchaingo` (OpenAI-compatible client).
- **Runtime/cost**: introduces outbound LLM calls; cost scales with queue depth —
  bounded by the cheap configurable model, dead-lettering, and once-per-version
  enqueue.
- **API**: none. The read API already exposes `enrichment` from phase 1.
- **Seam**: the outbox is the handoff point for later offloading enrich execution to
  an external queue (SQS/Rabbit) without changing the data model.
