-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_config ADD COLUMN note_preview_length INT NOT NULL DEFAULT 200;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_config DROP COLUMN IF EXISTS note_preview_length;
-- +goose StatementEnd
