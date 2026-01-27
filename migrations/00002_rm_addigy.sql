-- +goose Up
-- +goose StatementBegin
DROP TABLE IF EXISTS addigy_alert;
DROP TABLE IF EXISTS addigy_alert_config;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS addigy_alert_config (
    id INT PRIMARY KEY DEFAULT 1,
    cw_board_id INT NOT NULL REFERENCES cw_board(id),
    unattended_status_id INT NOT NULL REFERENCES cw_ticket_status(id),
    acknowledged_status_id INT NOT NULL REFERENCES cw_ticket_status(id),
    mute_1_day_status_id INT NOT NULL REFERENCES cw_ticket_status(id),
    mute_5_day_status_id INT NOT NULL REFERENCES cw_ticket_status(id),
    mute_10_day_status_id INT NOT NULL REFERENCES cw_ticket_status(id),
    mute_30_day_status_id INT NOT NULL REFERENCES cw_ticket_status(id),
    added_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF EXISTS addigy_alert (
    id TEXT PRIMARY KEY,
    ticket_id INT REFERENCES cw_ticket(id),
    level TEXT NOT NULL,
    category TEXT NOT NULL,
    name TEXT NOT NULL,
    fact_name TEXT NOT NULL,
    fact_identifier TEXT NOT NULL,
    fact_type TEXT NOT NULL,
    selector TEXT NOT NULL,
    status TEXT NOT NULL,
    value TEXT,
    muted BOOLEAN NOT NULL,
    remediation BOOLEAN NOT NULL,
    resolved_by_email TEXT,
    resolved_on TIMESTAMP,
    acknowledged_on TIMESTAMP,
    added_on TIMESTAMP NOT NULL
);
-- +goose StatementEnd
