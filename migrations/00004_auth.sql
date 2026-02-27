-- +goose Up
-- +goose StatementBegin
ALTER TABLE api_user ADD COLUMN password_hash BYTEA;

CREATE TABLE session (
    id         SERIAL PRIMARY KEY,
    user_id    INT       NOT NULL REFERENCES api_user(id) ON DELETE CASCADE,
    token_hash BYTEA     NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS session;
ALTER TABLE api_user DROP COLUMN password_hash;
-- +goose StatementEnd
