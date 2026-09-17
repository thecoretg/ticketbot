-- name: GetContact :one
SELECT * FROM cw_contact
WHERE id = $1 LIMIT 1;

-- name: ListContacts :many
SELECT * FROM cw_contact
ORDER BY id;

-- name: UpsertContact :one
INSERT INTO cw_contact
(id, first_name, last_name, company_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    company_id = EXCLUDED.company_id,
    updated_on = NOW()
RETURNING *;

-- name: SoftDeleteContact :exec
UPDATE cw_contact
SET
    deleted = TRUE,
    updated_on = NOW()
WHERE id = $1;

-- name: DeleteContact :exec
DELETE FROM cw_contact
WHERE id = $1;

-- name: SearchContacts :many
SELECT * FROM cw_contact
WHERE deleted = FALSE
  AND (sqlc.narg('search')::text IS NULL
       OR first_name ILIKE '%' || sqlc.narg('search')::text || '%'
       OR last_name ILIKE '%' || sqlc.narg('search')::text || '%'
       OR (first_name || ' ' || COALESCE(last_name, '')) ILIKE '%' || sqlc.narg('search')::text || '%')
  AND (sqlc.narg('company_id')::int IS NULL OR company_id = sqlc.narg('company_id')::int)
  AND (sqlc.narg('ids')::int[] IS NULL OR id = ANY(sqlc.narg('ids')::int[]))
ORDER BY first_name, last_name
LIMIT sqlc.arg('lim');
