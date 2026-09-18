-- name: GetUser :one
SELECT * FROM api_user
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM api_user
WHERE email_address = $1 LIMIT 1;

-- name: CheckUserExists :one
SELECT EXISTS(
    SELECT 1
    FROM api_user
    WHERE email_address = $1
) as exists;

-- name: ListUsers :many
SELECT * FROM api_user
ORDER BY email_address;

-- name: InsertUser :one
INSERT INTO api_user
(email_address, role)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserByEntraOID :one
SELECT * FROM api_user
WHERE entra_oid = $1 LIMIT 1;

-- name: GetUserByEmailFold :one
SELECT * FROM api_user
WHERE LOWER(email_address) = LOWER($1) LIMIT 1;

-- name: SetUserRole :exec
UPDATE api_user
SET role = $2, updated_on = NOW()
WHERE id = $1;

-- name: LinkUserEntra :exec
UPDATE api_user
SET entra_oid = $2, email_address = $3, updated_on = NOW()
WHERE id = $1;

-- name: UpdateUser :one
UPDATE api_user
SET
    email_address = $2,
    updated_on = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM api_user
WHERE id = $1;
