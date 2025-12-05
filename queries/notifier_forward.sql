-- name: ListNotifierForwards :many
SELECT * FROM notifier_forward
ORDER BY id;

-- name: GetNotifierForward :one
SELECT * FROM notifier_forward
WHERE id = $1 LIMIT 1;

-- name: ListNotifierForwardsBySourceRecipientID :many
SELECT * FROM notifier_forward
WHERE source_id = $1
ORDER BY id;

-- name: InsertNotifierForward :one
INSERT INTO notifier_forward (
    source_id, destination_id, start_date, end_date, enabled, user_keeps_copy
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: DeleteNotifierForward :exec
DELETE FROM notifier_forward
WHERE id = $1;
