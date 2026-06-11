## 1. Database

- [x] 1.1 Add `migrations/0005_users.sql`: `users` table (id identity PK, email text not null, password_hash text **NULLABLE** — passwordless users have none, created_at timestamptz default now()) with `UNIQUE (lower(email))`
- [x] 1.2 Add `internal/db/queries/users.sql` with `CreateUser` (returns id, email, created_at), `GetUserByEmail` (incl. password_hash, for login), `GetUserByID` (no password_hash, for /me)
- [x] 1.3 Run `make sqlc` and commit the regenerated `internal/db` code
- [x] 1.4 Recreate the dev DB volume (`docker compose down -v && make up`) to apply the new migration

## 2. Dependencies & Config

- [x] 2.1 Add `golang.org/x/crypto/bcrypt` and `github.com/golang-jwt/jwt/v5` to go.mod (`go get`)
- [x] 2.2 Add `JWTSecret` and `JWTTTL` to `config.Settings`; load `JWT_SECRET` (no default) and `JWT_TTL` (default 24h)
- [x] 2.3 Fail fast at server startup if `JWT_SECRET` is empty

## 3. Auth package (`internal/auth`)

- [x] 3.1 Implement `HashPassword(plain) (string, error)` and `CheckPassword(hash, plain) error` over bcrypt
- [x] 3.2 Implement an `Issuer` (secret + TTL): `Issue(userID) (string, error)` producing an HS256 JWT with `sub` + `exp`, and `Parse(token) (userID, error)` validating signature and expiry
- [x] 3.3 Implement the httpOnly cookie transport (`CookieName`, `SetTokenCookie`, `ClearTokenCookie`, single attribute source) and the `RequireAuth` Fiber middleware: read the auth cookie, validate via `Issuer`, store user id in `c.Locals`, return 401 on any failure
- [x] 3.4 Unit tests: hash round-trip + wrong-password rejection; issue→parse round-trip; expired token and bad-signature rejection

## 4. HTTP handlers (`internal/handler/auth.go`)

- [x] 4.1 Add an API user type (id, email, created_at) that never includes password_hash, and a JSON-tag-omitted mapping from the db row
- [x] 4.2 `Register` handler: validate email format + password length (>=8), lowercase email, hash, `CreateUser`, map duplicate (unique violation) to 409, set the httpOnly auth cookie, return 201 with `{"data": user}`
- [x] 4.3 `Login` handler: `GetUserByEmail`, `CheckPassword`, return generic 401 on unknown email OR wrong password OR account with a null `password_hash` (passwordless), else set the httpOnly auth cookie and return 200 with `{"data": user}`
- [x] 4.4 `Me` handler: read user id from `c.Locals`, `GetUserByID`, return 200 `{"data": user}`
- [x] 4.5 Wire routes in `handler.Register`: extend its signature with JWT secret/TTL and the cookie-secure flag, build the `auth.Issuer`, register `POST /api/v1/auth/register`, `POST /api/v1/auth/login`, `POST /api/v1/auth/logout`, and `GET /api/v1/auth/me` (guarded by `RequireAuth`)
- [x] 4.6 Update `cmd/server/main.go` to pass the new config into `handler.Register`
- [x] 4.7 `Logout` handler: clear the auth cookie via `ClearTokenCookie`; public and idempotent (no session is a no-op), return 204
- [x] 4.8 Add the central `handler.ErrorHandler` (`errors.go`) the thin auth handlers rely on: render the `{"error": msg}` envelope, map `*fiber.Error`→its code, `pgx.ErrNoRows`→404, else 500; wire it via `fiber.Config{ErrorHandler}` in `cmd/server` and drop the per-handler error JSON in `companies.go`/`jobs.go`

## 6. Web (SPA) auth integration

- [x] 6.1 Add auth API to `web/src/lib/api.ts`: `register`, `login`, `logout`, `me` functions sending credentials (the httpOnly cookie) via `fetch` `credentials: 'include'` — no token attached or read (keep public jobs/companies calls working without a session); add the `User` wire type to `web/src/lib/types.ts`
- [x] 6.2 Add `web/src/lib/auth.svelte.ts` auth store: `$state` user only (no token — the session lives in the httpOnly cookie, unreadable by JS, so no localStorage), `login`/`register`/`logout` actions, and an `initAuth()` that resolves the session via `GET /me` on boot (signed-out on failure, no error); call `initAuth()` in `main.ts`
- [x] 6.3 Add login + register form components (`web/src/lib/components/`), reachable from the layout (route or modal, matching the existing router pattern), surfacing API errors (e.g. 401/409) inline
- [x] 6.4 Wire auth controls into `TopBar.svelte`: signed-in shows user email + logout; signed-out shows Login/Register — placed alongside `ThemeToggle` in the `ml-auto` group

## 7. Verification

- [x] 7.1 `go build ./... && go vet ./... && go test ./...` all pass
- [x] 7.2 Manual e2e (API): register → `Set-Cookie`; login → `Set-Cookie`; `GET /me` with the cookie → 200; without/expired cookie → 401; `POST /logout` clears the cookie; confirm `GET /api/v1/jobs` still works with no cookie
- [x] 7.3 Manual e2e (web): register/login from the top bar → email + logout shown; reload stays signed in (cookie); logout returns to Login/Register; a tampered/expired cookie boots to signed-out without error
- [x] 7.4 Update `AGENT.md` (layout + conventions) to document the `internal/auth` package, the auth endpoints, the `JWT_SECRET`/`JWT_TTL` env vars, and the SPA auth store
