-- +goose Up
-- +goose StatementBegin
ALTER TABLE ticket_journal ADD COLUMN IF NOT EXISTS resource_names TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ticket_journal DROP COLUMN IF EXISTS resource_names;
-- +goose StatementEnd
