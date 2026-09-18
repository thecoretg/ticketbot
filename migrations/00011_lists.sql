-- +goose Up
-- +goose StatementBegin
CREATE TABLE list (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    item_type TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX list_name_lower_idx ON list (LOWER(name));

CREATE TABLE list_item (
    list_id INT NOT NULL REFERENCES list(id) ON DELETE CASCADE,
    item_id INT NOT NULL,
    added_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (list_id, item_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS list_item;
DROP TABLE IF EXISTS list;
-- +goose StatementEnd
