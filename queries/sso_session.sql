-- name: PutSSOSession :exec
INSERT INTO sso_session (key, user_id, created_on, last_seen_at, expires_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (key) DO UPDATE SET
    user_id = EXCLUDED.user_id,
    created_on = EXCLUDED.created_on,
    last_seen_at = EXCLUDED.last_seen_at,
    expires_at = EXCLUDED.expires_at;

-- name: GetSSOSession :one
SELECT * FROM sso_session
WHERE key = $1 LIMIT 1;

-- name: TouchSSOSession :exec
UPDATE sso_session
SET last_seen_at = $2, expires_at = $3
WHERE key = $1;

-- name: DeleteSSOSession :exec
DELETE FROM sso_session WHERE key = $1;

-- name: DeleteSSOSessionsByUser :exec
DELETE FROM sso_session WHERE user_id = $1;

-- name: DeleteExpiredSSOSessions :exec
DELETE FROM sso_session WHERE expires_at <= NOW();
