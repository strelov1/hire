## Context

freehire authentication today is a single credential: a same-origin, httpOnly
session cookie holding an HS256 JWT whose only claim is the user id. `auth.Issuer`
mints and verifies it; `auth.RequireAuth` reads the cookie, verifies the token,
and stores the user id in `c.Locals("auth.userID")`. Every per-user handler reads
identity solely through `auth.UserID(c)` — it never inspects how the id arrived.
The JWT is deliberately stateless (no server-side store), which is why it cannot
be revoked: it lives until `exp`.

There is no credential a user can use outside the browser. This change introduces
a second, **stateful** credential — a user-managed API key — that authenticates
the same per-user endpoints a session does, while leaving the cookie path and the
public reads untouched.

## Goals / Non-Goals

**Goals:**

- Let a signed-in user create, list, and revoke named API keys from the SPA.
- Authenticate the per-user data endpoints (record view, apply, save/unsave, list
  "my jobs") with a session cookie **or** an API key, with identical access.
- Make keys revocable, optionally expiring, and safe at rest (hashed, shown once).
- Reuse every existing handler unchanged via the shared `c.Locals` identity seam.

**Non-Goals:**

- Per-key scopes/permissions (moot: reads are public, so a read-only key would
  gate nothing).
- Per-key rate limiting (consistent with the absence of a login rate-limit today).
- Soft-delete/audit history of revoked keys.
- A public OpenAPI/docs site (the create screen ships a `curl` example instead).
- Changing the cookie/JWT session model or the public read endpoints.

## Decisions

- **D1 — Stateful opaque tokens, not long-lived JWTs.** Keys are DB rows looked
  up per request, which gives revocation, listing, naming, and "last used" for
  free. *Alternative:* issue a long-TTL JWT as the "key" — rejected: stateless,
  so not revocable and nothing to list or name, defeating the feature.
- **D2 — Store SHA-256 of the token, not bcrypt.** The token is high-entropy
  random, so a single SHA-256 is sufficient and lets us look up by an indexed
  `token_hash` in O(1). *Alternative:* bcrypt — rejected: per-row salt forbids an
  indexed lookup, and slow hashing only matters for low-entropy passwords.
- **D3 — A second middleware `RequireAuthOrKey` that sets the same `c.Locals`.**
  It tries the cookie first (existing path, unchanged); on a missing/invalid
  cookie it reads `Authorization: Bearer <key>`, hashes it, and resolves the
  owner. Because it writes the same `auth.userID` local, all existing handlers
  work with no change. *Alternatives:* (a) extend `RequireAuth` in place —
  rejected: forces a DB dependency onto cookie-only routes and we still need a
  cookie-only variant for key management; (b) put the middleware in `handler` —
  rejected: splits auth logic out of the `auth` package, which owns `RequireAuth`.
- **D4 — `APIKeyAuthenticator` is a tiny interface satisfied directly by
  `*db.Queries`.** `auth` declares `AuthenticateAPIKey(ctx, tokenHash) (int64,
  error)`; the sqlc-generated method matches it, so no adapter is needed and
  `auth` keeps importing no `db` (mirrors `oauth.Provider` / `enrich.Provider`).
- **D5 — Authenticate, enforce expiry, and touch `last_used_at` in one
  statement.** `UPDATE api_keys SET last_used_at = now() WHERE token_hash = $1 AND
  (expires_at IS NULL OR expires_at > now()) RETURNING user_id` — atomic; no row
  returned means unknown/expired/revoked → 401.
- **D6 — Key management is cookie-only.** `POST/GET/DELETE /api/v1/me/api-keys`
  use `RequireAuth` (cookie), so a leaked key cannot mint more keys or escalate.
- **D7 — Reads stay public; the read hot path does no key lookup.** A client may
  send the key uniformly to every endpoint, but read endpoints ignore it.
- **D8 — Revoke = hard `DELETE` of the row** (owner-scoped via `id AND user_id`).
  Soft-delete/audit is a noted seam.
- **D9 — Optional expiry** (`expires_at` nullable, default never).
- **D10 — Store a short display `token_prefix`** (e.g. `fhk_Ab12cd`) so a user can
  identify a key in the list; revealing ~36 bits of a 256-bit secret is safe and
  standard (GitHub/Stripe).

## Risks / Trade-offs

- **A leaked key grants full per-user access until revoked** → shown once, hashed
  at rest, instantly revocable, cookie-only management, optional expiry; per-key
  rate limiting is a noted seam.
- **No versioned migration runner; migrations apply via Postgres initdb only** →
  dev must recreate the volume (`docker compose down -v && make up`); prod applies
  `0013` manually via psql per the ops runbook. Rollback is `DROP TABLE api_keys`
  — additive change, nothing else depends on it.
- **`last_used_at` is written on every key-authenticated request** → these are
  low-volume per-user calls; the public read path (high volume) does no lookup, so
  there is no hot-path write.
- **Spec drift: `add-job-tracker` (save/unsave, `/me/jobs`) is merged in code but
  not yet archived into the main spec** → the dual-auth coverage of those
  endpoints is specified in the new `api-keys` capability; the wiring targets
  code that already exists on `main`.
- **Token comparison** → lookup is exact-equality on an indexed hash column, so
  there is no value-dependent timing oracle.

## Migration Plan

1. Add `migrations/0013_api_keys.sql` and `internal/db/queries/api_keys.sql`;
   run `make sqlc`; recreate the dev volume.
2. Ship code (auth middleware, handlers, routes, SPA).
3. Prod: apply `0013` via psql per the ops runbook before/with the deploy.

Rollback: hide the SPA route and remove the `/me/api-keys` routes + the
`RequireAuthOrKey` wiring (cookie auth keeps working); drop the table when fully
reverting. The change is additive and backward-compatible for the browser.

## Open Questions

None outstanding — expiry (optional, default never), revoke (hard delete), and
`last_used_at` (stored and shown) were confirmed during design. Future
considerations: a rate-limiting policy and soft-delete/audit if either becomes a
real need.
