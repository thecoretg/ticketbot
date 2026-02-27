-- name: CreateSession :one
INSERT INTO session (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetSessionByTokenHash :one
SELECT * FROM session
WHERE token_hash = $1 AND expires_at > NOW()
LIMIT 1;

-- name: DeleteSession :exec
DELETE FROM session WHERE id = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM session WHERE expires_at < NOW();
