-- name: GetMember :one
SELECT * FROM cw_member
WHERE id = $1 LIMIT 1;

-- name: GetMemberByIdentifier :one
SELECT * FROM cw_member
WHERE identifier = $1 LIMIT 1;

-- name: ListMembers :many
SELECT * FROM cw_member
ORDER BY id;

-- name: UpsertMember :one
INSERT INTO cw_member
(id, identifier, first_name, last_name, primary_email)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET
    identifier = EXCLUDED.identifier,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    primary_email = EXCLUDED.primary_email,
    updated_on = NOW()
RETURNING *;

-- name: SoftDeleteMember :exec
UPDATE cw_member
SET deleted = TRUE
WHERE id = $1;

-- name: DeleteMember :exec
DELETE FROM cw_member
WHERE id = $1;
