## Context

`user_jobs` is one row per `(user, job)` carrying `viewed_at`, `saved_at`, and
`applied_at`. `RecordView` / `MarkApplied` / `SaveJob` / `UnsaveJob` upsert it;
`ListMyJobs` lists and filters it. Handlers resolve the user via `auth.UserID`,
populated by `RequireAuthOrKey` (session cookie or API key). Nothing tracks where
an application stands once applied, and there is nowhere to keep a note —
`hire/CLAUDE.md` flags exactly this (`stage` column + notes) as the next slice.

## Goals / Non-Goals

**Goals:** track an application's position through a controlled stage pipeline and
hold free-text notes; settable by cookie or API key (so the CLI / career-ops can
use it); shown and editable on My Jobs; fully additive and backward-compatible.

**Non-Goals:** a stage-specific filter tab; a notes timeline/log; forward-only
transition enforcement; funnel analytics (career-ops owns those); the
freehire-cli `stage`/`note` commands (separate repo, downstream follow-up).

## Decisions

- **D1 — `stage` + `notes` as nullable `TEXT` columns on `user_jobs`, not a new
  table.** They are attributes of the existing interaction; NULL stage = not in
  the pipeline. *Alternative:* a separate applications table — rejected: the
  one-row-per-(user,job) interaction already is the application.
- **D2 — the stage vocabulary is validated in Go, not a DB `CHECK`.** Mirrors the
  enrichment vocabularies; adding a stage needs no migration, and an unknown
  value is a clean `400`. *Alternative:* a Postgres enum/CHECK — rejected: rigid,
  needs a migration to evolve.
- **D3 — transitions are free** (the handler validates the value is in the
  vocabulary, not the order). A personal tracker: users jump or correct.
  Forward-only enforcement is over-engineering.
- **D4 — one endpoint `PATCH /api/v1/jobs/:slug/track` with `{stage?, notes?}`,
  partial update.** The query upserts the row (like save/apply) and uses
  `COALESCE(EXCLUDED.col, user_jobs.col)` so a `nil` field leaves the column
  untouched. Behind `RequireAuthOrKey` — same access as the other per-user
  writes. At least one field required (else `400`).
- **D5 — `MarkApplied` seeds `stage='applied'` only when it is NULL**
  (`COALESCE(user_jobs.stage, 'applied')` on conflict), so applying enters the
  pipeline while a manually-advanced stage survives a re-apply.
- **D6 — `stage`/`notes` are added to `interactionResponse` and `myJobResponse`**,
  so every interaction read carries them; no new read endpoint.
- **D7 — My Jobs UI:** a stage badge + a dropdown to change it (calls `track`)
  and an inline notes textarea saved on blur (calls `track`). Folded into the
  existing page; no separate Applications page.

## Risks / Trade-offs

- **Stage vocabulary is duplicated (Go backend ↔ SPA labels ↔ future CLI)** → Go
  is the source of truth; the SPA mirrors it with a documented note (as
  `facets.ts` mirrors the enrich vocab). Drift renders an unknown stage as a
  humanized label, never a crash.
- **No migration runner; `0014` is manual** → recreate the dev volume; prod
  applies `0014` via psql (`ADD COLUMN IF NOT EXISTS`). Columns are additive +
  nullable; rollback is reverting the code (dropping the columns is optional and
  harmless).
- **Free transitions allow nonsensical jumps** → acceptable for a personal
  tracker; the UI presents stages in pipeline order.
- **`notes` empty-string vs NULL** → partial update keys on field *presence*: a
  body omitting `notes` leaves it untouched; sending `notes:""` clears it to
  empty. Both are intentional.

## Migration Plan

1. Add `migrations/0014_application_stage.sql` (`ALTER TABLE user_jobs ADD COLUMN
   IF NOT EXISTS stage TEXT; ... ADD COLUMN IF NOT EXISTS notes TEXT`); run
   `make sqlc`; recreate the dev volume.
2. Ship code (queries, handler + route, SPA).
3. Prod: apply `0014` via psql per the ops runbook (the columns are additive, so
   it is safe on the live table), then deploy the app + web images.

Rollback: revert the code; the columns can stay (ignored) or be dropped — nothing
else references them.

## Open Questions

None — the vocabulary, free transitions, single-field notes, and the single
`track` endpoint were confirmed during design.
