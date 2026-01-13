-- name: GetTicketStatus :one
SELECT * FROM cw_ticket_status
WHERE id = $1 LIMIT 1;

-- name: ListAllTicketStatuses :many
SELECT * FROM cw_ticket_status
ORDER BY id;

-- name: ListTicketStatusesByBoard :many
SELECT * FROM cw_ticket_status
WHERE board_id = $1
ORDER BY id;

-- name: UpsertTicketStatus :one
INSERT INTO cw_ticket_status (
    id,
    board_id,
    name,
    default_status,
    display_on_board,
    inactive,
    closed
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO UPDATE SET
    board_id = EXCLUDED.board_id,
    name = EXCLUDED.name,
    default_status = EXCLUDED.default_status,
    display_on_board = EXCLUDED.display_on_board,
    inactive = EXCLUDED.inactive,
    closed = EXCLUDED.closed,
    updated_on = NOW()
RETURNING *;

-- name: SoftDeleteTicketStatus :exec
UPDATE cw_ticket_status
SET
    deleted = TRUE,
    updated_on = NOW()
WHERE id = $1;

-- name: DeleteTicketStatus :exec
DELETE FROM cw_ticket_status
WHERE id = $1;
