-- name: GetTicketNotification :one
SELECT * FROM ticket_notification
WHERE id = $1;

-- name: ListTicketNotifications :many
SELECT * FROM ticket_notification
ORDER BY created_on;

-- name: ListTicketNotificationsByNoteID :many
SELECT * FROM ticket_notification
WHERE ticket_note_id = $1;

-- name: CheckNotificationsExistByTicketID :one
SELECT EXISTS (
    SELECT 1
    FROM ticket_notification
    WHERE ticket_id = $1
) AS exists;

-- name: CheckNotificationsExistByNote :one
SELECT EXISTS (
    SELECT 1
    FROM ticket_notification
    WHERE ticket_note_id = $1
) AS exists;

-- name: InsertTicketNotification :one
INSERT INTO ticket_notification
(ticket_id, ticket_note_id, webex_room_id, sent_to_email, sent, skipped)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: DeleteTicketNotification :exec
DELETE FROM ticket_notification
WHERE id = $1;
