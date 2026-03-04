-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_config ADD COLUMN log_buffer_size INT NOT NULL DEFAULT 500;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_config DROP COLUMN log_buffer_size;
-- +goose StatementEnd
