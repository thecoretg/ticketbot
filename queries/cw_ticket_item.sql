-- name: GetTicketItem :one
SELECT * FROM cw_ticket_item
WHERE id = $1 LIMIT 1;

-- name: ListAllTicketItems :many
SELECT * FROM cw_ticket_item
ORDER BY id;

-- name: ListTicketItemsByBoard :many
SELECT * FROM cw_ticket_item
WHERE board_id = $1
ORDER BY id;

-- name: UpsertTicketItem :one
INSERT INTO cw_ticket_item (
    id,
    board_id,
    name,
    inactive
)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
    board_id = EXCLUDED.board_id,
    name = EXCLUDED.name,
    inactive = EXCLUDED.inactive,
    updated_on = NOW()
RETURNING *;

-- name: SoftDeleteTicketItem :exec
UPDATE cw_ticket_item
SET
    deleted = TRUE,
    updated_on = NOW()
WHERE id = $1;

-- name: DeleteTicketItem :exec
DELETE FROM cw_ticket_item
WHERE id = $1;
