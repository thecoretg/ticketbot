-- +goose Up
-- +goose StatementBegin
-- Switch for the MCP server and its OAuth endpoints. Off hides every one of them (404).
ALTER TABLE app_config ADD COLUMN mcp_enabled BOOLEAN NOT NULL DEFAULT FALSE;

-- Dynamically registered OAuth clients (Claude.ai, Claude Desktop, Claude Code). A registration
-- that never completes a grant expires; expires_at is cleared by the first grant.
CREATE TABLE oauth_client (
    id            TEXT        PRIMARY KEY,
    name          TEXT        NOT NULL,
    redirect_uris TEXT[]      NOT NULL,
    created_on    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ
);

-- A user's approval of a client for a set of scopes. Tokens hang off the grant, so deleting
-- the grant revokes everything the client holds.
CREATE TABLE oauth_grant (
    id           SERIAL      PRIMARY KEY,
    user_id      INT         NOT NULL REFERENCES api_user(id) ON DELETE CASCADE,
    client_id    TEXT        NOT NULL REFERENCES oauth_client(id) ON DELETE CASCADE,
    scopes       TEXT[]      NOT NULL,
    created_on   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    UNIQUE (user_id, client_id)
);

-- One-shot authorization codes, stored as SHA-256 of the code.
CREATE TABLE oauth_code (
    code_hash      BYTEA       PRIMARY KEY,
    grant_id       INT         NOT NULL REFERENCES oauth_grant(id) ON DELETE CASCADE,
    redirect_uri   TEXT        NOT NULL,
    code_challenge TEXT        NOT NULL,
    expires_at     TIMESTAMPTZ NOT NULL
);

-- Access and refresh tokens, stored as SHA-256 of the token. A refresh token is kept after use
-- with used_at set so a replay can be recognised and the grant revoked.
CREATE TABLE oauth_token (
    id         SERIAL      PRIMARY KEY,
    grant_id   INT         NOT NULL REFERENCES oauth_grant(id) ON DELETE CASCADE,
    kind       TEXT        NOT NULL CHECK (kind IN ('access', 'refresh')),
    token_hash BYTEA       NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_on TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used_at    TIMESTAMPTZ
);
CREATE INDEX oauth_token_grant_idx ON oauth_token (grant_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS oauth_token;
DROP TABLE IF EXISTS oauth_code;
DROP TABLE IF EXISTS oauth_grant;
DROP TABLE IF EXISTS oauth_client;
ALTER TABLE app_config DROP COLUMN IF EXISTS mcp_enabled;
-- +goose StatementEnd
