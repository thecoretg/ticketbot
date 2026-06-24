-- name: GetTicketType :one
SELECT * FROM cw_ticket_type
WHERE id = $1 LIMIT 1;

-- name: ListAllTicketTypes :many
SELECT * FROM cw_ticket_type
ORDER BY id;

-- name: ListTicketTypesByBoard :many
SELECT * FROM cw_ticket_type
WHERE board_id = $1
ORDER BY id;

-- name: UpsertTicketType :one
INSERT INTO cw_ticket_type (
    id,
    board_id,
    name,
    category,
    default_flag,
    inactive
)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO UPDATE SET
    board_id = EXCLUDED.board_id,
    name = EXCLUDED.name,
    category = EXCLUDED.category,
    default_flag = EXCLUDED.default_flag,
    inactive = EXCLUDED.inactive,
    updated_on = NOW()
RETURNING *;

-- name: SoftDeleteTicketType :exec
UPDATE cw_ticket_type
SET
    deleted = TRUE,
    updated_on = NOW()
WHERE id = $1;

-- name: DeleteTicketType :exec
DELETE FROM cw_ticket_type
WHERE id = $1;
