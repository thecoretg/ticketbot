-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_config ADD COLUMN root_url       TEXT NOT NULL DEFAULT '';
ALTER TABLE app_config ADD COLUMN cw_company_id  TEXT NOT NULL DEFAULT '';
ALTER TABLE app_config ADD COLUMN cw_client_id   TEXT NOT NULL DEFAULT '';
ALTER TABLE app_config ADD COLUMN cw_public_key  TEXT NOT NULL DEFAULT '';
ALTER TABLE app_config ADD COLUMN cw_private_key TEXT NOT NULL DEFAULT '';
ALTER TABLE app_config ADD COLUMN webex_secret   TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_config DROP COLUMN webex_secret;
ALTER TABLE app_config DROP COLUMN cw_private_key;
ALTER TABLE app_config DROP COLUMN cw_public_key;
ALTER TABLE app_config DROP COLUMN cw_client_id;
ALTER TABLE app_config DROP COLUMN cw_company_id;
ALTER TABLE app_config DROP COLUMN root_url;
-- +goose StatementEnd
