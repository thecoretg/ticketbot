-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_config
    ADD COLUMN intake_retention_days INT NOT NULL DEFAULT 7,
    ADD COLUMN closed_ticket_retention_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN closed_ticket_retention_days INT NOT NULL DEFAULT 60,
    DROP COLUMN IF EXISTS log_cleanup_interval_hours;

-- intake rows aged out on log_retention_days until now; carry the value over so nothing changes
UPDATE app_config SET intake_retention_days = log_retention_days WHERE log_retention_days > 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_config
    ADD COLUMN log_cleanup_interval_hours INT NOT NULL DEFAULT 24,
    DROP COLUMN IF EXISTS intake_retention_days,
    DROP COLUMN IF EXISTS closed_ticket_retention_enabled,
    DROP COLUMN IF EXISTS closed_ticket_retention_days;
-- +goose StatementEnd
