## 1. Database schema & queries

- [x] 1.1 Add `migrations/0014_application_stage.sql` (confirm the next free number with `ls migrations/`): `ALTER TABLE user_jobs ADD COLUMN IF NOT EXISTS stage TEXT; ALTER TABLE user_jobs ADD COLUMN IF NOT EXISTS notes TEXT;` with a comment on the controlled-vocabulary stage and the initdb/manual-apply note.
- [x] 1.2 In `internal/db/queries/user_jobs.sql`: add `TrackJob` (:one) — `INSERT ... (user_id, job_id, stage, notes) ... ON CONFLICT (user_id, job_id) DO UPDATE SET stage = COALESCE(EXCLUDED.stage, user_jobs.stage), notes = COALESCE(EXCLUDED.notes, user_jobs.notes) RETURNING *`; tweak `MarkJobApplied` so the conflict update sets `stage = COALESCE(user_jobs.stage, 'applied')`.
- [x] 1.3 Run `make sqlc`; commit the regenerated `internal/db`. Confirm the row structs gain `Stage`/`Notes` and `TrackJobParams` carries nil-able `pgtype.Text` stage/notes.
- [ ] 1.4 Recreate the dev volume (`docker compose down -v && make up`) so initdb applies `0014`. _Deferred manual step (env-affecting); the `0014` schema is validated by the group-3 testcontainers tests._

## 2. Handler: track endpoint, stage vocabulary, response fields

- [x] 2.1 Add the stage vocabulary as a Go set + an `isValidStage` helper (active: applied/screening/responded/interview/offer; terminal: accepted/rejected/withdrawn). Unit tests: known values valid, unknown invalid.
- [x] 2.2 Add a `TrackJob` handler in `user_jobs.go`: parse `{stage?, notes?}` (pointers), require at least one (else `400`), validate a provided stage (else `400`), call `db.TrackJob` with `pgtype.Text` (Valid only when the field was present), return `toInteraction`. Unit tests (no DB): empty body → 400; unknown stage → 400; track behind cookie/key (the gate). DB-backed behavior in group 3.
- [x] 2.3 Add `Stage`/`Notes` to `interactionResponse` and `myJobResponse` (and `toInteraction`/the my-jobs mapper); marshal as null when unset. Shape tests: the fields are present.
- [x] 2.4 Wire `PATCH /api/v1/jobs/:slug/track` behind `auth.RequireAuthOrKey(h.issuer, h.queries)` in `Register`.
- [x] 2.5 Confirm `MarkApplied` seeds `stage='applied'` (via the query) without a handler change — covered by the group-3 integration test.

## 3. DB integration tests (testcontainers)

- [x] 3.1 `TrackJob` partial-update: stage-only leaves notes unchanged; notes-only leaves stage unchanged; first track creates the row (upsert).
- [x] 3.2 `MarkJobApplied` seeds `stage='applied'` when stage is NULL, and leaves an already-advanced stage untouched on re-apply.

## 4. SPA: stage + notes on My Jobs

- [x] 4.1 `lib/types.ts`: `UserJob` / `MyJob` gain `stage: string | null` and `notes: string | null`. `lib/api.ts`: add `trackJob(slug, { stage?, notes? })` → `PATCH /api/v1/jobs/:slug/track`.
- [x] 4.2 Add the stage vocabulary + humanized labels in the SPA (mirroring `facets.ts`; note the Go source of truth).
- [x] 4.3 `MyJobsView.svelte`: per row, a stage badge (humanized) when set, a dropdown to change the stage (calls `trackJob({stage})`), and a notes textarea saved on blur (calls `trackJob({notes})`); optimistic local update.
- [x] 4.4 Verify the SPA: `npm run check` (svelte-check) + lint pass. No unit runner added.

## 5. Verification & rollout

- [x] 5.1 `go build ./... && go vet ./... && go test ./...` green; `go test -tags=integration ./internal/db/ ./internal/handler/` passes with Docker.
- [x] 5.2 End-to-end by hand: `curl -X PATCH -H "Authorization: Bearer fhk_…" -d '{"stage":"interview"}' …/jobs/<slug>/track` sets the stage; set notes; confirm both appear in `GET /me/jobs`; exercise the My Jobs UI. _Verified on prod (freehire.dev) with a real key: track→200 with stage+notes, /me/jobs row carries both._
- [x] 5.3 Record the prod migration step (apply `0014_application_stage.sql` via psql per the ops runbook) in the PR; deploy app + web. _0014 applied via psql (additive, idempotent); app+web deployed from clean origin/main._
- [ ] 5.4 Follow-up (separate `freehire-cli` repo, not this change): add `freehire stage <slug> <stage>` and `freehire note <slug> <text>` (→ PATCH track) and show stage/notes in `my`.
