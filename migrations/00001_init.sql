-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS boards (
    id INT PRIMARY KEY,
    name TEXT NOT NULL,
    notify_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    webex_room_id TEXT
);

CREATE TABLE IF NOT EXISTS tickets (
    id INT PRIMARY KEY,
    summary TEXT NOT NULL,
    board_id INT REFERENCES boards(id) NOT NULL,
    owner_id INT,
    resources TEXT,
    updated_by TEXT,
    added_to_store TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS ticket_notes (
    id INT PRIMARY KEY,
    ticket_id INT REFERENCES tickets(id) NOT NULL,
    notified BOOLEAN NOT NULL DEFAULT FALSE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ticket_notes;
DROP TABLE IF EXISTS tickets;
DROP TABLE IF EXISTS boards;
-- +goose StatementEnd
