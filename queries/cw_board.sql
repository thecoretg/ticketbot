-- name: GetBoard :one
SELECT * FROM cw_board
WHERE id = $1 LIMIT 1;

-- name: ListBoards :many
SELECT * FROM cw_board
ORDER BY id;

-- name: UpsertBoard :one
INSERT INTO cw_board
(id, name)
VALUES ($1, $2)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    updated_on = NOW()
RETURNING *;

-- name: SoftDeleteBoard :exec
UPDATE cw_board
SET
    deleted = TRUE,
    updated_on = NOW()
WHERE id = $1;

-- name: DeleteBoard :exec
DELETE FROM cw_board
WHERE id = $1;

