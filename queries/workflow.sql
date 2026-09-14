-- name: ListWorkflows :many
SELECT w.*, b.name AS board_name
FROM workflow w
JOIN cw_board b ON b.id = w.board_id
ORDER BY b.name, w.id;

-- name: GetWorkflow :one
SELECT w.*, b.name AS board_name
FROM workflow w
JOIN cw_board b ON b.id = w.board_id
WHERE w.id = $1 LIMIT 1;

-- name: GetWorkflowByBoard :one
SELECT w.*, b.name AS board_name
FROM workflow w
JOIN cw_board b ON b.id = w.board_id
WHERE w.board_id = $1 LIMIT 1;

-- name: WorkflowExistsForBoard :one
SELECT EXISTS (
    SELECT 1 FROM workflow WHERE board_id = $1
) AS exists;

-- name: InsertWorkflow :one
INSERT INTO workflow (board_id, name, enabled, dry_run, rules)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateWorkflow :one
UPDATE workflow
SET
    name = $2,
    enabled = $3,
    dry_run = $4,
    rules = $5,
    updated_on = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteWorkflow :exec
DELETE FROM workflow
WHERE id = $1;
