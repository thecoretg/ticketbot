-- name: InsertTOTPRecoveryCode :exec
INSERT INTO totp_recovery_code (user_id, code_hash)
VALUES ($1, $2);

-- name: GetUnusedTOTPRecoveryCodeByHash :one
SELECT * FROM totp_recovery_code
WHERE user_id = $1 AND code_hash = $2 AND used = FALSE
LIMIT 1;

-- name: MarkTOTPRecoveryCodeUsed :exec
UPDATE totp_recovery_code SET used = TRUE WHERE id = $1;

-- name: DeleteTOTPRecoveryCodes :exec
DELETE FROM totp_recovery_code WHERE user_id = $1;
