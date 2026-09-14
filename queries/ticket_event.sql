-- name: InsertTicketEvent :one
INSERT INTO ticket_event (ticket_id, run_id, kind, source, dry_run, payload, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListTicketEventsByTicket :many
SELECT * FROM ticket_event
WHERE ticket_id = $1
  AND (sqlc.narg('before_id')::bigint IS NULL OR id < sqlc.narg('before_id')::bigint)
ORDER BY id DESC
LIMIT sqlc.arg('lim');

-- name: ListRecentTicketEvents :many
SELECT * FROM ticket_event
WHERE (sqlc.narg('kind')::text IS NULL OR kind = sqlc.narg('kind')::text)
ORDER BY id DESC
LIMIT sqlc.arg('lim');
