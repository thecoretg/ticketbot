-- name: GetCompany :one
SELECT * FROM cw_company
WHERE id = $1 LIMIT 1;

-- name: ListCompanies :many
SELECT * FROM cw_company
ORDER BY id;

-- name: UpsertCompany :one
INSERT INTO cw_company
(id, name)
VALUES ($1, $2)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    updated_on = NOW()
RETURNING *;

-- name: UpdateCompany :one
UPDATE cw_company
SET
    name = $2,
    updated_on = NOW()
WHERE id = $1
RETURNING *;

-- name: SoftDeleteCompany :exec
UPDATE cw_company
SET deleted = TRUE
WHERE id = $1;

-- name: DeleteCompany :exec
DELETE FROM cw_company
WHERE id = $1;
