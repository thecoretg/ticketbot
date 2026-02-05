-- name: ListNotifierForwardsFull :many
SELECT 
    f.id AS id,
    f.enabled AS enabled,
    f.user_keeps_copy AS user_keeps_copy,
    f.start_date AS start_date,
    f.end_date AS end_date,
    src.id AS source_id,
    src.name AS source_name,
    src.type AS source_type,
    dst.id AS destination_id,
    dst.name AS destination_name,
    dst.type AS destination_type
FROM notifier_forward AS f
JOIN webex_recipient AS src
ON src.id = f.source_id
JOIN webex_recipient AS dst
ON dst.id = f.destination_id;

-- name: ListNotifierForwards :many
SELECT * FROM notifier_forward
ORDER BY id;

-- name: ListActiveNotifierForwards :many
SELECT
    f.id AS id,
    f.enabled AS enabled,
    f.user_keeps_copy AS user_keeps_copy,
    f.start_date AS start_date,
    f.end_date AS end_date,
    src.id AS source_id,
    src.name AS source_name,
    src.type AS source_type,
    dst.id AS destination_id,
    dst.name AS destination_name,
    dst.type AS destination_type
FROM notifier_forward AS f
JOIN webex_recipient AS src ON src.id = f.source_id
JOIN webex_recipient AS dst ON dst.id = f.destination_id
WHERE f.enabled = true
    AND (f.start_date IS NULL OR f.start_date <= NOW())
    AND (f.end_date IS NULL OR f.end_date > NOW())
ORDER BY f.id;

-- name: ListInactiveNotifierForwards :many
SELECT
    f.id AS id,
    f.enabled AS enabled,
    f.user_keeps_copy AS user_keeps_copy,
    f.start_date AS start_date,
    f.end_date AS end_date,
    src.id AS source_id,
    src.name AS source_name,
    src.type AS source_type,
    dst.id AS destination_id,
    dst.name AS destination_name,
    dst.type AS destination_type
FROM notifier_forward AS f
JOIN webex_recipient AS src ON src.id = f.source_id
JOIN webex_recipient AS dst ON dst.id = f.destination_id
WHERE f.enabled = false
    OR (f.start_date IS NOT NULL AND f.start_date > NOW())
    OR (f.end_date IS NOT NULL AND f.end_date <= NOW())
ORDER BY f.id;

-- name: ListActiveForwardsBySourceRecipient :many
SELECT
    f.id AS id,
    f.enabled AS enabled,
    f.user_keeps_copy AS user_keeps_copy,
    f.start_date AS start_date,
    f.end_date AS end_date,
    src.id AS source_id,
    src.name AS source_name,
    src.type AS source_type,
    dst.id AS destination_id,
    dst.name AS destination_name,
    dst.type AS destination_type
FROM notifier_forward AS f
JOIN webex_recipient AS src ON src.id = f.source_id
JOIN webex_recipient AS dst ON dst.id = f.destination_id
WHERE f.source_id = $1
    AND f.enabled = true
    AND (f.start_date IS NULL OR f.start_date <= NOW())
    AND (f.end_date IS NULL OR f.end_date > NOW())
ORDER BY f.id;

-- name: GetNotifierForward :one
SELECT * FROM notifier_forward
WHERE id = $1 LIMIT 1;

-- name: CheckNotifierForwardExists :one
SELECT EXISTS (
    SELECT 1
    FROM notifier_forward
    WHERE id = $1
) AS exists;

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
