-- name: ListTransformerRules :many
SELECT * FROM transformer_rule
ORDER BY priority, id;

-- name: ListEnabledTransformerRules :many
SELECT * FROM transformer_rule
WHERE enabled = true
ORDER BY priority, id;

-- name: GetTransformerRule :one
SELECT * FROM transformer_rule
WHERE id = $1 LIMIT 1;

-- name: CheckTransformerRuleExists :one
SELECT EXISTS (
    SELECT 1
    FROM transformer_rule
    WHERE id = $1
) AS exists;

-- name: InsertTransformerRule :one
INSERT INTO transformer_rule(name, action, cw_board_id, config, conditions, apply_on, priority, enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateTransformerRule :one
UPDATE transformer_rule
SET
    name        = $2,
    action      = $3,
    cw_board_id = $4,
    config      = $5,
    conditions  = $6,
    apply_on    = $7,
    priority    = $8,
    enabled     = $9,
    updated_on  = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteTransformerRule :exec
DELETE FROM transformer_rule
WHERE id = $1;
