-- +goose Up
-- +goose StatementBegin
-- Roles. Every existing user keeps the full access they have today.
ALTER TABLE api_user ADD COLUMN role TEXT NOT NULL DEFAULT 'viewer'
    CHECK (role IN ('admin', 'editor', 'viewer'));
UPDATE api_user SET role = 'admin';

-- Entra object ID linking a local account to a Microsoft identity.
ALTER TABLE api_user ADD COLUMN entra_oid TEXT;
CREATE UNIQUE INDEX api_user_entra_oid_idx ON api_user (entra_oid) WHERE entra_oid IS NOT NULL;

ALTER TABLE app_config ADD COLUMN sso_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE app_config ADD COLUMN password_login_enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- entra.StateStore: one row per in-flight Microsoft sign-in, keyed by the flow cookie digest.
CREATE TABLE sso_flow_state (
    key TEXT PRIMARY KEY,
    state TEXT NOT NULL,
    nonce TEXT NOT NULL,
    verifier TEXT NOT NULL,
    next TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL
);

-- entra.SessionStore: signed-in browsers, keyed by the session cookie digest.
CREATE TABLE sso_session (
    key TEXT PRIMARY KEY,
    user_id INT NOT NULL REFERENCES api_user(id) ON DELETE CASCADE,
    created_on TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX sso_session_user_idx ON sso_session (user_id);

-- Entra app role value -> ticketbot role.
CREATE TABLE sso_role_mapping (
    id SERIAL PRIMARY KEY,
    entra_role TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL CHECK (role IN ('admin', 'editor', 'viewer')),
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sso_role_mapping;
DROP TABLE IF EXISTS sso_session;
DROP TABLE IF EXISTS sso_flow_state;
ALTER TABLE app_config DROP COLUMN IF EXISTS password_login_enabled;
ALTER TABLE app_config DROP COLUMN IF EXISTS sso_enabled;
DROP INDEX IF EXISTS api_user_entra_oid_idx;
ALTER TABLE api_user DROP COLUMN IF EXISTS entra_oid;
ALTER TABLE api_user DROP COLUMN IF EXISTS role;
-- +goose StatementEnd
