-- name: CheckWorkflowRunExists :one
SELECT EXISTS (
    SELECT 1
    FROM workflow_run
    WHERE ticket_id = $1 AND workflow_id = $2 AND action_index = $3
) AS exists;

-- name: InsertWorkflowRun :exec
INSERT INTO workflow_run(ticket_id, workflow_id, action_index)
VALUES ($1, $2, $3)
ON CONFLICT (ticket_id, workflow_id, action_index) DO NOTHING;
