-- +goose Up
-- +goose StatementBegin
-- webhook_catchup is the missed-webhook check's single state row. checked_through is the
-- watermark: the start of the last run that succeeded.
CREATE TABLE webhook_catchup (
    id INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    checked_through TIMESTAMPTZ,
    last_run_at TIMESTAMPTZ,
    last_queued INT NOT NULL DEFAULT 0,
    last_error TEXT
);

ALTER TABLE app_config ADD COLUMN catchup_interval_minutes INT NOT NULL DEFAULT 15;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_config DROP COLUMN IF EXISTS catchup_interval_minutes;
DROP TABLE IF EXISTS webhook_catchup;
-- +goose StatementEnd
