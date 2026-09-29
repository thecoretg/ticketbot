-- name: GetWebhookCatchup :one
SELECT * FROM webhook_catchup WHERE id = 1;

-- name: SaveWebhookCatchup :one
INSERT INTO webhook_catchup (id, checked_through, last_run_at, last_queued, last_error)
VALUES (1, $1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
    checked_through = EXCLUDED.checked_through,
    last_run_at = EXCLUDED.last_run_at,
    last_queued = EXCLUDED.last_queued,
    last_error = EXCLUDED.last_error
RETURNING *;
