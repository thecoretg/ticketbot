-- +goose Up
-- +goose StatementBegin
ALTER TABLE ticket_notes
ADD COLUMN member TEXT;

ALTER TABLE ticket_notes
DROP CONSTRAINT ticket_notes_ticket_id_fkey;

ALTER TABLE ticket_notes
ADD CONSTRAINT ticket_notes_ticket_id_fkey
FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE CASCADE;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
ALTER TABLE ticket_notes
DROP CONSTRAINT ticket_notes_ticket_id_fkey;

ALTER TABLE ticket_notes
DROP COLUMN member;

ALTER TABLE ticket_notes
ADD CONSTRAINT ticket_notes_ticket_id_fkey
FOREIGN KEY (ticket_id) REFERENCES tickets(id);
-- +goose StatementEnd
