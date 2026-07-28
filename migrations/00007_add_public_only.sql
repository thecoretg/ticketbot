-- +goose Up
-- +goose StatementBegin
ALTER TABLE notifier_forward
    ADD COLUMN public_only BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE cw_ticket_note
    ADD COLUMN internal_analysis_flag BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE notifier_forward DROP COLUMN public_only;

ALTER TABLE cw_ticket_note DROP COLUMN internal_analysis_flag;
-- +goose StatementEnd
