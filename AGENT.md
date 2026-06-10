# AGENT.md

Guidance for AI agents working in this repository.

## Working principles

Non-negotiable. Bias toward caution over speed; use judgment on trivial tasks.

- **Think before coding.** Surface assumptions. If multiple interpretations exist, present them — don't pick silently. If something is unclear, ask.
- **Simplicity first.** Minimum code that solves the problem. No features, abstractions, or error handling that wasn't asked for. Prefer a library's intended API over a clever shim.
- **Surgical changes.** Touch only what the task requires; don't refactor unbroken things or rework formatting. Match existing style. Clean up what your change orphaned; leave pre-existing dead code alone. Exception: do the real refactor when a clean change genuinely requires reshaping existing code.
- **Fix root causes, not symptoms.**
- **No overengineering, and no MVP shortcuts.** Hold the middle path: don't build infrastructure before there's a concrete need (note the seam for later instead), and don't ship quick-and-dirty or "for now" hacks. Build each feature correctly and idiomatically — neither gold-plated nor a placeholder.
- **MVP stage — keep the architecture fluid.** The project is early/MVP; the current structure is not load-bearing legacy. When a new feature doesn't fit the existing architecture cleanly, prefer reshaping or refactoring the affected part over bolting on an awkward special case — re-architect freely to keep the design clean rather than accumulating legacy. This complements "no MVP shortcuts" (still build each feature correctly) and extends "surgical changes" (existing structure is not frozen when a clean fit needs it reshaped).
- **English only.** All code, comments, identifiers, docs, and commits are in English.

## What this is

`hire` is an open-source IT job aggregator backend. Intended shape: many source parsers feed a pipeline that normalizes jobs into one schema, deduplicates them, and enriches them with AI; served over an HTTP API with rich filters.

**Current state: early backend.** Fiber HTTP server with `/health`, `/api/v1/jobs[/:id]`, companies endpoints, and a `/api/v1/auth` surface (register/login/me with stateless JWT); Postgres via sqlc with `jobs`, `companies`, and `users` tables; a typed, versioned enrichment schema on `jobs`; and a standalone AI enrichment worker (`cmd/enrich`) that fills it from job descriptions via a durable outbox queue. **Source parsers and the ingest pipeline do not exist yet** — the enrichment worker currently backfills from existing rows. A Svelte SPA lives under `web/` and consumes the API (including layout-level auth).

Stack: **Go + Fiber v2**, **PostgreSQL**, **sqlc** (generated DB access, no ORM), **Docker Compose**, **langchaingo** (LLM access over any OpenAI-compatible endpoint — provider-agnostic, no vendor baked in).

## Layout

```
cmd/server/main.go   entry point: Fiber startup + graceful shutdown
cmd/enrich/main.go   standalone AI enrichment worker (run on a schedule; drains the outbox queue)
internal/
  config/            env config (server: PORT, DATABASE_URL, FRONTEND_ORIGIN, JWT_SECRET, JWT_TTL; enrich: LLM_BASE_URL, LLM_API_KEY, LLM_MODEL)
  database/          pgxpool connection pool
  db/                GENERATED sqlc code (do not edit) + queries/*.sql (hand-written)
  handler/           HTTP handlers (Handler struct + Register wires routes); auth.go holds register/login/me
  auth/              security primitives: bcrypt password hashing, JWT Issuer (issue/verify), and the RequireAuth Fiber middleware
  enrich/            enrichment contract (typed Enrichment + controlled vocabularies), the LLM Provider abstraction, and the queue-draining Runner
  normalize/         slug normalization
migrations/          SQL schema — single source for BOTH sqlc and Postgres initdb
```

Future features slot in here without restructuring: `internal/sources/` (parsers as interface + registry) and `internal/pipeline/` (fetch → normalize → dedup → upsert). `UpsertJob` already exists as the pipeline's write path; when ingest lands it should also enqueue into `enrichment_outbox` in the same transaction (transactional-outbox seam — see the enrichment convention below).

## Commands

