-- name: PutSSOFlowState :exec
INSERT INTO sso_flow_state (key, state, nonce, verifier, next, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (key) DO UPDATE SET
    state = EXCLUDED.state,
    nonce = EXCLUDED.nonce,
    verifier = EXCLUDED.verifier,
    next = EXCLUDED.next,
    expires_at = EXCLUDED.expires_at;

-- name: TakeSSOFlowState :one
DELETE FROM sso_flow_state
WHERE key = $1
RETURNING *;

-- name: DeleteExpiredSSOFlowStates :exec
DELETE FROM sso_flow_state WHERE expires_at <= NOW();
