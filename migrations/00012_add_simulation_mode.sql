-- +goose Up
-- +goose StatementBegin
ALTER TABLE workflow         ADD COLUMN simulation_mode BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE notifier_rule    ADD COLUMN simulation_mode BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE notifier_forward ADD COLUMN simulation_mode BOOLEAN NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE workflow         DROP COLUMN simulation_mode;
ALTER TABLE notifier_rule    DROP COLUMN simulation_mode;
ALTER TABLE notifier_forward DROP COLUMN simulation_mode;
-- +goose StatementEnd
