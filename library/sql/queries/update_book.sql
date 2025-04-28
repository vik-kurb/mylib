-- name: UpdateBook :one
UPDATE books SET title = $2, updated_at = NOW()
WHERE id = $1
RETURNING 1;