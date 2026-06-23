-- +goose Up
-- +goose StatementBegin
DROP TABLE IF EXISTS transformer_run;
DROP TABLE IF EXISTS transformer_rule;

ALTER TABLE app_config RENAME COLUMN attempt_transform TO attempt_workflow;

CREATE TABLE IF NOT EXISTS workflow (
    id               SERIAL PRIMARY KEY,
    name             TEXT    NOT NULL,
    cw_board_id      INT     NOT NULL REFERENCES cw_board(id) ON DELETE CASCADE,
    on_ticket_action TEXT    NOT NULL DEFAULT 'both',
    conditions       JSONB   NOT NULL DEFAULT '{}'::jsonb, -- root ConditionGroup ({} = no conditions)
    actions          JSONB   NOT NULL DEFAULT '[]'::jsonb, -- ordered []Action
    priority         INT     NOT NULL DEFAULT 100,
    enabled          BOOLEAN NOT NULL DEFAULT true,
    created_on       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_workflow_board ON workflow(cw_board_id);

-- run-once markers for non-idempotent actions (add_note, send_message), keyed
-- per workflow per action since actions have no DB id under the JSONB design.
CREATE TABLE IF NOT EXISTS workflow_run (
    ticket_id    INT NOT NULL,
    workflow_id  INT NOT NULL REFERENCES workflow(id) ON DELETE CASCADE,
    action_index INT NOT NULL,
    created_on   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (ticket_id, workflow_id, action_index)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS workflow_run;
DROP TABLE IF EXISTS workflow;

ALTER TABLE app_config RENAME COLUMN attempt_workflow TO attempt_transform;

-- Recreate the transformer tables so the migration is reversible.
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
    updated_on  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    conditions  JSONB   NOT NULL DEFAULT '[]'::jsonb
);
CREATE INDEX idx_transformer_rule_board ON transformer_rule(cw_board_id);
CREATE TABLE IF NOT EXISTS transformer_run (
    ticket_id  INT NOT NULL,
    rule_id    INT NOT NULL REFERENCES transformer_rule(id) ON DELETE CASCADE,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (ticket_id, rule_id)
);
-- +goose StatementEnd
