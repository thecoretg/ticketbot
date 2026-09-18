-- name: ListLists :many
SELECT l.*, (SELECT COUNT(*) FROM list_item li WHERE li.list_id = l.id) AS item_count
FROM list l
ORDER BY LOWER(l.name), l.id;

-- name: GetList :one
SELECT l.*, (SELECT COUNT(*) FROM list_item li WHERE li.list_id = l.id) AS item_count
FROM list l
WHERE l.id = $1 LIMIT 1;

-- name: InsertList :one
INSERT INTO list (name, item_type, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateList :one
UPDATE list
SET name = $2, description = $3, updated_on = NOW()
WHERE id = $1
RETURNING *;

-- name: TouchList :exec
UPDATE list SET updated_on = NOW() WHERE id = $1;

-- name: DeleteList :execrows
DELETE FROM list WHERE id = $1;

-- name: ListListItems :many
SELECT * FROM list_item
WHERE list_id = $1
ORDER BY added_on, item_id;

-- name: InsertListItem :execrows
INSERT INTO list_item (list_id, item_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: DeleteListItem :execrows
DELETE FROM list_item
WHERE list_id = $1 AND item_id = $2;

-- name: ListAllListItems :many
SELECT list_id, item_id FROM list_item
ORDER BY list_id, item_id;
