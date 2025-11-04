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
