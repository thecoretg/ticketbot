-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS app_config (
    id INT PRIMARY KEY DEFAULT 1,
    attempt_notify BOOLEAN NOT NULL DEFAULT false,
    max_message_length INT NOT NULL DEFAULT 300,
    max_concurrent_syncs INT NOT NULL DEFAULT 5
);

CREATE TABLE IF NOT EXISTS api_user (
    id SERIAL PRIMARY KEY,
    email_address TEXT UNIQUE NOT NULL,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS api_key (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES api_user(id) ON DELETE CASCADE,
    key_hash BYTEA NOT NULL,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, key_hash)
);

CREATE TABLE IF NOT EXISTS webex_recipient (
    id SERIAL PRIMARY KEY,
    webex_id TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    email TEXT,
    type TEXT NOT NULL,
    last_activity TIMESTAMP NOT NULL,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cw_board (
    id INT PRIMARY KEY,
    name TEXT NOT NULL,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    added_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS notifier_rule (
    id SERIAL PRIMARY KEY,
    cw_board_id INT NOT NULL REFERENCES cw_board(id) ON DELETE CASCADE,
    webex_recipient_id INT NOT NULL REFERENCES webex_recipient(id) ON DELETE CASCADE,
    notify_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (cw_board_id, webex_recipient_id)
);

CREATE TABLE IF NOT EXISTS notifier_forward (
    id SERIAL PRIMARY KEY,
    source_id INT NOT NULL REFERENCES webex_recipient(id),
    destination_id INT NOT NULL REFERENCES webex_recipient(id),
    start_date TIMESTAMP,
    end_date TIMESTAMP,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    user_keeps_copy BOOLEAN NOT NULL DEFAULT TRUE,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (source_id, destination_id, start_date, end_date),
    CHECK(source_id <> destination_id),
    CHECK (start_date < end_date)
);

CREATE TABLE IF NOT EXISTS cw_company (
    id INT PRIMARY KEY,
    name TEXT NOT NULL,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    added_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cw_contact (
    id INT PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT,
    company_id INT REFERENCES cw_company(id),
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    added_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cw_member (
    id INT PRIMARY KEY,
    identifier TEXT NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    primary_email TEXT NOT NULL,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    added_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cw_ticket (
    id INT PRIMARY KEY,
    summary TEXT NOT NULL,
    board_id INT REFERENCES cw_board(id) NOT NULL,
    owner_id INT REFERENCES cw_member(id),
    company_id INT REFERENCES cw_company(id) NOT NULL,
    contact_id INT REFERENCES cw_contact(id),
    resources TEXT,
    updated_by TEXT,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    added_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cw_ticket_note (
    id INT PRIMARY KEY,
    ticket_id INT NOT NULL REFERENCES cw_ticket(id) ON DELETE CASCADE,
    member_id INT REFERENCES cw_member(id),
    contact_id INT REFERENCES cw_contact(id),
    content TEXT,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    added_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ticket_notification (
    id SERIAL PRIMARY KEY,
    ticket_id INT NOT NULL REFERENCES cw_ticket(id) ON DELETE CASCADE,
    ticket_note_id INT REFERENCES cw_ticket_note(id) ON DELETE CASCADE,
    recipient_id INT NOT NULL REFERENCES webex_recipient(id) ON DELETE CASCADE,
    sent BOOLEAN NOT NULL DEFAULT FALSE,
    skipped BOOLEAN NOT NULL DEFAULT TRUE,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ticket_notification;
DROP TABLE IF EXISTS notifier_forward;
DROP TABLE IF EXISTS notifier_rule;
DROP TABLE IF EXISTS cw_ticket_note;
DROP TABLE IF EXISTS cw_ticket;
DROP TABLE IF EXISTS cw_board;
DROP TABLE IF EXISTS cw_member;
DROP TABLE IF EXISTS cw_company;
DROP TABLE IF EXISTS cw_contact;
DROP TABLE IF EXISTS webex_recipient;
DROP TABLE IF EXISTS api_key;
DROP TABLE IF EXISTS api_user;
DROP TABLE IF EXISTS app_config;
-- +goose StatementEnd
