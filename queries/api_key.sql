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
