-- name: GetBoard :one
SELECT * FROM boards
WHERE id = $1 LIMIT 1;

-- name: ListBoards :many
SELECT * FROM boards
ORDER BY id;

-- name: InsertBoard :one
INSERT INTO boards
(id, name, notify_enabled, webex_room_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateBoard :one
UPDATE boards
SET
    name = $2,
    notify_enabled = $3,
    webex_room_id = $4
WHERE id = $1
RETURNING *;

-- name: DeleteBoard :exec
DELETE FROM boards
WHERE id = $1;

-- name: GetTicket :one
SELECT * FROM tickets
WHERE id = $1 LIMIT 1;

-- name: ListTickets :many
SELECT * FROM tickets
ORDER BY id;

-- name: InsertTicket :one
INSERT INTO tickets
(id, summary, board_id, owner_id, resources, updated_by, added_to_store)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateTicket :one
UPDATE tickets
SET
    summary = $2,
    board_id = $3,
    owner_id = $4,
    resources = $5,
    updated_by = $6,
    added_to_store = $7
WHERE id = $1
RETURNING *;

-- name: DeleteTicket :exec
DELETE FROM tickets
WHERE id = $1;

-- name: GetTicketNote :one
SELECT * FROM ticket_notes
WHERE id = $1;

-- name: ListAllTicketNotes :many
SELECT * FROM ticket_notes
ORDER BY id;

-- name: ListTicketNotesByTicket :many
SELECT * FROM ticket_notes
WHERE ticket_id = $1
ORDER BY id;

-- name: InsertTicketNote :one
INSERT INTO ticket_notes
(id, ticket_id, notified, member, contact)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateTicketNote :one
UPDATE ticket_notes
SET
    ticket_id = $2,
    notified = $3,
    member = $4,
    contact = $5
WHERE id = $1
RETURNING *;

-- name: SetNoteNotified :one
UPDATE ticket_notes
SET
    notified = $2
WHERE id = $1
RETURNING *;

-- name: DeleteTicketNote :exec
DELETE FROM ticket_notes
WHERE id = $1;