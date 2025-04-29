-- name: DeleteBook :exec
DELETE FROM books WHERE id = $1;