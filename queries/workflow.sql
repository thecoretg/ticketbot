-- name: ListWorkflows :many
SELECT * FROM workflow
ORDER BY priority, id;

-- name: ListEnabledWorkflows :many
SELECT * FROM workflow
WHERE enabled = true
ORDER BY priority, id;

-- name: GetWorkflow :one
SELECT * FROM workflow
WHERE id = $1 LIMIT 1;

-- name: CheckWorkflowExists :one
SELECT EXISTS (
    SELECT 1
    FROM workflow
    WHERE id = $1
) AS exists;

-- name: InsertWorkflow :one
INSERT INTO workflow(name, cw_board_id, on_ticket_action, conditions, actions, priority, enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateWorkflow :one
UPDATE workflow
SET
    name             = $2,
    cw_board_id      = $3,
    on_ticket_action = $4,
    conditions       = $5,
    actions          = $6,
    priority         = $7,
    enabled          = $8,
    updated_on       = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteWorkflow :exec
DELETE FROM workflow
WHERE id = $1;
