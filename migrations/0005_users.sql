-- User accounts: the canonical identity for authentication.
-- Applied automatically by Postgres on first volume init (same as 0001) and
-- also serves as schema source for sqlc.

CREATE TABLE IF NOT EXISTS users (
    id            BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         TEXT        NOT NULL,
    -- NULLABLE on purpose: passwordless sign-in methods (Google OAuth,
    -- magic link) create accounts with no password. Password login treats a
    -- NULL hash as "this account has no password" and rejects it.
    password_hash TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Email is the canonical account key, case-insensitive. A UNIQUE constraint
-- can't target an expression, so uniqueness is enforced by an index on
-- lower(email); the handler also stores the lowercased form.
CREATE UNIQUE INDEX IF NOT EXISTS users_email_lower_idx ON users (lower(email));
