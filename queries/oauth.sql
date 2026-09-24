-- name: CreateOAuthClient :one
INSERT INTO oauth_client (id, name, redirect_uris, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetOAuthClient :one
SELECT * FROM oauth_client WHERE id = $1;

-- name: UpsertOAuthGrant :one
INSERT INTO oauth_grant (user_id, client_id, scopes)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, client_id) DO UPDATE SET scopes = EXCLUDED.scopes
RETURNING *;

-- name: ClearOAuthClientExpiry :exec
UPDATE oauth_client SET expires_at = NULL WHERE id = $1;

-- name: GetOAuthGrant :one
SELECT * FROM oauth_grant WHERE id = $1;

-- name: ListOAuthGrantsByUser :many
SELECT g.*, c.name AS client_name
FROM oauth_grant g
JOIN oauth_client c ON c.id = g.client_id
WHERE g.user_id = $1
ORDER BY g.created_on DESC;

-- name: DeleteOAuthGrant :execrows
DELETE FROM oauth_grant WHERE id = $1;

-- name: TouchOAuthGrant :exec
UPDATE oauth_grant SET last_used_at = NOW() WHERE id = $1;

-- name: CreateOAuthCode :exec
INSERT INTO oauth_code (code_hash, grant_id, redirect_uri, code_challenge, expires_at)
VALUES ($1, $2, $3, $4, $5);

-- name: TakeOAuthCode :one
DELETE FROM oauth_code WHERE code_hash = $1
RETURNING *;

-- name: CreateOAuthToken :one
INSERT INTO oauth_token (grant_id, kind, token_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetOAuthToken :one
SELECT * FROM oauth_token WHERE token_hash = $1;

-- name: MarkOAuthTokenUsed :exec
UPDATE oauth_token SET used_at = NOW() WHERE id = $1;

-- name: ResolveOAuthAccessToken :one
SELECT t.grant_id, t.expires_at, g.user_id, g.scopes, c.name AS client_name
FROM oauth_token t
JOIN oauth_grant g ON g.id = t.grant_id
JOIN oauth_client c ON c.id = g.client_id
WHERE t.token_hash = $1 AND t.kind = 'access';

-- name: DeleteExpiredOAuthCodes :execrows
DELETE FROM oauth_code WHERE expires_at < $1;

-- name: DeleteExpiredOAuthTokens :execrows
DELETE FROM oauth_token WHERE expires_at < $1;

-- name: DeleteExpiredOAuthClients :execrows
-- Abandoned registrations, and clients whose every grant has since been revoked.
DELETE FROM oauth_client c
WHERE c.expires_at < sqlc.arg(now)::timestamptz
   OR (c.created_on < sqlc.arg(now)::timestamptz - interval '10 minutes'
       AND NOT EXISTS (SELECT 1 FROM oauth_grant g WHERE g.client_id = c.id));
