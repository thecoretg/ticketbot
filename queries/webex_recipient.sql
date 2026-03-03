-- name: GetWebexRecipient :one
SELECT * FROM webex_recipient
WHERE id = $1;

-- name: GetWebexRecipientByWebexID :one
SELECT * FROM webex_recipient
WHERE webex_id = $1;

-- name: ListWebexRecipients :many
SELECT * FROM webex_recipient
ORDER BY name;

-- name: ListWebexRooms :many
SELECT * FROM webex_recipient
WHERE type = 'room';

-- name: ListWebexPeople :many
SELECT * FROM webex_recipient
WHERE type = 'person';

-- name: ListByEmail :many
SELECT * FROM webex_recipient
WHERE email = $1;

-- name: UpsertWebexRecipient :one
INSERT INTO webex_recipient
(webex_id, name, type, email, last_activity)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (webex_id) DO UPDATE SET
    name = EXCLUDED.name,
    type = EXCLUDED.type,
    email = EXCLUDED.email,
    last_activity = EXCLUDED.last_activity,
    updated_on = NOW()
RETURNING *;

-- name: DeleteWebexRecipient :exec
DELETE FROM webex_recipient
WHERE id = $1;

