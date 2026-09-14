-- +goose Up
-- +goose StatementBegin
ALTER TABLE cw_ticket
    ADD COLUMN raw JSONB,
    ADD COLUMN priority_id INT,
    ADD COLUMN priority_name TEXT,
    ADD COLUMN type_id INT,
    ADD COLUMN type_name TEXT,
    ADD COLUMN subtype_id INT,
    ADD COLUMN subtype_name TEXT,
    ADD COLUMN item_id INT,
    ADD COLUMN item_name TEXT,
    ADD COLUMN closed_flag BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN latest_note_id INT;

UPDATE cw_ticket t
SET latest_note_id = (SELECT MAX(n.id) FROM cw_ticket_note n WHERE n.ticket_id = t.id);

UPDATE cw_ticket t
SET closed_flag = s.closed
FROM cw_ticket_status s
WHERE s.id = t.status_id;

CREATE TABLE ticket_event (
    id BIGSERIAL PRIMARY KEY,
    ticket_id INT NOT NULL REFERENCES cw_ticket(id) ON DELETE CASCADE,
    run_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    source TEXT NOT NULL,
    dry_run BOOLEAN NOT NULL DEFAULT FALSE,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ticket_event_ticket ON ticket_event(ticket_id, id DESC);
CREATE INDEX idx_ticket_event_run ON ticket_event(run_id);

ALTER TABLE app_config
    ADD COLUMN master_dry_run BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN cw_api_member_identifier TEXT NOT NULL DEFAULT '';

UPDATE app_config SET master_dry_run = NOT attempt_notify;

ALTER TABLE app_config DROP COLUMN attempt_notify;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_config ADD COLUMN attempt_notify BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE app_config SET attempt_notify = NOT master_dry_run;

ALTER TABLE app_config
    DROP COLUMN cw_api_member_identifier,
    DROP COLUMN master_dry_run;

DROP TABLE IF EXISTS ticket_event;

ALTER TABLE cw_ticket
    DROP COLUMN latest_note_id,
    DROP COLUMN closed_flag,
    DROP COLUMN item_name,
    DROP COLUMN item_id,
    DROP COLUMN subtype_name,
    DROP COLUMN subtype_id,
    DROP COLUMN type_name,
    DROP COLUMN type_id,
    DROP COLUMN priority_name,
    DROP COLUMN priority_id,
    DROP COLUMN raw;
-- +goose StatementEnd
