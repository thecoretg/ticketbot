-- name: GetScheduledJob :one
SELECT * FROM scheduled_job WHERE name = $1;

-- name: SaveScheduledJob :one
INSERT INTO scheduled_job (name, last_started_at, last_finished_at, last_error)
VALUES ($1, $2, $3, $4)
ON CONFLICT (name) DO UPDATE SET
    last_started_at = EXCLUDED.last_started_at,
    last_finished_at = EXCLUDED.last_finished_at,
    last_error = EXCLUDED.last_error
RETURNING *;
