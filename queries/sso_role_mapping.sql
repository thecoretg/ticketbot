-- name: ListSSORoleMappings :many
SELECT * FROM sso_role_mapping
ORDER BY entra_role;

-- name: UpsertSSORoleMapping :one
INSERT INTO sso_role_mapping (entra_role, role)
VALUES ($1, $2)
ON CONFLICT (entra_role) DO UPDATE SET role = EXCLUDED.role
RETURNING *;

-- name: DeleteSSORoleMapping :exec
DELETE FROM sso_role_mapping WHERE id = $1;
