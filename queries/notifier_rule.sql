-- name: ListNotifierRules :many
SELECT * FROM notifier_rule
ORDER BY id;

-- name: ListNotifierRulesFull :many
SELECT
    r.id AS id,
    r.notify_enabled AS enabled,
    r.simulation_mode AS simulation_mode,
    r.notify_on_update AS notify_on_update,
    b.id AS board_id,
    b.name AS board_name,
    wr.id AS recipient_id,
    wr.name AS recipient_name,
    wr.type AS recipient_type
FROM notifier_rule AS r
JOIN webex_recipient AS wr
ON wr.id = r.webex_recipient_id
JOIN cw_board AS b
ON b.id = r.cw_board_id
ORDER BY r.id;

-- name: GetNotifierRule :one
SELECT * FROM notifier_rule
WHERE id = $1 LIMIT 1;

-- name: CheckNotifierExists :one
SELECT EXISTS (
    SELECT 1
    FROM notifier_rule
    WHERE id = $1
) AS exists;

-- name: CheckNotifierExistsByBoardAndRecipient :one
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
INSERT INTO notifier_rule(cw_board_id, webex_recipient_id, notify_enabled, simulation_mode, notify_on_update)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateNotifierRule :one
UPDATE notifier_rule
SET
    cw_board_id = $2,
    webex_recipient_id = $3,
    notify_enabled = $4,
    simulation_mode = $5,
    notify_on_update = $6
WHERE id = $1
RETURNING *;

-- name: DeleteNotifierRule :exec
DELETE FROM notifier_rule
WHERE id = $1;
