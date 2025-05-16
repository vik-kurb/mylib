-- name: GetBooks :many
SELECT id, title FROM books
WHERE id IN (SELECT UNNEST($1::UUID[]));