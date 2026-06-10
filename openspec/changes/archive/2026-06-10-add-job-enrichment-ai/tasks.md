## 1. Migration and queries

- [x] 1.1 Add `migrations/0004_enrichment_outbox.sql`: create `enrichment_outbox` (`id`, `job_id` FK → `jobs(id)` ON DELETE CASCADE, `target_version`, `attempts` default 0, `claimed_at`, `failed_at`, `last_error`, `created_at`, `UNIQUE (job_id, target_version)`) + a partial index `WHERE failed_at IS NULL` for the claim query
- [x] 1.2 Add `internal/db/queries/enrichment.sql`: `EnqueuePendingJobs(target_version)` (INSERT…SELECT from jobs where pending, ON CONFLICT DO NOTHING), `ClaimEnrichmentBatch(batch, lease_seconds)` (CTE + `FOR UPDATE SKIP LOCKED`, stamp `claimed_at`, RETURNING id/job_id/target_version/attempts), `RecordEnrichmentFailure(id, last_error, max_attempts)` (attempts++, clear `claimed_at`, set `failed_at` when attempts reach max), `DeleteEnrichmentEntry(id)`
- [x] 1.3 Add `SetJobEnrichment(id, enrichment, enriched_at, enrichment_version)` to `internal/db/queries/jobs.sql` — update only those columns (+ `updated_at = now()`) by id; touch no raw source field
- [x] 1.4 Run `make sqlc`, commit regenerated `internal/db` code; confirm `go build ./...` passes

## 2. Enrichment contract additions

- [x] 2.1 Add `const Version = 1` to `internal/enrich` (stamped on write; used as `target_version`)
- [x] 2.2 Define `JobInput` (title, company, location, remote, description) and the `Provider` interface (`Enrich(ctx, JobInput) (Enrichment, error)`) in `internal/enrich`

## 3. LangChainGo provider

- [x] 3.1 Add `github.com/tmc/langchaingo` to `go.mod` (`go get`); commit
- [x] 3.2 Implement a LangChainGo-backed `Provider` using the OpenAI-compatible client with `openai.WithBaseURL(LLM_BASE_URL)` — build the prompt from the controlled vocabularies, request a structured `Enrichment`, read base URL + API key + model from config
- [x] 3.3 Map the structured result into the `Enrichment` struct; omit fields the model left unset rather than zero-filling

## 4. Config

- [x] 4.1 Add `LLM_BASE_URL`, `LLM_API_KEY`, `LLM_MODEL` to `internal/config` (required; fail with a clear error if any is unset). Add tunables with conservative defaults: batch size, lease seconds, max attempts

## 5. Enrichment command

- [x] 5.1 Create `cmd/enrich/main.go`: load config, open the pgx pool, construct the provider
- [x] 5.2 Enqueue step: call `EnqueuePendingJobs(enrich.Version)` (idempotent backfill of pending jobs)
- [x] 5.3 Drain loop: `ClaimEnrichmentBatch(batch, lease)`; for each entry `GetJob(job_id)` → `Provider.Enrich`
- [x] 5.4 Validate each result with `Enrichment.Validate`; retry once on failure
- [x] 5.5 On success: one transaction — `SetJobEnrichment(id, payload, now, target_version)` + `DeleteEnrichmentEntry(outbox_id)`
- [x] 5.6 On failure (invalid twice or LLM error): `RecordEnrichmentFailure(outbox_id, err, maxAttempts)`; never write an invalid payload
- [x] 5.7 One entry's error must not abort the run; print enriched / failed / dead-lettered counts on exit

## 6. Tests

- [x] 6.1 Add a fake `Provider` and unit-test the command loop: valid → written + entry deleted, invalid-twice → failure recorded (no write), provider error → failure recorded, counts reported
- [x] 6.2 Test the queue semantics against a test DB (testcontainers-go): enqueue idempotency (no duplicate per `(job_id, target_version)`), claim skips dead-lettered + leased entries, stale lease is reclaimable, attempts reach max → `failed_at` set — run with `go test -tags=integration ./internal/db/` (requires Docker)
- [x] 6.3 Unit-test the provider's prompt/response mapping with a stubbed LLM response (vocabulary-constrained fields map correctly; unstated fields omitted)

## 7. Verification

- [x] 7.1 `go build ./... && go vet ./...` pass; full unit suite green; integration suite green via `-tags=integration`
- [x] 7.2 Verified end-to-end against a live OpenAI-compatible endpoint with a seeded local DB: run 1 enriched the job (valid vocabulary payload, `enriched_at` set, `enrichment_version = 1`, raw fields unchanged, outbox entry deleted); run 2 was a no-op (`enriched=0`, version still 1)
