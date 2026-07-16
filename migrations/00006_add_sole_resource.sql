-- +goose Up
-- +goose StatementBegin
ALTER TABLE notifier_forward
    ADD COLUMN only_if_sole_resource BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE notifier_forward DROP COLUMN only_if_sole_resource;
-- +goose StatementEnd
