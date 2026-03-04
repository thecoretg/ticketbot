-- +goose Up
-- +goose StatementBegin
CREATE TABLE app_log (
    id         SERIAL      PRIMARY KEY,
    time       TIMESTAMPTZ NOT NULL,
    level      TEXT        NOT NULL,
    message    TEXT        NOT NULL,
    attrs      JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_app_log_time ON app_log(time);

ALTER TABLE app_config ADD COLUMN log_retention_days         INT NOT NULL DEFAULT 7;
ALTER TABLE app_config ADD COLUMN log_cleanup_interval_hours INT NOT NULL DEFAULT 24;
ALTER TABLE app_config ADD COLUMN log_buffer_size            INT NOT NULL DEFAULT 500;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_config DROP COLUMN log_buffer_size;
ALTER TABLE app_config DROP COLUMN log_cleanup_interval_hours;
ALTER TABLE app_config DROP COLUMN log_retention_days;
DROP INDEX IF EXISTS idx_app_log_time;
DROP TABLE IF EXISTS app_log;
-- +goose StatementEnd
