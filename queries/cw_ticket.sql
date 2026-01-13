-- name: GetTicket :one
SELECT * FROM cw_ticket
WHERE id = $1 LIMIT 1;

-- name: ListTickets :many
SELECT * FROM cw_ticket
ORDER BY id;

-- name: CheckTicketExists :one
SELECT EXISTS (
    SELECT 1
    FROM cw_ticket
    WHERE id = $1
) AS exists;

-- name: UpsertTicket :one
INSERT INTO cw_ticket
(id, summary, board_id, status_id, owner_id, company_id, contact_id, resources, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE SET
    summary = EXCLUDED.summary,
    board_id = EXCLUDED.board_id,
    status_id = EXCLUDED.status_id,
    owner_id = EXCLUDED.owner_id,
    company_id = EXCLUDED.company_id,
    contact_id = EXCLUDED.contact_id,
    resources = EXCLUDED.resources,
    updated_by = EXCLUDED.updated_by,
    updated_on = NOW()
RETURNING *;

-- name: SoftDeleteTicket :exec
UPDATE cw_ticket
SET
    deleted = TRUE,
    updated_on = NOW()
WHERE id = $1;

-- name: DeleteTicket :exec
DELETE FROM cw_ticket
WHERE id = $1;

