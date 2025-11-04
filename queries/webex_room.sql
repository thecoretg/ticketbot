-- name: GetWebexRoomIDByInternalID :one
SELECT webex_id FROM webex_room
WHERE id = $1;

-- name: GetWebexRoom :one
SELECT * FROM webex_room
WHERE id = $1;

-- name: GetWebexRoomByWebexID :one
SELECT * FROM webex_room
WHERE webex_id = $1;

-- name: ListWebexRooms :many
SELECT * FROM webex_room
ORDER BY id;

-- name: UpsertWebexRoom :one
INSERT INTO webex_room
(webex_id, name, type)
VALUES ($1, $2, $3)
ON CONFLICT (webex_id) DO UPDATE SET
    name = EXCLUDED.name,
    type = EXCLUDED.type,
    updated_on = NOW()
RETURNING *;

-- name: SoftDeleteWebexRoom :exec
UPDATE cw_board
SET
    deleted = TRUE,
    updated_on = NOW()
WHERE id = $1;

-- name: DeleteWebexRoom :exec
DELETE FROM webex_room
WHERE id = $1;

