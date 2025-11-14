-- name: GetTicket :one
SELECT * FROM cw_ticket
WHERE id = $1 LIMIT 1;

-- name: ListTickets :many
SELECT * FROM cw_ticket
ORDER BY id;


-- name: UpsertTicket :one
INSERT INTO cw_ticket
(id, summary, board_id, owner_id, company_id, contact_id, resources, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE SET
    summary = EXCLUDED.summary,
    board_id = EXCLUDED.board_id,
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

-- name: GetTicketNote :one
SELECT * FROM cw_ticket_note
WHERE id = $1 LIMIT 1;

-- name: ListAllTicketNotes :many
SELECT * FROM cw_ticket_note
ORDER BY id;

-- name: ListTicketNotesByTicket :many
SELECT * FROM cw_ticket_note
WHERE ticket_id = $1
ORDER BY id;

-- name: UpsertTicketNote :one
INSERT INTO cw_ticket_note
(id, ticket_id, member_id, contact_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
    ticket_id = EXCLUDED.ticket_id,
    member_id = EXCLUDED.member_id,
    contact_id = EXCLUDED.contact_id,
    updated_on = NOW()
RETURNING *;

-- name: SoftDeleteTicketNote :exec
UPDATE cw_ticket_note
SET deleted = TRUE
WHERE id = $1;

-- name: DeleteTicketNote :exec
DELETE FROM cw_ticket_note
WHERE id = $1;
