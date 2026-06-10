## ADDED Requirements

### Requirement: User registration

The system SHALL allow a new user to register with an email and password,
creating exactly one account per email and starting a session on success.

- Email MUST be unique (case-insensitive); the stored form is lowercased.
- Password MUST be at least 8 characters; it is stored only as a bcrypt hash,
  never in plaintext and never returned in any response.
- On success the system returns the created user (id, email, created_at) and
  sets the httpOnly session cookie carrying a signed JWT.

#### Scenario: Successful registration

- **WHEN** a client POSTs a unique, well-formed email and an 8+ character password to `/api/v1/auth/register`
- **THEN** the system creates the user, stores a bcrypt hash of the password, and responds `201` with the user (no password hash) and a `Set-Cookie` carrying the session token

#### Scenario: Duplicate email

- **WHEN** a client registers with an email that already exists (in any letter case)
- **THEN** the system responds `409` and creates no new account

#### Scenario: Invalid input

- **WHEN** a client submits a malformed email or a password shorter than 8 characters
- **THEN** the system responds `400` and creates no account

### Requirement: User login

The system SHALL authenticate an existing user by email and password and start a
session, without revealing whether the email or the password was the failing
factor.

#### Scenario: Successful login

- **WHEN** a client POSTs a registered email and the correct password to `/api/v1/auth/login`
- **THEN** the system responds `200` with the user and sets the httpOnly session cookie

#### Scenario: Wrong password

- **WHEN** a client submits a registered email with an incorrect password
- **THEN** the system responds `401` with a generic "invalid credentials" message and sets no cookie

#### Scenario: Unknown email

- **WHEN** a client submits an email that has no account
- **THEN** the system responds `401` with the same generic "invalid credentials" message as a wrong password

#### Scenario: Account has no password

- **WHEN** a client attempts password login for an account that has no stored password hash (e.g. one created through a future passwordless sign-in method)
- **THEN** the system responds `401` with the same generic "invalid credentials" message, never treating an absent password as a match

### Requirement: Stateless cookie session

The system SHALL issue stateless JWTs (HS256) on register and login, delivered
in an httpOnly cookie, and SHALL validate that cookie on protected requests.

- The token SHALL encode the user id as its subject and carry an expiry.
- The cookie SHALL be `HttpOnly` and `SameSite=Lax`, with `Secure` configurable
  (set in HTTPS deployments) and a max-age matching the token expiry.
- A protected handler MUST be able to resolve the authenticated user's id from
  the validated cookie.

#### Scenario: Valid cookie grants access

- **WHEN** a client calls a protected endpoint with a valid, unexpired session cookie
- **THEN** the system resolves the user from the cookie and serves the request

#### Scenario: Missing cookie

- **WHEN** a client calls a protected endpoint with no session cookie
- **THEN** the system responds `401` and does not serve the protected resource

#### Scenario: Expired or invalid signature

- **WHEN** a client calls a protected endpoint with an expired cookie or one whose signature does not verify against the server secret
- **THEN** the system responds `401`

### Requirement: Session logout

The system SHALL expose `POST /api/v1/auth/logout` that clears the session
cookie. It is public and idempotent.

#### Scenario: Logout clears the session

- **WHEN** a client calls `POST /api/v1/auth/logout`
- **THEN** the system responds with a `Set-Cookie` that expires the session cookie, so subsequent protected requests are unauthenticated

#### Scenario: Logout without a session

- **WHEN** a client calls `POST /api/v1/auth/logout` with no (or an already-expired) cookie
- **THEN** the system still responds successfully, treating it as a no-op

### Requirement: Current user endpoint

The system SHALL expose `GET /api/v1/auth/me` that returns the authenticated
user's profile and is only reachable with a valid session cookie.

#### Scenario: Authenticated request

- **WHEN** an authenticated client calls `GET /api/v1/auth/me`
- **THEN** the system responds `200` with the user (id, email, created_at) and never includes the password hash

#### Scenario: Unauthenticated request

- **WHEN** a client calls `GET /api/v1/auth/me` without a valid session cookie
- **THEN** the system responds `401`

### Requirement: Web client authentication

The Svelte SPA SHALL let a user register, log in, and log out from the
application layout, persist the session across reloads, and reflect the current
auth state in the top bar.

- The session SHALL live entirely in the httpOnly cookie; the SPA SHALL hold no
  token (it cannot read the cookie) and SHALL use no `localStorage`. On boot it
  SHALL resolve the session via `GET /me`; failure SHALL leave the user signed
  out without error.
- The SPA SHALL send credentials (the cookie) with API requests; the public
  jobs/companies requests SHALL remain unauthenticated either way.
- The top bar SHALL show the signed-in user's email and a logout action when
  authenticated, and Login/Register actions when not.

#### Scenario: Sign in from the layout

- **WHEN** a signed-out user submits valid credentials in the login (or register) form opened from the top bar
- **THEN** the SPA shows the user's email with a logout action in the top bar, and the session (cookie) keeps the user signed in across a page reload

#### Scenario: Log out

- **WHEN** a signed-in user activates the logout action
- **THEN** the SPA calls the logout endpoint to clear the cookie and returns the top bar to its Login/Register state

#### Scenario: Stale cookie on boot

- **WHEN** the SPA boots and `GET /me` is rejected (expired or invalid cookie)
- **THEN** the SPA presents the signed-out state without error

### Requirement: Public endpoints remain unauthenticated

The existing read endpoints SHALL remain publicly accessible without a token, so
this change adds authentication without gating current functionality.

#### Scenario: Public read without a token

- **WHEN** a client calls `GET /api/v1/jobs`, `GET /api/v1/jobs/:id`, `GET /api/v1/companies`, or `GET /api/v1/companies/:slug` without any token
- **THEN** the system serves the request as before, unaffected by the auth layer
