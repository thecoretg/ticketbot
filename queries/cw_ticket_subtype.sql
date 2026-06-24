-- name: GetTicketSubType :one
SELECT * FROM cw_ticket_subtype
WHERE id = $1 LIMIT 1;

-- name: ListAllTicketSubTypes :many
SELECT * FROM cw_ticket_subtype
ORDER BY id;

-- name: ListTicketSubTypesByBoard :many
SELECT * FROM cw_ticket_subtype
WHERE board_id = $1
ORDER BY id;

-- name: UpsertTicketSubType :one
INSERT INTO cw_ticket_subtype (
    id,
    board_id,
    name,
    inactive,
    type_association_ids
)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET
    board_id = EXCLUDED.board_id,
    name = EXCLUDED.name,
    inactive = EXCLUDED.inactive,
    type_association_ids = EXCLUDED.type_association_ids,
    updated_on = NOW()
RETURNING *;

-- name: SoftDeleteTicketSubType :exec
UPDATE cw_ticket_subtype
SET
    deleted = TRUE,
    updated_on = NOW()
WHERE id = $1;

-- name: DeleteTicketSubType :exec
DELETE FROM cw_ticket_subtype
WHERE id = $1;
