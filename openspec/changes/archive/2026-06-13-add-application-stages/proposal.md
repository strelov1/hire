## Why

A signed-in user can record that they viewed, saved, or applied to a job — but
once they apply, there is nowhere to track what happens next (recruiter reply,
screen, interview, offer) or to keep a note. The `user_jobs` model was built as
the first slice of a personal application tracker; `hire/CLAUDE.md` flags a
`stage` column (Applied→…→Offer) and notes as the designed-but-unbuilt next
slice. This change adds them, so a user — or an API-key client, the freehire
CLI, or career-ops — can move an application through its pipeline and jot notes.

## What Changes

- Two new **nullable** columns on `user_jobs`: `stage` (application pipeline
  position; NULL = not in the pipeline) and `notes` (free text; NULL = none).
- A **stage vocabulary**, validated in Go (like the enrichment vocabularies, not
  a DB constraint): active `applied, screening, responded, interview, offer`;
  terminal `accepted, rejected, withdrawn`. Transitions are free (any valid
  stage from any other — a personal tracker, the user may jump or correct).
- `MarkApplied` seeds `stage = 'applied'` when it is NULL, so applying enters the
  pipeline; an existing stage is left untouched. stage/notes are otherwise
  independent of `applied_at`.
- New endpoint **`PATCH /api/v1/jobs/:slug/track`** with `{stage?, notes?}`,
  authenticated by session cookie **or** API key. It upserts the interaction row
  and sets the provided fields (partial update), validating the stage. At least
  one field is required.
- The interaction record and the "my jobs" listing rows gain `stage` and `notes`.
- The **My Jobs** SPA page shows each application's stage as a badge, lets the
  user change it from a dropdown, and edit notes inline.

## Capabilities

### Modified Capabilities

- `user-job-tracking`: applications gain a **stage** and **notes** — a new
  `track` endpoint, `MarkApplied` seeding `stage='applied'`, the stage
  vocabulary with validation, `stage`/`notes` on the interaction and my-jobs
  response shapes, and the My Jobs SPA surface that shows and edits them. (This
  capability already owns the SPA's interaction-state requirements, so the UI
  part lives here rather than in `web-frontend`.)

## Impact

- **Database:** `migrations/0014_application_stage.sql` adds `stage` + `notes` to
  `user_jobs`; regenerate sqlc; `internal/db/queries/user_jobs.sql` gains
  `TrackJob` and tweaks `MarkJobApplied` to seed `stage='applied'`.
- **internal/handler:** a `TrackJob` handler + `PATCH /jobs/:slug/track` wired
  behind `RequireAuthOrKey`; the stage vocabulary + validation; `stage`/`notes`
  added to `interactionResponse` and `myJobResponse`.
- **web/ (SPA):** `MyJobsView.svelte` (stage badge + dropdown + notes textarea),
  a `trackJob` function in `lib/api.ts`, and `stage`/`notes` on the `UserJob` /
  `MyJob` types.
- **No breaking changes:** additive columns + a new endpoint; existing endpoints
  only gain fields. **Prod:** apply `0014` manually via psql per the ops runbook,
  then deploy the app + web images.
- **Out of scope:** a stage-specific filter tab, a notes timeline/log,
  forward-only transition enforcement, and funnel analytics (career-ops owns
  those); the freehire-cli `stage`/`note` commands live in the separate
  `freehire-cli` repo as a downstream follow-up.
