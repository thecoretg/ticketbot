-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_config ADD COLUMN write_cap_per_ticket INT NOT NULL DEFAULT 20;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_config DROP COLUMN IF EXISTS write_cap_per_ticket;
-- +goose StatementEnd
