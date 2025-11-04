-- name: ListNotifierConnections :many
SELECT sqlc.embed(nc), sqlc.embed(cb), sqlc.embed(wr)
FROM notifier_connection nc
JOIN cw_board AS cb ON cb.id = nc.cw_board_id
JOIN webex_room AS wr ON wr.id = nc.webex_room_id;

-- name: GetNotifierConnection :one
SELECT sqlc.embed(nc), sqlc.embed(cb), sqlc.embed(wr)
FROM notifier_connection nc
JOIN cw_board AS cb ON cb.id = nc.cw_board_id
JOIN webex_room AS wr ON wr.id = nc.webex_room_id
WHERE nc.id = $1;

-- name: ListNotifierConnectionsByBoard :many
SELECT sqlc.embed(nc), sqlc.embed(cb), sqlc.embed(wr)
FROM notifier_connection nc
JOIN cw_board AS cb ON cb.id = nc.cw_board_id
JOIN webex_room AS wr ON wr.id = nc.webex_room_id
WHERE cb.id = $1;

-- name: ListNotifierConnectionsByRoom :many
SELECT sqlc.embed(nc), sqlc.embed(cb), sqlc.embed(wr)
FROM notifier_connection nc
JOIN cw_board AS cb ON cb.id = nc.cw_board_id
JOIN webex_room AS wr ON wr.id = nc.webex_room_id
WHERE wr.id = $1;

-- name: InsertNotifierConnection :one
INSERT INTO notifier_connection(cw_board_id, webex_room_id, notify_enabled)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateNotifierConnection :one
UPDATE notifier_connection
SET notify_enabled = $3
WHERE cw_board_id = $1 AND webex_room_id = $2
RETURNING *;

-- name: SoftDeleteNotifierConnection :exec
UPDATE notifier_connection
SET
    deleted = TRUE,
    updated_on = NOW()
WHERE id = $1;

-- name: DeleteNotifierConnection :exec
DELETE FROM notifier_connection
WHERE id = $1;
