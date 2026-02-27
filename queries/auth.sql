-- name: GetUserForAuth :one
SELECT id, email_address, password_hash, created_on, updated_on
FROM api_user
WHERE email_address = $1 LIMIT 1;

-- name: SetUserPassword :exec
UPDATE api_user
SET password_hash = $2, updated_on = NOW()
WHERE id = $1;
