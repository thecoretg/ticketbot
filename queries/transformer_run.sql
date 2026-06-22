-- name: CheckTransformerRunExists :one
SELECT EXISTS (
    SELECT 1
    FROM transformer_run
    WHERE ticket_id = $1 AND rule_id = $2
) AS exists;

-- name: InsertTransformerRun :exec
INSERT INTO transformer_run(ticket_id, rule_id)
VALUES ($1, $2)
ON CONFLICT (ticket_id, rule_id) DO NOTHING;
