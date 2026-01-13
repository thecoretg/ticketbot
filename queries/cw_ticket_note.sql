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
(id, ticket_id, content, member_id, contact_id)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET
    ticket_id = EXCLUDED.ticket_id,
    content = EXCLUDED.content,
    member_id = EXCLUDED.member_id,
    contact_id = EXCLUDED.contact_id,
    updated_on = NOW()
RETURNING *;

-- name: SoftDeleteTicketNote :exec
UPDATE cw_ticket_note
SET
    deleted = TRUE,
    updated_on = NOW()
WHERE id = $1;

-- name: DeleteTicketNote :exec
DELETE FROM cw_ticket_note
WHERE id = $1;
