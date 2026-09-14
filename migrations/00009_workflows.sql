-- +goose Up
-- +goose StatementBegin
CREATE TABLE workflow (
    id SERIAL PRIMARY KEY,
    board_id INT NOT NULL UNIQUE REFERENCES cw_board(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    dry_run BOOLEAN NOT NULL DEFAULT FALSE,
    rules JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Convert legacy notifier rules: one workflow per board that has any rule.
-- Rule A (create): notify each configured room, then ticket resources/owner.
-- Rule B (update): notify ticket resources/owner.
-- Both rules are enabled only if at least one legacy rule was enabled, mirroring the old
-- behaviour where a board with no enabled rules sent nothing at all.
INSERT INTO workflow (board_id, name, enabled, dry_run, rules)
SELECT
    b.id,
    b.name,
    TRUE,
    FALSE,
    jsonb_build_array(
        jsonb_build_object(
            'id', gen_random_uuid()::text,
            'name', 'New ticket: notify rooms, resources and owner',
            'enabled', EXISTS (SELECT 1 FROM notifier_rule r WHERE r.cw_board_id = b.id AND r.notify_enabled),
            'trigger', 'create',
            'condition', '',
            'stop_processing', FALSE,
            'actions',
                COALESCE((
                    SELECT jsonb_agg(
                        jsonb_build_object(
                            'kind', 'notify',
                            'enabled', r.notify_enabled,
                            'notify', jsonb_build_object('target', 'room', 'recipient_id', r.webex_recipient_id)
                        ) ORDER BY r.id)
                    FROM notifier_rule r WHERE r.cw_board_id = b.id
                ), '[]'::jsonb)
                || jsonb_build_array(
                    jsonb_build_object(
                        'kind', 'notify',
                        'enabled', TRUE,
                        'notify', jsonb_build_object('target', 'resources_owner')
                    )
                )
        ),
        jsonb_build_object(
            'id', gen_random_uuid()::text,
            'name', 'Ticket updated: notify resources and owner',
            'enabled', EXISTS (SELECT 1 FROM notifier_rule r WHERE r.cw_board_id = b.id AND r.notify_enabled),
            'trigger', 'update',
            'condition', '',
            'stop_processing', FALSE,
            'actions', jsonb_build_array(
                jsonb_build_object(
                    'kind', 'notify',
                    'enabled', TRUE,
                    'notify', jsonb_build_object('target', 'resources_owner')
                )
            )
        )
    )
FROM cw_board b
WHERE EXISTS (SELECT 1 FROM notifier_rule r WHERE r.cw_board_id = b.id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS workflow;
-- +goose StatementEnd
