-- name: InsertWorkflowRun :exec
INSERT INTO workflow_run (run_id, ticket_id, board_id, workflow_id, workflow_name, event, source, dry_run,
    started_at, duration_ms, steps, actions, writes, notif_sent, notif_would_send, notif_none, errors, outcome)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
ON CONFLICT (run_id) DO NOTHING;

-- name: GetWorkflowRun :one
SELECT * FROM workflow_run WHERE run_id = $1;

-- name: ListWorkflowRuns :many
SELECT * FROM workflow_run
WHERE (sqlc.narg('board_id')::int IS NULL OR board_id = sqlc.narg('board_id')::int)
  AND (sqlc.narg('outcome')::text IS NULL OR outcome = sqlc.narg('outcome')::text)
  AND (sqlc.narg('ticket_id')::int IS NULL OR ticket_id = sqlc.narg('ticket_id')::int)
  AND (sqlc.narg('from_at')::timestamptz IS NULL OR started_at >= sqlc.narg('from_at')::timestamptz)
  AND (sqlc.narg('to_at')::timestamptz IS NULL OR started_at < sqlc.narg('to_at')::timestamptz)
  AND (sqlc.narg('before_at')::timestamptz IS NULL OR started_at < sqlc.narg('before_at')::timestamptz)
ORDER BY started_at DESC, run_id DESC
LIMIT sqlc.arg('lim');

-- name: DeleteWorkflowRunsBefore :execrows
DELETE FROM workflow_run WHERE started_at < $1;
