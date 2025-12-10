-- +goose Up 
-- +goose StatementBegin
ALTER TABLE ticket_notification
    ALTER COLUMN recipient_id DROP NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ticket_notification
    ALTER COLUMN recipient_id SET NOT NULL;
-- +goose StatementEnd
