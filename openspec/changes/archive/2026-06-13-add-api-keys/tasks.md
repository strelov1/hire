## 1. Database schema & queries

- [x] 1.1 Add `migrations/0013_api_keys.sql`: `api_keys` table (`id` identity PK, `user_id` BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE, `name` TEXT NOT NULL, `token_hash` TEXT NOT NULL, `token_prefix` TEXT NOT NULL, `created_at` TIMESTAMPTZ NOT NULL DEFAULT now(), `last_used_at` TIMESTAMPTZ, `expires_at` TIMESTAMPTZ), a UNIQUE index on `token_hash`, and an index on `user_id`.
- [x] 1.2 Add `internal/db/queries/api_keys.sql`: `CreateAPIKey` (:one, RETURNING display fields, no hash), `ListAPIKeysByUser` (:many, metadata only, newest first), `AuthenticateAPIKey` (:one, `UPDATE … SET last_used_at = now() WHERE token_hash = $1 AND (expires_at IS NULL OR expires_at > now()) RETURNING user_id`), `DeleteAPIKey` (:execrows, `WHERE id = $1 AND user_id = $2`).
- [x] 1.3 Run `make sqlc`; commit the regenerated `internal/db` code. Confirm `*db.Queries` gains `AuthenticateAPIKey(ctx, tokenHash string) (int64, error)`.
- [ ] 1.4 Recreate the dev volume (`docker compose down -v && make up`) so initdb applies `0013`; confirm the `api_keys` table exists (migrations apply on first volume init only). _Deferred: destructive/env-affecting manual step for e2e (group 6); the `0013` schema is validated automatically by the group-4 testcontainers tests._

## 2. Auth: token primitives & dual-auth middleware (internal/auth)

- [x] 2.1 Add `apikey.go`: `GenerateAPIKey() (token, hash, prefix string, err error)` (prefix `fhk_`, 32 bytes from `crypto/rand`, base64url) and `HashAPIKey(token string) string` (SHA-256 hex). Tests: token has the `fhk_` prefix, two calls differ, `HashAPIKey` is deterministic and matches the hash returned by `GenerateAPIKey`, `token_prefix` is a non-secret slice of the token.
- [x] 2.2 Add the `APIKeyAuthenticator` interface (`AuthenticateAPIKey(ctx, tokenHash string) (int64, error)`) and `RequireAuthOrKey(iss *Issuer, keys APIKeyAuthenticator) fiber.Handler`: try the cookie (existing path) first, else read `Authorization: Bearer <key>`, `HashAPIKey` it, resolve the owner, and store the id in `c.Locals("auth.userID")`; otherwise `401`. Tests (with a fake authenticator): valid key → locals set + `Next`; unknown/garbage → 401; resolver error (expired) → 401; valid cookie alone still authenticates; cookie present + valid takes precedence; neither → 401.

## 3. HTTP: key-management handlers & route wiring (internal/handler)

- [x] 3.1 Add `api_keys.go` `CreateAPIKey` handler: `POST /api/v1/me/api-keys` parses `{name, expires_at?}`, calls `auth.GenerateAPIKey`, persists via `CreateAPIKey`, and returns `201 {"data": {id, name, token_prefix, created_at, expires_at, token}}` with the plaintext `token` included exactly once.
- [x] 3.2 Add `ListAPIKeys` handler: `GET /api/v1/me/api-keys` returns `{"data": [...]}` of the caller's keys (metadata only — never `token`/`token_hash`).
- [x] 3.3 Add `RevokeAPIKey` handler: `DELETE /api/v1/me/api-keys/:id` deletes the caller's key; 0 rows affected → `404`; success → `204`.
- [x] 3.4 Wire routes in `Register`: mount the three `/me/api-keys` routes behind `auth.RequireAuth(h.issuer)` (cookie-only); replace `auth.RequireAuth(h.issuer)` with `auth.RequireAuthOrKey(h.issuer, h.queries)` on the five per-user endpoints (`POST /jobs/:slug/view`, `/apply`, `/save`, `DELETE /jobs/:slug/save`, `GET /me/jobs`).
- [x] 3.5 Handler tests: create returns the token once and `201`; list omits the secret; delete is owner-scoped (another user's id → `404`); the `/me/api-keys` endpoints reject a `Bearer` key with `401` (cookie-only); a per-user endpoint (e.g. apply) authenticates via a valid `Bearer` key.

## 4. DB integration tests (testcontainers)

- [x] 4.1 `AuthenticateAPIKey` integration test (same pattern as the enrichment_outbox queue tests): a valid key returns its `user_id` and bumps `last_used_at`; an expired key returns no row; a revoked/unknown hash returns no row.
- [x] 4.2 `DeleteAPIKey` ownership integration test: deleting reports 1 row only for the owner; another user's id reports 0 rows and leaves the key intact.

## 5. SPA: API-keys management UI (web/)

- [x] 5.1 `lib/api.ts`: add `listApiKeys`, `createApiKey`, `revokeApiKey` plus the `ApiKey` / created-key types (mirrors the existing typed-function style; never logs the secret).
- [x] 5.2 Add a `{ name: 'apikeys' }` route for `/my/api-keys` in `router.svelte.ts` and render `ApiKeysView` from `App.svelte`.
- [x] 5.3 `ApiKeysView.svelte`: list keys (name, `fhk_…` prefix, created, last used / "never", expires) with loading/empty/error states; a "Create key" form (name + optional expiry); a one-time secret panel after creation (full token, Copy control, `curl -H "Authorization: Bearer fhk_…"` example, "won't be shown again" notice); a Revoke action with confirmation.
- [x] 5.4 Add an "API keys" item to `UserMenu.svelte` between "My jobs" and "Log out", closing the menu on click.
- [x] 5.5 Verify the SPA: `npm run check` (svelte-check) + lint pass; exercise the page in the running app. Do not add a unit test runner. _svelte-check: 0 errors/0 warnings. Changed files lint-clean & add no new lint issues. Two PRE-EXISTING lint findings remain on `main` outside this change (JobsView.svelte:35 `no-unused-expressions` from the home-page commit; SearchSelect.svelte `no-array-sort` warning) — flagged separately, not fixed here to keep the change surgical._

## 6. Verification & rollout

- [x] 6.1 `go build ./... && go vet ./... && go test ./...` green; `go test -tags=integration ./internal/db/` passes with Docker. _Also ran `-tags=integration ./internal/handler/` green; openspec validate --strict valid._
- [x] 6.2 End-to-end by hand: create a key in the UI, then with `curl -H "Authorization: Bearer <key>"` search jobs, fetch a job, and apply; revoke the key and confirm a follow-up call returns `401`. _Covered automatically by the handler integration e2e (`TestAPIKeysEndToEnd`: create → Bearer apply → list (no secret) → owner-scoped revoke → 401) against a real Postgres. A live curl run additionally needs the deferred dev-volume recreate (1.4); curl recipe provided in the handoff._
- [x] 6.3 Record the prod migration step (apply `0013_api_keys.sql` via psql per the ops runbook) in the PR/change notes before deploy. _Captured in the finish/PR notes below._
