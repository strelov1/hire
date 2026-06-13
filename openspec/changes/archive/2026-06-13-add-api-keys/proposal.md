## Why

Today a user can only reach the freehire API through the browser SPA: auth is a
same-origin, httpOnly session cookie that JavaScript cannot read and that is tied
to the browser. There is no credential a user can put in a script, CLI, or
integration. Users who want to automate their own job workflow — search, open a
posting, and track applications — have nothing to authenticate with outside the
browser. Personal API keys give each user a revocable, named credential for
non-browser access to the endpoints they already use.

## What Changes

- Users **create, list, and revoke named API keys** in the SPA. The plaintext
  token is shown **exactly once**, at creation.
- New **API-key authentication**: a request may present
  `Authorization: Bearer <key>`; the server resolves it to the owning user. Keys
  are opaque (`fhk_<random>`), stored only as a SHA-256 hash plus a short display
  prefix, optionally expiring, and revocable — unlike the stateless session JWT,
  which cannot be revoked.
- The per-user data endpoints — record view, apply, save/unsave, and the
  "my jobs" listing — now accept a **session cookie *or* an API key**, granting
  the same per-user access either way. Public reads (jobs list/search/detail,
  companies) are unchanged and require no key.
- Key-management endpoints (`/api/v1/me/api-keys`) are **cookie-only**, so a
  leaked key cannot mint more keys.
- New SPA page `/my/api-keys` (reached from the user menu) to manage keys, with a
  one-time secret reveal and a copy-paste `curl` example.

## Capabilities

### New Capabilities

- `api-keys`: the per-user API-key lifecycle (create with a one-time secret,
  list without secrets, revoke), the opaque hashed-token model with optional
  expiry, and the Bearer API-key authentication path that identifies the owning
  user on the per-user data endpoints.

### Modified Capabilities

- `user-job-tracking`: the record-view, apply, save/unsave, and "my jobs"
  endpoints broaden their authentication precondition from session-cookie-only to
  **session cookie or API key** (identical per-user access).
- `web-frontend`: add an API-keys management page reachable from the
  authenticated user menu — list keys, create a key with a one-time secret
  reveal, and revoke a key.

## Impact

- **Database:** new `migrations/0013_api_keys.sql` (`api_keys` table, unique
  index on `token_hash` doubling as the auth lookup index, index on `user_id`);
  regenerate `internal/db` via sqlc. New `internal/db/queries/api_keys.sql`.
- **internal/auth:** new `apikey.go` (token generate + SHA-256 hash) and a
  `RequireAuthOrKey` middleware plus an `APIKeyAuthenticator` seam (satisfied
  directly by `*db.Queries`); existing `RequireAuth` is unchanged.
- **internal/handler:** new `api_keys.go` (create/list/revoke under
  `/api/v1/me/api-keys`, cookie-only); `Register` wires `RequireAuthOrKey` onto
  the five per-user endpoints.
- **web/ (SPA):** new `/my/api-keys` route, `ApiKeysView.svelte`, a `UserMenu`
  entry, and three `lib/api.ts` functions.
- **No breaking changes:** existing cookie auth and public reads are untouched;
  the dual-auth endpoints stay fully backward-compatible for the browser.
- **Out of scope (noted seams):** per-key rate limiting (consistent with no login
  rate-limit today), per-key scopes, soft-delete/audit of revoked keys, and an
  OpenAPI/public docs page (the create screen's `curl` example covers usage).