```bash
make up                      # build + start app and postgres in Docker
HIRE_HOST_PORT=8090 make up  # use another host port if 8080 is taken
make down / make logs        # stop containers / tail app logs
make run                     # run server on host (needs a running Postgres)
make psql                    # psql into the DB container
make sqlc                    # regenerate internal/db from queries/migrations (via Docker)
go build ./...  &&  go vet ./...
go test ./...                              # unit tests (no external deps)
go test -tags=integration ./internal/db/  # queue integration tests (needs Docker; uses testcontainers)
go run ./cmd/enrich          # enrichment worker — needs DATABASE_URL + LLM_BASE_URL/LLM_API_KEY/LLM_MODEL
```

## Conventions and gotchas

- **sqlc is the only DB layer.** `internal/db/*.go` is generated — never edit by hand (committed so the repo builds without sqlc installed). To change DB access, edit `internal/db/queries/*.sql` (or `migrations/` for schema), run `make sqlc` (runs sqlc via Docker), and commit the result. Handlers use `*db.Queries`, built once in `handler.Register`.
- **Migrations apply via Postgres initdb.** `migrations/` is mounted into `/docker-entrypoint-initdb.d`, so Postgres runs each `*.sql` **once, on first volume init only**. Changing a migration does NOT re-apply to an existing volume — recreate it with `docker compose down -v && make up`. The same dir is sqlc's schema source, keeping schema and code in sync. *Known seam:* no versioned migration runner yet; needed before the first schema change ships to a persistent DB.
- **Response shapes.** Lists: `{"data": ..., "meta": {...}}`; single items: `{"data": ...}`. Errors use `fiber.NewError(status, msg)` — no central `ErrorHandler` yet (deferred on purpose; don't hand-roll per-handler error JSON).
- **Dedup key.** `jobs.UNIQUE (source, external_id)` is the dedup key; `UpsertJob` is `ON CONFLICT` on it.
- **Auth is stateless JWT, provider-agnostic for the future.** `internal/auth` owns the primitives (bcrypt hashing, an HS256 `Issuer`, the `RequireAuth` middleware that puts the user id in `c.Locals`); handlers stay thin. The token carries only the user id (`sub`), so it survives new sign-in methods unchanged. `users.password_hash` is **nullable** on purpose — passwordless sign-in (Google/magic-link, future) creates accounts with no password, and password login rejects a null hash with the same generic `401`. `email` is the canonical account key (`UNIQUE (lower(email))`); future providers link to it via an additive `user_identities` table (seam, not built). `JWT_SECRET` is required at server startup (fail-fast in `cmd/server`); the enrich worker shares `config.Load` but ignores it. *Known seams:* no token revocation/refresh (modest TTL instead) and no login rate-limit yet. Bump nothing to "re-auth" — tokens just expire.
- **Enrichment is queue-driven and provider-agnostic.** The typed `Enrichment` contract + controlled vocabularies in `internal/enrich` are the schema's source of truth (stored in `jobs.enrichment` JSONB; provenance in `enriched_at`/`enrichment_version`; bump `enrich.Version` to re-enrich). Work flows through `enrichment_outbox` — a reference-only queue (`job_id` + `target_version` + lease/retry bookkeeping), not a copy of the job; `jobs` stays canonical. `cmd/enrich` enqueues pending rows, claims a batch with `FOR UPDATE SKIP LOCKED` + a `claimed_at` lease (the lease expiry is the built-in reaper — no separate process), enriches via the `Provider` (LLM behind an interface; swap the impl, don't couple callers), validates with `Enrichment.Validate` (retry once, then dead-letter — never persist an out-of-vocabulary payload), and on success writes via `SetJobEnrichment` + deletes the outbox row in one transaction. `SetJobEnrichment` is deliberately separate from `UpsertJob` so ingest and enrichment stay decoupled. The LLM is configured by `LLM_BASE_URL`/`LLM_API_KEY`/`LLM_MODEL` (any OpenAI-compatible endpoint) — never hard-code a vendor or model.
