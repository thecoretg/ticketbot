-- name: GetUserForAuth :one
SELECT id, email_address, password_hash, password_reset_required, totp_secret, totp_enabled, created_on, updated_on
FROM api_user
WHERE email_address = $1 LIMIT 1;

-- name: GetUserForAuthByID :one
SELECT id, email_address, password_hash, password_reset_required, totp_secret, totp_enabled, created_on, updated_on
FROM api_user
WHERE id = $1 LIMIT 1;

-- name: SetUserPassword :exec
UPDATE api_user
SET password_hash = $2, updated_on = NOW()
WHERE id = $1;

-- name: SetTOTPSecret :exec
UPDATE api_user SET totp_secret = $2, updated_on = NOW() WHERE id = $1;

-- name: SetTOTPEnabled :exec
UPDATE api_user SET totp_enabled = $2, updated_on = NOW() WHERE id = $1;

-- name: SetPasswordResetRequired :exec
UPDATE api_user
SET password_reset_required = $2, updated_on = NOW()
WHERE id = $1;
