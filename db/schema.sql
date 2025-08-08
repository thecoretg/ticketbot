CREATE TABLE IF NOT EXISTS boards (
    id INT PRIMARY KEY,
    name TEXT NOT NULL,
    notify_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    webex_room_id TEXT
);

CREATE TABLE IF NOT EXISTS tickets (
    id INT PRIMARY KEY,
    summary TEXT NOT NULL,
    board_id INT REFERENCES boards(id),
    owner_id INT,
    resources TEXT,
    updated_by TEXT,
    added_to_store TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS ticket_notes (
    id INT PRIMARY KEY,
    ticket_id INT REFERENCES tickets(id),
    notified BOOLEAN NOT NULL DEFAULT FALSE
);
