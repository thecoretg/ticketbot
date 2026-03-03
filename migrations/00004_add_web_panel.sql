-- +goose Up
-- +goose StatementBegin
ALTER TABLE api_user ADD COLUMN password_hash            BYTEA;
ALTER TABLE api_user ADD COLUMN password_reset_required  BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE api_user ADD COLUMN totp_secret              TEXT;
ALTER TABLE api_user ADD COLUMN totp_enabled             BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE session (
    id         SERIAL PRIMARY KEY,
    user_id    INT         NOT NULL REFERENCES api_user(id) ON DELETE CASCADE,
    token_hash BYTEA       NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_on TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE totp_pending (
    id         SERIAL PRIMARY KEY,
    user_id    INT         NOT NULL REFERENCES api_user(id) ON DELETE CASCADE,
    token_hash BYTEA       NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_on TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE totp_recovery_code (
    id         SERIAL PRIMARY KEY,
    user_id    INT       NOT NULL REFERENCES api_user(id) ON DELETE CASCADE,
    code_hash  BYTEA     NOT NULL,
    used       BOOLEAN   NOT NULL DEFAULT FALSE,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE app_config ADD COLUMN require_totp BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE app_config DROP COLUMN skip_launch_syncs;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_config ADD COLUMN skip_launch_syncs BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE app_config DROP COLUMN require_totp;
DROP TABLE IF EXISTS totp_recovery_code;
DROP TABLE IF EXISTS totp_pending;
DROP TABLE IF EXISTS session;
ALTER TABLE api_user DROP COLUMN totp_enabled;
ALTER TABLE api_user DROP COLUMN totp_secret;
ALTER TABLE api_user DROP COLUMN password_reset_required;
ALTER TABLE api_user DROP COLUMN password_hash;
-- +goose StatementEnd
