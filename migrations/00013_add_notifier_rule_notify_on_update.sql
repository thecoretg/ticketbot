-- +goose Up
-- +goose StatementBegin
-- notify_on_update controls whether a board setting notifies the ticket's people
-- (owner/resources) on UPDATED tickets. The configured recipient is only notified
-- on new tickets. Defaults true to preserve the prior "people always notified on
-- updates" behavior for existing rules.
ALTER TABLE notifier_rule ADD COLUMN notify_on_update BOOLEAN NOT NULL DEFAULT true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE notifier_rule DROP COLUMN notify_on_update;
-- +goose StatementEnd
