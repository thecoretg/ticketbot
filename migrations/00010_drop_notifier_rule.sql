-- +goose Up
-- +goose StatementBegin
DROP TABLE IF EXISTS notifier_rule;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS notifier_rule (
    id SERIAL PRIMARY KEY,
    cw_board_id INT NOT NULL REFERENCES cw_board(id) ON DELETE CASCADE,
    webex_recipient_id INT NOT NULL REFERENCES webex_recipient(id) ON DELETE CASCADE,
    notify_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (cw_board_id, webex_recipient_id)
);

-- Best-effort re-derivation from room notify actions in workflows.
INSERT INTO notifier_rule (cw_board_id, webex_recipient_id, notify_enabled)
SELECT DISTINCT ON (w.board_id, (a->'notify'->>'recipient_id')::int)
    w.board_id,
    (a->'notify'->>'recipient_id')::int,
    COALESCE((a->>'enabled')::bool, TRUE)
FROM workflow w,
     jsonb_array_elements(w.rules) r,
     jsonb_array_elements(r->'actions') a
WHERE a->>'kind' = 'notify'
  AND a->'notify'->>'target' = 'room'
  AND a->'notify'->>'recipient_id' IS NOT NULL
  AND EXISTS (SELECT 1 FROM webex_recipient wr WHERE wr.id = (a->'notify'->>'recipient_id')::int)
ON CONFLICT DO NOTHING;
-- +goose StatementEnd
