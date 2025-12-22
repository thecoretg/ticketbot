-- name: GetAppConfig :one
SELECT * FROM app_config
WHERE id = 1;

-- name: InsertDefaultAppConfig :one
INSERT INTO app_config (id) VALUES (1)
ON CONFLICT (id) DO UPDATE SET id = EXCLUDED.id
RETURNING *;

-- name: UpsertAppConfig :one
INSERT INTO app_config(id, attempt_notify, max_message_length, max_concurrent_syncs, skip_launch_syncs)
VALUES(1, $1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
    attempt_notify = EXCLUDED.attempt_notify,
    max_message_length = EXCLUDED.max_message_length,
    max_concurrent_syncs = EXCLUDED.max_concurrent_syncs,
    skip_launch_syncs = EXCLUDED.skip_launch_syncs
RETURNING *;

