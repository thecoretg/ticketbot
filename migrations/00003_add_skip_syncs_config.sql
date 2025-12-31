-- +goose Up 
-- +goose StatementBegin
ALTER TABLE app_config
    ADD COLUMN skip_launch_syncs BOOLEAN NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_config
    DROP COLUMN skip_launch_syncs;
-- +goose StatementEnd
