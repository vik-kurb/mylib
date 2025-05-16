-- name: GetAuthorsByIDs :many
SELECT id, full_name FROM authors
WHERE id IN (SELECT UNNEST($1::UUID[]));