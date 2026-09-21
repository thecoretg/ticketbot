-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_config
    ADD COLUMN ops_room_id INT REFERENCES webex_recipient(id) ON DELETE SET NULL,
    ADD COLUMN redirect_room_id INT REFERENCES webex_recipient(id) ON DELETE SET NULL,
    ADD COLUMN stale_alert_minutes INT NOT NULL DEFAULT 60;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_config
    DROP COLUMN IF EXISTS ops_room_id,
    DROP COLUMN IF EXISTS redirect_room_id,
    DROP COLUMN IF EXISTS stale_alert_minutes;
-- +goose StatementEnd
