-- +goose Up
-- +goose StatementBegin
ALTER TABLE transformer_rule ADD COLUMN conditions JSONB NOT NULL DEFAULT '[]'::jsonb;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transformer_rule DROP COLUMN conditions;
-- +goose StatementEnd
