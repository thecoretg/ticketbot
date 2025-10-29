-- name: GetAppState :one
SELECT * FROM app_state
WHERE id = 1;

-- name: UpsertAppState :one
INSERT INTO app_state(id, syncing_tickets, syncing_webex_rooms)
VALUES(1, $1, $2)
ON CONFLICT (id) DO UPDATE SET
    syncing_tickets = EXCLUDED.syncing_tickets,
    syncing_webex_rooms = EXCLUDED.syncing_webex_rooms
RETURNING *;

-- name: GetAppConfig :one
SELECT * FROM app_config
WHERE id = 1;

-- name: UpsertAppConfig :one
INSERT INTO app_config(id, debug, attempt_notify, max_message_length, max_concurrent_syncs)
VALUES(1, $1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
    debug = EXCLUDED.debug,
    attempt_notify = EXCLUDED.attempt_notify,
    max_message_length = EXCLUDED.max_message_length,
    max_concurrent_syncs = EXCLUDED.max_concurrent_syncs
RETURNING *;

-- name: GetUser :one
SELECT * FROM api_user
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM api_user
WHERE email_address = $1 LIMIT 1;

-- name: ListUsers :many
SELECT * FROM api_user
ORDER BY email_address;

-- name: InsertUser :one
INSERT INTO api_user
(email_address)
VALUES ($1)
RETURNING *;

-- name: UpdateUser :one
UPDATE api_user
SET
    email_address = $1,
    updated_on = NOW()
WHERE id = $1
RETURNING *;

-- name: SoftDeleteUser :one
UPDATE api_user
SET
    deleted = true,
    updated_on = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM api_user
WHERE id = $1;

-- name: GetAPILKey :one
SELECT * FROM api_key
WHERE id = $1;

-- name: ListAPIKeys :many
SELECT * FROM api_key
ORDER BY created_on;

-- name: InsertAPIKey :one
INSERT INTO api_key
(user_id, key_hash)
VALUES ($1, $2)
RETURNING *;

-- name: SoftDeleteAPIKey :one
UPDATE api_key
SET
    delete = true,
    updated_on = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteAPIKey :exec
DELETE FROM api_key
WHERE id = $1;

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

-- name: InsertWebexRoom :one
INSERT INTO webex_room
(webex_id, name, type)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateWebexRoom :one
UPDATE webex_room
SET
    name = $2,
    type = $3,
    updated_on = NOW()
WHERE id = $1
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

-- name: GetBoard :one
SELECT * FROM cw_board
WHERE id = $1 LIMIT 1;

-- name: ListBoards :many
SELECT * FROM cw_board
ORDER BY id;

-- name: InsertBoard :one
INSERT INTO cw_board
(id, name)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateBoard :one
UPDATE cw_board
SET
    name = $2,
    updated_on = NOW()
WHERE id = $1
RETURNING *;

-- name: SoftDeleteBoard :exec
UPDATE cw_board
SET
    deleted = TRUE,
    updated_on = NOW()
WHERE id = $1;

-- name: DeleteBoard :exec
DELETE FROM cw_board
WHERE id = $1;

-- name: ListNotifierConnections :many
SELECT * FROM notifier_connection
ORDER BY cw_board_id;

-- name: ListRoomsByBoard :many
SELECT w.* FROM webex_room w
JOIN notifier_connection nc ON nc.webex_room_id = w.id
WHERE nc.cw_board_id = $1 AND nc.notify_enabled = TRUE;

-- name: ListBoardsByRoom :many
SELECT b.* FROM cw_board b
JOIN notifier_connection nc ON nc.cw_board_id = b.id
WHERE nc.webex_room_id = $1;

-- name: InsertNotifierConnection :one
INSERT INTO notifier_connection(cw_board_id, webex_room_id)
VALUES ($1, $2)
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
WHERE cw_board_id = $1 AND webex_room_id = $2;

-- name: DeleteNotifierConnection :exec
DELETE FROM notifier_connection
WHERE cw_board_id = $1 AND webex_room_id = $2;

-- name: GetTicket :one
SELECT * FROM cw_ticket
WHERE id = $1 LIMIT 1;

-- name: ListTickets :many
SELECT * FROM cw_ticket
ORDER BY id;


-- name: InsertTicket :one
INSERT INTO cw_ticket
(id, summary, board_id, owner_id, company_id, contact_id, resources, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateTicket :one
UPDATE cw_ticket
SET
    summary = $2,
    board_id = $3,
    owner_id = $4,
    company_id = $5,
    contact_id = $6,
    resources = $7,
    updated_by = $8,
    updated_on = NOW()
WHERE id = $1
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

-- name: InsertTicketNote :one
INSERT INTO cw_ticket_note
(id, ticket_id, member_id, contact_id, notified, skipped_notify)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateTicketNote :one
UPDATE cw_ticket_note
SET
    ticket_id = $2,
    member_id = $3,
    contact_id = $4,
    notified = $5,
    skipped_notify = $6,
    updated_on = NOW()
WHERE id = $1
RETURNING *;

-- name: SetNoteNotified :one
UPDATE cw_ticket_note
SET
    notified = $2
WHERE id = $1
RETURNING *;

-- name: SetNoteSkippedNotify :one
UPDATE cw_ticket_note
SET
    skipped_notify = $2
WHERE id = $1
RETURNING *;

-- name: DeleteTicketNote :exec
DELETE FROM cw_ticket_note
WHERE id = $1;

-- COMPANY QUERIES

-- name: GetCompany :one
SELECT * FROM cw_company
WHERE id = $1 LIMIT 1;

-- name: ListCompanies :many
SELECT * FROM cw_company
ORDER BY id;

-- name: InsertCompany :one
INSERT INTO cw_company
(id, name)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateCompany :one
UPDATE cw_company
SET
    name = $2,
    updated_on = NOW()
WHERE id = $1
RETURNING *;

-- name: SoftDeleteCompany :exec
UPDATE cw_company
SET deleted = TRUE
WHERE id = $1;

-- name: DeleteCompany :exec
DELETE FROM cw_company
WHERE id = $1;

-- CONTACT QUERIES

-- name: GetContact :one
SELECT * FROM cw_contact
WHERE id = $1 LIMIT 1;

-- name: ListContacts :many
SELECT * FROM cw_contact
ORDER BY id;

-- name: InsertContact :one
INSERT INTO cw_contact
(id, first_name, last_name, company_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateContact :one
UPDATE cw_contact
SET
    first_name = $2,
    last_name = $3,
    company_id = $4,
    updated_on = NOW()
WHERE id = $1
RETURNING *;

-- name: SoftDeleteContact :exec
UPDATE cw_contact
SET deleted = TRUE
WHERE id = $1;

-- name: DeleteContact :exec
DELETE FROM cw_contact
WHERE id = $1;

-- MEMBER QUERIES

-- name: GetMember :one
SELECT * FROM cw_member
WHERE id = $1 LIMIT 1;

-- name: GetMemberByIdentifier :one
SELECT * FROM cw_member
WHERE identifier = $1 LIMIT 1;

-- name: ListMembers :many
SELECT * FROM cw_member
ORDER BY id;

-- name: InsertMember :one
INSERT INTO cw_member
(id, identifier, first_name, last_name, primary_email)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateMember :one
UPDATE cw_member
SET
    identifier = $2,
    first_name = $3,
    last_name = $4,
    primary_email = $5,
    updated_on = NOW()
WHERE id = $1
RETURNING *;

-- name: SoftDeleteMember :exec
UPDATE cw_member
SET deleted = TRUE
WHERE id = $1;

-- name: DeleteMember :exec
DELETE FROM cw_member
WHERE id = $1;

-- name: SoftDeleteTicketNote :exec
UPDATE cw_ticket_note
SET deleted = TRUE
WHERE id = $1;