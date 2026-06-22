-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_config ADD COLUMN attempt_transform        BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE app_config ADD COLUMN cw_bot_member_identifier TEXT    NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS transformer_rule (
    id          SERIAL PRIMARY KEY,
    name        TEXT    NOT NULL,
    action      TEXT    NOT NULL,
    cw_board_id INT     REFERENCES cw_board(id) ON DELETE CASCADE,
    config      JSONB   NOT NULL DEFAULT '{}'::jsonb,
    apply_on    TEXT    NOT NULL DEFAULT 'both',
    priority    INT     NOT NULL DEFAULT 100,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_on  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_transformer_rule_board ON transformer_rule(cw_board_id);

-- run-once markers for non-idempotent actions (e.g. add_note)
CREATE TABLE IF NOT EXISTS transformer_run (
    ticket_id  INT NOT NULL,
    rule_id    INT NOT NULL REFERENCES transformer_rule(id) ON DELETE CASCADE,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (ticket_id, rule_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transformer_run;
DROP TABLE IF EXISTS transformer_rule;
ALTER TABLE app_config DROP COLUMN cw_bot_member_identifier;
ALTER TABLE app_config DROP COLUMN attempt_transform;
-- +goose StatementEnd
