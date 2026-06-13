-- Per-user API keys: a stateful, revocable credential for non-browser API access.
-- Each row is one named key owned by a user. The plaintext token is never stored —
-- only its SHA-256 hash (token_hash, the per-request lookup key) and a short
-- non-secret display prefix (token_prefix). last_used_at is touched on each
-- authenticated use; NULL expires_at means the key never expires; revoking a key
-- deletes its row. Applied automatically by Postgres on first volume init (same as
-- 0001) and also serves as schema source for sqlc. Existing volumes/prod need a
-- manual apply (the versioned-migration-runner seam from AGENT.md remains open).

CREATE TABLE IF NOT EXISTS api_keys (
    id           BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id      BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name         TEXT        NOT NULL,
    -- SHA-256 (hex) of the opaque token. The token itself is shown once at creation
    -- and never stored; this hash is the per-request authentication lookup key.
    token_hash   TEXT        NOT NULL,
    -- Non-secret leading slice of the token (e.g. "fhk_Ab12cd") for display, so a
    -- user can tell their keys apart in the list.
    token_prefix TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- NULL until the key first authenticates a request; touched on each use.
    last_used_at TIMESTAMPTZ,
    -- NULL = never expires.
    expires_at   TIMESTAMPTZ
);

-- The hash is unique and is the authentication lookup key (a single indexed probe).
CREATE UNIQUE INDEX IF NOT EXISTS api_keys_token_hash_idx ON api_keys (token_hash);
-- List-by-owner (newest first) for the management page.
CREATE INDEX IF NOT EXISTS api_keys_user_id_idx ON api_keys (user_id);
