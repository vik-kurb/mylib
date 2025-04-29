-- name: GetBook :one
SELECT title FROM books
WHERE id = $1;