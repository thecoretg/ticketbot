-- +goose Up
-- +goose StatementBegin
ALTER TABLE api_user ADD COLUMN totp_secret  TEXT;
ALTER TABLE api_user ADD COLUMN totp_enabled BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE totp_pending (
    id         SERIAL PRIMARY KEY,
    user_id    INT       NOT NULL REFERENCES api_user(id) ON DELETE CASCADE,
    token_hash BYTEA     NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE totp_recovery_code (
    id         SERIAL PRIMARY KEY,
    user_id    INT       NOT NULL REFERENCES api_user(id) ON DELETE CASCADE,
    code_hash  BYTEA     NOT NULL,
    used       BOOLEAN   NOT NULL DEFAULT FALSE,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS totp_recovery_code;
DROP TABLE IF EXISTS totp_pending;
ALTER TABLE api_user DROP COLUMN totp_enabled;
ALTER TABLE api_user DROP COLUMN totp_secret;
-- +goose StatementEnd
