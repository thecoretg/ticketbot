-- +goose Up
-- +goose StatementBegin
ALTER TABLE api_user ADD COLUMN password_reset_required BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE api_user DROP COLUMN password_reset_required;
-- +goose StatementEnd
