-- name: CreateTOTPPending :one
INSERT INTO totp_pending (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTOTPPendingByTokenHash :one
SELECT * FROM totp_pending
WHERE token_hash = $1 AND expires_at > NOW()
LIMIT 1;

-- name: DeleteTOTPPending :exec
DELETE FROM totp_pending WHERE id = $1;

-- name: DeleteExpiredTOTPPendings :exec
DELETE FROM totp_pending WHERE expires_at < NOW();
