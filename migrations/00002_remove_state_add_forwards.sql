-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS webex_user_forward (
    id SERIAL PRIMARY KEY,
    user_email TEXT NOT NULL,
    dest_email TEXT NOT NULL,
    start_date TIMESTAMP,
    end_date TIMESTAMP,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    user_keeps_copy BOOLEAN NOT NULL DEFAULT TRUE,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_email, dest_email, start_date, end_date),
    CHECK (start_date < end_date)
);

CREATE TABLE IF NOT EXISTS ticket_notification (
    id SERIAL PRIMARY KEY,
    ticket_id INT NOT NULL REFERENCES cw_ticket(id) ON DELETE CASCADE,
    ticket_note_id INT REFERENCES cw_ticket_note(id) ON DELETE CASCADE,
    webex_room_id INT REFERENCES webex_room(id) ON DELETE CASCADE,
    sent_to_email TEXT,
    sent BOOLEAN NOT NULL DEFAULT FALSE,
    skipped BOOLEAN NOT NULL DEFAULT TRUE,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE notifier_connection RENAME TO notifier_rule;

ALTER TABLE cw_ticket_note
ADD content TEXT,
DROP CONSTRAINT cw_ticket_note_ticket_id_fkey,
DROP COLUMN notified,
DROP COLUMN skipped_notify;

ALTER TABLE cw_ticket_note
ADD CONSTRAINT fk_ticket_note_ticket
    FOREIGN KEY (ticket_id)
    REFERENCES cw_ticket(id)
    ON DELETE CASCADE;

DROP TABLE IF EXISTS app_state;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE notifier_rule RENAME TO notifier_connection;

CREATE TABLE IF NOT EXISTS app_state (
    id INT PRIMARY KEY DEFAULT 1,
    syncing_tickets BOOLEAN NOT NULL DEFAULT false,
    syncing_webex_rooms BOOLEAN NOT NULL DEFAULT false
);

ALTER TABLE cw_ticket_note
DROP COLUMN content,
ADD notified BOOLEAN NOT NULL DEFAULT FALSE,
ADD skipped_notify BOOLEAN NOT NULL DEFAULT FALSE,
DROP CONSTRAINT fk_ticket_note_ticket;

ALTER TABLE cw_ticket_note
ADD CONSTRAINT cw_ticket_note_ticket_id_fkey
    FOREIGN KEY (ticket_id)
    REFERENCES cw_ticket(id);
-- +goose StatementEnd
