-- name: DeleteAuthor :exec
DELETE FROM authors WHERE id = $1;