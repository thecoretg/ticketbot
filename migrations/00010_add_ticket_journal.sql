-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS ticket_journal (
    ticket_id    INT PRIMARY KEY,
    summary      TEXT        NOT NULL DEFAULT '',
    board_name   TEXT        NOT NULL DEFAULT '',
    company_name TEXT        NOT NULL DEFAULT '',
    contact_name TEXT        NOT NULL DEFAULT '',
    status_name  TEXT        NOT NULL DEFAULT '',
    owner_name   TEXT        NOT NULL DEFAULT '',
    last_trigger TEXT        NOT NULL DEFAULT '',
    last_outcome TEXT        NOT NULL DEFAULT '',
    had_error    BOOLEAN     NOT NULL DEFAULT false,
    first_seen   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_run     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    runs         JSONB       NOT NULL DEFAULT '[]'::jsonb
);

CREATE INDEX idx_ticket_journal_last_run ON ticket_journal(last_run DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ticket_journal;
-- +goose StatementEnd
