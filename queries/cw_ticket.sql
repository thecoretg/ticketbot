-- name: GetTicket :one
SELECT * FROM cw_ticket
WHERE id = $1 LIMIT 1;

-- name: ListTickets :many
SELECT * FROM cw_ticket
ORDER BY id;

-- name: CheckTicketExists :one
SELECT EXISTS (
    SELECT 1
    FROM cw_ticket
    WHERE id = $1
) AS exists;

-- name: UpsertTicket :one
INSERT INTO cw_ticket
(id, summary, board_id, status_id, owner_id, company_id, contact_id, resources, updated_by,
 priority_id, priority_name, type_id, type_name, subtype_id, subtype_name, item_id, item_name,
 closed_flag, latest_note_id, raw)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
ON CONFLICT (id) DO UPDATE SET
    summary = EXCLUDED.summary,
    board_id = EXCLUDED.board_id,
    status_id = EXCLUDED.status_id,
    owner_id = EXCLUDED.owner_id,
    company_id = EXCLUDED.company_id,
    contact_id = EXCLUDED.contact_id,
    resources = EXCLUDED.resources,
    updated_by = EXCLUDED.updated_by,
    priority_id = EXCLUDED.priority_id,
    priority_name = EXCLUDED.priority_name,
    type_id = EXCLUDED.type_id,
    type_name = EXCLUDED.type_name,
    subtype_id = EXCLUDED.subtype_id,
    subtype_name = EXCLUDED.subtype_name,
    item_id = EXCLUDED.item_id,
    item_name = EXCLUDED.item_name,
    closed_flag = EXCLUDED.closed_flag,
    latest_note_id = EXCLUDED.latest_note_id,
    raw = EXCLUDED.raw,
    deleted = FALSE,
    updated_on = NOW()
RETURNING *;

-- name: ListTicketsPaged :many
SELECT
    t.id, t.summary, t.board_id, t.status_id, t.owner_id, t.company_id, t.contact_id, t.resources,
    t.updated_by, t.updated_on, t.added_on, t.deleted, t.priority_id, t.priority_name, t.type_id,
    t.type_name, t.subtype_id, t.subtype_name, t.item_id, t.item_name, t.closed_flag, t.latest_note_id,
    b.name AS board_name,
    s.name AS status_name,
    c.name AS company_name,
    m.first_name AS owner_first_name,
    m.last_name AS owner_last_name
FROM cw_ticket t
JOIN cw_board b ON b.id = t.board_id
JOIN cw_ticket_status s ON s.id = t.status_id
JOIN cw_company c ON c.id = t.company_id
LEFT JOIN cw_member m ON m.id = t.owner_id
WHERE (sqlc.narg('board_id')::int IS NULL OR t.board_id = sqlc.narg('board_id')::int)
  AND (sqlc.narg('status_id')::int IS NULL OR t.status_id = sqlc.narg('status_id')::int)
  AND (sqlc.narg('closed')::bool IS NULL OR t.closed_flag = sqlc.narg('closed')::bool)
  AND (sqlc.narg('include_deleted')::bool IS TRUE OR t.deleted = FALSE)
  AND (sqlc.narg('search')::text IS NULL
       OR t.summary ILIKE '%' || sqlc.narg('search')::text || '%'
       OR t.id::text = sqlc.narg('search')::text)
ORDER BY t.updated_on DESC, t.id DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountTicketsPaged :one
SELECT COUNT(*)
FROM cw_ticket t
WHERE (sqlc.narg('board_id')::int IS NULL OR t.board_id = sqlc.narg('board_id')::int)
  AND (sqlc.narg('status_id')::int IS NULL OR t.status_id = sqlc.narg('status_id')::int)
  AND (sqlc.narg('closed')::bool IS NULL OR t.closed_flag = sqlc.narg('closed')::bool)
  AND (sqlc.narg('include_deleted')::bool IS TRUE OR t.deleted = FALSE)
  AND (sqlc.narg('search')::text IS NULL
       OR t.summary ILIKE '%' || sqlc.narg('search')::text || '%'
       OR t.id::text = sqlc.narg('search')::text);

-- name: SoftDeleteTicket :exec
UPDATE cw_ticket
SET
    deleted = TRUE,
    updated_on = NOW()
WHERE id = $1;

-- name: DeleteTicket :exec
DELETE FROM cw_ticket
WHERE id = $1;
