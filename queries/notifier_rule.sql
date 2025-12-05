-- name: ListNotifierRules :many
SELECT * FROM notifier_rule
ORDER BY id;

-- name: GetNotifierRule :one
SELECT * FROM notifier_rule
WHERE id = $1 LIMIT 1;

-- name: CheckNotifierExists :one
SELECT EXISTS (
    SELECT 1
    FROM notifier_rule
    WHERE cw_board_id = $1 AND webex_recipient_id = $2
) AS exists;

-- name: ListNotifierRulesByBoard :many
SELECT * FROM notifier_rule
WHERE cw_board_id = $1
ORDER BY id;

-- name: ListNotifierRulesByRecipient :many
SELECT * FROM notifier_rule
WHERE webex_recipient_id = $1
ORDER BY id;

-- name: InsertNotifierRule :one
INSERT INTO notifier_rule(cw_board_id, webex_recipient_id, notify_enabled)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateNotifierRule :one
UPDATE notifier_rule
SET
    cw_board_id = $2,
    webex_recipient_id = $3,
    notify_enabled = $4
WHERE id = $1
RETURNING *;

-- name: SoftDeleteNotifierRule :exec
UPDATE notifier_rule
SET
    deleted = TRUE,
    updated_on = NOW()
WHERE id = $1;

-- name: DeleteNotifierRule :exec
DELETE FROM notifier_rule
WHERE id = $1;
