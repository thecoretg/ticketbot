-- name: GetAppState :one
SELECT * FROM app_state
WHERE id = 1;

-- name: InsertDefaultAppState :one
INSERT INTO app_state (id) VALUES (1)
ON CONFLICT (id) DO UPDATE SET id = EXCLUDED.id
RETURNING *;

-- name: UpsertAppState :one
INSERT INTO app_state(id, syncing_tickets, syncing_webex_rooms)
VALUES(1, $1, $2)
ON CONFLICT (id) DO UPDATE SET
    syncing_tickets = EXCLUDED.syncing_tickets,
    syncing_webex_rooms = EXCLUDED.syncing_webex_rooms
RETURNING *;
