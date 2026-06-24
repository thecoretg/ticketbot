-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS cw_ticket_type (
    id           INT PRIMARY KEY,
    board_id     INT NOT NULL REFERENCES cw_board(id),
    name         TEXT NOT NULL,
    category     TEXT NOT NULL DEFAULT '',
    default_flag BOOLEAN NOT NULL DEFAULT false,
    inactive     BOOLEAN NOT NULL DEFAULT false,
    added_on     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted      BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS cw_ticket_subtype (
    id                   INT PRIMARY KEY,
    board_id             INT NOT NULL REFERENCES cw_board(id),
    name                 TEXT NOT NULL,
    inactive             BOOLEAN NOT NULL DEFAULT false,
    type_association_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    added_on             TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted              BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS cw_ticket_item (
    id         INT PRIMARY KEY,
    board_id   INT NOT NULL REFERENCES cw_board(id),
    name       TEXT NOT NULL,
    inactive   BOOLEAN NOT NULL DEFAULT false,
    added_on   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted    BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_cw_ticket_type_board ON cw_ticket_type(board_id);
CREATE INDEX idx_cw_ticket_subtype_board ON cw_ticket_subtype(board_id);
CREATE INDEX idx_cw_ticket_item_board ON cw_ticket_item(board_id);

ALTER TABLE cw_ticket
    ADD COLUMN type_id    INT REFERENCES cw_ticket_type(id),
    ADD COLUMN subtype_id INT REFERENCES cw_ticket_subtype(id),
    ADD COLUMN item_id    INT REFERENCES cw_ticket_item(id);

ALTER TABLE ticket_journal
    ADD COLUMN type_name    TEXT NOT NULL DEFAULT '',
    ADD COLUMN subtype_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN item_name    TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ticket_journal
    DROP COLUMN IF EXISTS type_name,
    DROP COLUMN IF EXISTS subtype_name,
    DROP COLUMN IF EXISTS item_name;

ALTER TABLE cw_ticket
    DROP COLUMN IF EXISTS type_id,
    DROP COLUMN IF EXISTS subtype_id,
    DROP COLUMN IF EXISTS item_id;

DROP TABLE IF EXISTS cw_ticket_item;
DROP TABLE IF EXISTS cw_ticket_subtype;
DROP TABLE IF EXISTS cw_ticket_type;
-- +goose StatementEnd
