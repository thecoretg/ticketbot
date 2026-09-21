-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_config
    ADD COLUMN business_open TEXT NOT NULL DEFAULT '07:30',
    ADD COLUMN business_close TEXT NOT NULL DEFAULT '19:00',
    ADD COLUMN business_days TEXT NOT NULL DEFAULT 'mon,tue,wed,thu,fri',
    ADD COLUMN business_zone TEXT NOT NULL DEFAULT 'America/Chicago';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_config
    DROP COLUMN IF EXISTS business_open,
    DROP COLUMN IF EXISTS business_close,
    DROP COLUMN IF EXISTS business_days,
    DROP COLUMN IF EXISTS business_zone;
-- +goose StatementEnd
