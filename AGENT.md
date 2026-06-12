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

`freehire` ([freehire.dev](https://freehire.dev)) is an open-source IT job aggregator backend. Intended shape: many source parsers feed a pipeline that normalizes jobs into one schema, deduplicates them, and enriches them with AI; served over an HTTP API with rich filters.

**Current state: early backend.** Fiber HTTP server with `/health`, `/api/v1/jobs[/:id]`, companies endpoints, a `/api/v1/auth` surface (register/login/me with stateless JWT), and per-user job-interaction endpoints (`POST /api/v1/jobs/:id/view` and `.../apply`, behind auth); Postgres via sqlc with `jobs`, `companies`, `users`, and `user_jobs` tables; a typed, versioned enrichment schema on `jobs`; a standalone AI enrichment worker (`cmd/enrich`) that fills it from job descriptions via a durable outbox queue; and a **standalone source-ingest worker (`cmd/ingest`)** that crawls the ATS boards listed in `sources.yml` (greenhouse/lever/ashby adapters), normalizes the postings, and upserts them — enqueuing new ones for enrichment in the same write. A Svelte SPA lives under `web/` and consumes the API (including layout-level auth).

Stack: **Go + Fiber v2**, **PostgreSQL**, **sqlc** (generated DB access, no ORM), **Docker Compose**, **langchaingo** (LLM access over any OpenAI-compatible endpoint — provider-agnostic, no vendor baked in).

## Layout

```
cmd/server/main.go   entry point: Fiber startup + graceful shutdown
cmd/enrich/main.go   standalone AI enrichment worker (run on a schedule; drains the outbox queue)
cmd/ingest/main.go   standalone source-ingest worker (run on a schedule; crawls every board in sources.yml once and exits)
sources.yml          boards to crawl: each entry = company + provider (registered adapter) + platform board id
internal/
  config/            env config (server: PORT, DATABASE_URL, FRONTEND_ORIGIN, JWT_SECRET, JWT_TTL, COOKIE_SECURE, OAUTH_<PROVIDER>_CLIENT_ID/_CLIENT_SECRET; enrich: LLM_BASE_URL, LLM_API_KEY, LLM_MODEL)
  database/          pgxpool connection pool
  db/                GENERATED sqlc code (do not edit) + queries/*.sql (hand-written)
  handler/           HTTP handlers (Handler struct + Register wires routes); auth.go holds register/login/logout/me; user_jobs.go holds the view/apply interaction endpoints; errors.go is the central ErrorHandler
  auth/oauth/        OAuth sign-in: Provider interface + google/github/linkedin implementations, the config-driven registry, and the CSRF state cookie
  jobview/           the single public wire shape of a job (shared by list/detail/company/search responses and the search index)
  auth/              security primitives: bcrypt password hashing, JWT Issuer (issue/verify), the httpOnly cookie transport, and the RequireAuth Fiber middleware
  enrich/            enrichment contract (typed Enrichment + controlled vocabularies), the LLM Provider abstraction, and the queue-draining Runner
  sources/           source adapters as interface + registry (greenhouse, lever, ashby), the shared HTTP client, and sources.yml parsing/validation
  pipeline/          ingest Runner (fetch → normalize → dedup → upsert) over the source registry
  search/            Meilisearch-backed job search (document shape + indexing)
  normalize/         slug normalization
migrations/          SQL schema — single source for BOTH sqlc and Postgres initdb
```

Adding a company is one entry in `sources.yml`; adding an ATS platform is a new adapter in `internal/sources` plus one line in `sources.All`. The ingest pipeline's write path is `UpsertJob`, which also enqueues into `enrichment_outbox` in the same transaction (transactional-outbox — see the enrichment convention below). Future ingest features (more providers, scheduling) slot in without restructuring.

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
go run ./cmd/ingest          # source-ingest worker — crawls sources.yml (override path via SOURCES_FILE); needs DATABASE_URL
```

## Conventions and gotchas

- **sqlc is the only DB layer.** `internal/db/*.go` is generated — never edit by hand (committed so the repo builds without sqlc installed). To change DB access, edit `internal/db/queries/*.sql` (or `migrations/` for schema), run `make sqlc` (runs sqlc via Docker), and commit the result. Handlers use `*db.Queries`, built once in `handler.Register`.
- **Migrations apply via Postgres initdb.** `migrations/` is mounted into `/docker-entrypoint-initdb.d`, so Postgres runs each `*.sql` **once, on first volume init only**. Changing a migration does NOT re-apply to an existing volume — recreate it with `docker compose down -v && make up`. The same dir is sqlc's schema source, keeping schema and code in sync. *Known seam:* no versioned migration runner yet; needed before the first schema change ships to a persistent DB.
- **Response shapes.** Lists: `{"data": ..., "meta": {...}}`; single items: `{"data": ...}`; errors: `{"error": msg}`. Handlers signal failure by returning an error — `fiber.NewError(status, msg)` to set a specific status/message, or a bare error (e.g. `pgx.ErrNoRows`) for the common cases. The central `handler.ErrorHandler` (wired in `cmd/server` via `fiber.Config{ErrorHandler}`) renders the JSON envelope and maps `*fiber.Error`→its code, `pgx.ErrNoRows`→404, a foreign-key violation (SQLSTATE 23503, a write referencing a missing parent row)→404, everything else→500. Don't hand-roll per-handler error JSON; don't re-map `ErrNoRows` in read handlers (just `return err`). Genuinely domain-specific status choices (e.g. `Me` returning 401, not 404, for a token whose user is gone) stay in the handler.
- **Dedup key.** `jobs.UNIQUE (source, external_id)` is the dedup key; `UpsertJob` is `ON CONFLICT` on it.
- **Auth is stateless JWT in an httpOnly cookie, same-origin, provider-agnostic for the future.** `internal/auth` owns the primitives (bcrypt hashing, an HS256 `Issuer`, the `CookieName`/`SetTokenCookie`/`ClearTokenCookie` transport, and the `RequireAuth` middleware that reads the cookie and puts the user id in `c.Locals`); handlers stay thin (`register`/`login` set the cookie + return `{"data": user}`, `logout` clears it, `me` is guarded). The JWT carries only the user id (`sub`), so it survives both new sign-in methods and a later swap to opaque sessions. **Transport is an `HttpOnly; SameSite=Lax` cookie, never a `Bearer` header or `localStorage`** — the SPA can't read it (XSS-safe) and the browser attaches it automatically. This relies on the SPA and API being **same-origin**: in dev the Vite proxy (`web/vite.config.ts`) forwards `/api` to the backend; `SameSite=Lax` + same-origin is the CSRF defense (no CSRF token needed yet). `users.password_hash` is **nullable** on purpose — passwordless sign-in (Google/magic-link, future) creates accounts with no password, and password login rejects a null hash with the same generic `401`. `email` is the canonical account key (`UNIQUE (lower(email))`); external providers link to it via the `user_identities` table (see the OAuth convention below). `JWT_SECRET` is required at server startup (fail-fast in `cmd/server`; the enrich worker shares `config.Load` but ignores it); `COOKIE_SECURE=true` for HTTPS (default false for http://localhost dev). *Known seams:* no token revocation/refresh (logout clears the cookie but the JWT lives until `exp`; modest TTL instead), no login rate-limit, and a CSRF token only if a future need forces `SameSite=None`. Bump nothing to "re-auth" — tokens just expire.
- **OAuth sign-in is a provider registry over the same cookie session.** Google/GitHub/LinkedIn sign-in uses the server-side authorization-code flow: `GET /api/v1/auth/oauth/:provider/start` sets a 10-minute httpOnly CSRF `state` cookie and redirects to the provider; `.../callback` verifies the state, exchanges the code, fetches the identity (id + **verified** email), resolves the account, sets the same JWT session cookie as password login, and 302s back to the SPA (failures 302 with `?auth_error=oauth`, never JSON — details go to the server log). `internal/auth/oauth` owns the `Provider` interface (Google/LinkedIn share an OIDC-userinfo implementation; GitHub reads `/user` + `/user/emails`), the registry (`NewRegistry`), and the state cookie; handlers live in `internal/handler/oauth.go`. Identities are keyed `user_identities (provider, provider_user_id) → user_id`; resolution is identity-first (a later provider-email change never re-keys the account), then a **verified-email** link to the existing account, then a new passwordless user (`password_hash` NULL) — the last two in one transaction. **Never link or create by an unverified email** (account-takeover vector). Config: `OAUTH_<PROVIDER>_CLIENT_ID`/`_CLIENT_SECRET` (GOOGLE/GITHUB/LINKEDIN); a provider is enabled only when both are set, and `GET /api/v1/auth/oauth/providers` lists the enabled ones (the SPA renders buttons from it). Redirect URLs derive from `FRONTEND_ORIGIN` (`<origin>/api/v1/auth/oauth/<p>/callback` is what you register at each provider). Provider tokens are used once to fetch the identity and never stored. *Known seams:* identity unlinking/management UI, magic-link sign-in.
- **Per-user job tracking is one row per (user, job).** `user_jobs (user_id, job_id, viewed_at, applied_at, PRIMARY KEY (user_id, job_id))` records a user's interaction with a job; the composite PK is the dedup key (the invariant "at most one interaction — and one application — per (user, job)"). Both writes are idempotent upserts behind `RequireAuth`: `RecordJobView` (touches `viewed_at`, "most-recent view") and `MarkJobApplied` (sets `applied_at`). View history = all rows; applications = `applied_at IS NOT NULL`. Handlers (`user_jobs.go`) return `{"data": interaction}` with `user_id` omitted; public job reads stay unauthenticated. The SPA records a view silently when a signed-in user opens a job (failure is swallowed — it must not break the page), shows a "You applied" badge, and offers a "Did you apply?" prompt after the Apply click (Yes → `markJobApplied`; No writes nothing). *Known seam (designed, not built):* a `stage` column (Applied→…→Offer) keyed off `applied_at` for an application pipeline, plus an "Applications" listing page and list-row badges.
- **Source ingest is a declarative board list behind a provider registry.** `sources.yml` lists boards (`company` + `provider` + `board`); `sources.All` maps each `provider` string to a registered adapter (greenhouse/lever/ashby), all speaking the `Source` interface over a shared HTTP client. `cmd/ingest` loads the config, **validates every entry against the registry and fails fast** (a misconfigured board never starts a run), then the `pipeline.Runner` fetches each board once, normalizes postings, and `UpsertJob`s them (idempotent on the dedup key, so re-running is safe). It's a run-once-and-exit worker meant for cron — no long-lived process. Adapters are read-only over public ATS JSON APIs; the per-board crawl is independent, so one failing board is counted (`stats.Failed`) but does not abort the rest.
- **Enrichment is queue-driven and provider-agnostic.** The typed `Enrichment` contract + controlled vocabularies in `internal/enrich` are the schema's source of truth (stored in `jobs.enrichment` JSONB; provenance in `enriched_at`/`enrichment_version`; bump `enrich.Version` to re-enrich). Work flows through `enrichment_outbox` — a reference-only queue (`job_id` + `target_version` + lease/retry bookkeeping), not a copy of the job; `jobs` stays canonical. `cmd/enrich` enqueues pending rows, claims a batch with `FOR UPDATE SKIP LOCKED` + a `claimed_at` lease (the lease expiry is the built-in reaper — no separate process), enriches via the `Provider` (LLM behind an interface; swap the impl, don't couple callers) under a per-call timeout so a stalled gateway can't hang the worker, `Enrichment.Sanitize`s out-of-vocabulary enum values (drops the stray field rather than dead-lettering the whole job — the invariant is still "never persist an out-of-vocabulary value") then `Validate`s as a guard (an LLM/parse error retries once, then dead-letters), and on success writes via `SetJobEnrichment` + deletes the outbox row in one transaction. `SetJobEnrichment` is deliberately separate from `UpsertJob` so ingest and enrichment stay decoupled. The LLM is configured by `LLM_BASE_URL`/`LLM_API_KEY`/`LLM_MODEL` (any OpenAI-compatible endpoint) — never hard-code a vendor or model.
