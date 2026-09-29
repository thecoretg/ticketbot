-- +goose Up
-- +goose StatementBegin
-- scheduled_job records each job that runs on a clock, one row per job name, so a restart knows
-- whether today's run already happened.
CREATE TABLE scheduled_job (
    name TEXT PRIMARY KEY,
    last_started_at TIMESTAMPTZ,
    last_finished_at TIMESTAMPTZ,
    last_error TEXT
);

ALTER TABLE app_config
    ADD COLUMN nightly_sync_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN nightly_sync_time TEXT NOT NULL DEFAULT '02:00',
    ADD COLUMN nightly_sync_run_workflows BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_config
    DROP COLUMN IF EXISTS nightly_sync_enabled,
    DROP COLUMN IF EXISTS nightly_sync_time,
    DROP COLUMN IF EXISTS nightly_sync_run_workflows;
DROP TABLE IF EXISTS scheduled_job;
-- +goose StatementEnd
