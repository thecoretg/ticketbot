-- +goose Up
-- +goose StatementBegin
ALTER TABLE ticket_notes
ADD COLUMN contact TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ticket_notes
DROP COLUMN contact;
-- +goose StatementEnd
